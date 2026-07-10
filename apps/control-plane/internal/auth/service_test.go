package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"deltauptime/packages/database/postgres"
)

func TestServiceRegisterCreatesOwnerMembership(t *testing.T) {
	repo := newFakeRepo()
	service := newTestService(t, repo)

	result, err := service.Register(context.Background(), RegisterInput{
		Email:            "user@example.com",
		Password:         "supersecret1",
		DisplayName:      "User",
		OrganizationName: "Acme",
		OrganizationSlug: "acme",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if result.AccessToken == "" || result.RefreshToken == "" {
		t.Fatal("expected both access and refresh tokens")
	}
	if len(result.Organizations) != 1 || result.Organizations[0].Role != "owner" {
		t.Fatalf("unexpected organizations result: %+v", result.Organizations)
	}
	if len(repo.memberships) != 1 || repo.memberships[0].Role != "owner" {
		t.Fatalf("owner membership not created: %+v", repo.memberships)
	}
}

func TestServiceLoginRejectsWrongPassword(t *testing.T) {
	repo := newFakeRepo()
	service := newTestService(t, repo)
	if _, err := service.Register(context.Background(), RegisterInput{
		Email:            "user@example.com",
		Password:         "supersecret1",
		DisplayName:      "User",
		OrganizationName: "Acme",
		OrganizationSlug: "acme",
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	_, err := service.Login(context.Background(), LoginInput{
		Email:    "user@example.com",
		Password: "wrong-password",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login error = %v, want ErrInvalidCredentials", err)
	}
}

func TestServiceRefreshRotatesToken(t *testing.T) {
	repo := newFakeRepo()
	service := newTestService(t, repo)

	initial, err := service.Register(context.Background(), RegisterInput{
		Email:            "user@example.com",
		Password:         "supersecret1",
		DisplayName:      "User",
		OrganizationName: "Acme",
		OrganizationSlug: "acme",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	refreshed, err := service.Refresh(context.Background(), RefreshInput{
		RefreshToken: initial.RefreshToken,
	})
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if refreshed.RefreshToken == initial.RefreshToken {
		t.Fatal("refresh rotation returned the same refresh token")
	}

	oldHash := service.tokens.HashRefreshToken(initial.RefreshToken)
	oldSession, err := repo.GetAuthRefreshTokenByTokenHash(context.Background(), oldHash)
	if err != nil {
		t.Fatalf("GetAuthRefreshTokenByTokenHash(old): %v", err)
	}
	if !oldSession.UsedAt.Valid || !oldSession.ReplacedByTokenID.Valid {
		t.Fatalf("old session was not rotated: %+v", oldSession)
	}
}

func TestServiceLoginRequiresTOTPWhenEnabled(t *testing.T) {
	repo := newFakeRepo()
	service := newTestService(t, repo)

	result, err := service.Register(context.Background(), RegisterInput{
		Email:            "user@example.com",
		Password:         "supersecret1",
		DisplayName:      "User",
		OrganizationName: "Acme",
		OrganizationSlug: "acme",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	principal, err := service.AuthenticateAccessToken(context.Background(), result.AccessToken)
	if err != nil {
		t.Fatalf("AuthenticateAccessToken: %v", err)
	}

	setup, err := service.SetupTOTP(context.Background(), principal)
	if err != nil {
		t.Fatalf("SetupTOTP: %v", err)
	}
	code := service.totp.codeAt(setup.Secret, service.timeNow().UTC().Unix()/totpPeriod)
	if _, err := service.EnableTOTP(context.Background(), principal, code); err != nil {
		t.Fatalf("EnableTOTP: %v", err)
	}

	_, err = service.Login(context.Background(), LoginInput{
		Email:    "user@example.com",
		Password: "supersecret1",
	})
	if !errors.Is(err, ErrTOTPRequired) {
		t.Fatalf("Login error = %v, want ErrTOTPRequired", err)
	}
}

func newTestService(t *testing.T, repo Repository) *Service {
	t.Helper()

	tokens, err := NewTokenManager("access-secret", "refresh-secret", 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("NewTokenManager: %v", err)
	}
	secrets, err := NewSecretsManager("test-secrets-master-key")
	if err != nil {
		t.Fatalf("NewSecretsManager: %v", err)
	}

	return &Service{
		repo:       repo,
		hasher:     PasswordHasher{},
		tokens:     tokens,
		secrets:    secrets,
		totp:       NewTOTPManager("DeltaUptime"),
		timeNow:    time.Now,
		totpIssuer: "DeltaUptime",
	}
}

type fakeRepo struct {
	usersByEmail   map[string]postgres.User
	usersByID      map[string]postgres.User
	orgsBySlug     map[string]postgres.Organization
	orgsByUserID   map[string][]postgres.ListOrganizationsByUserRow
	refreshByHash  map[string]postgres.AuthRefreshToken
	refreshByID    map[string]string
	apiKeysByID    map[string]postgres.ApiKey
	apiKeysByOrg   map[string][]postgres.ApiKey
	apiKeysByPref  map[string]postgres.ApiKey
	totpByUserID   map[string]postgres.UserTotpCredential
	auditLogsByOrg map[string][]postgres.AuditLog
	memberships    []postgres.Membership
	nextUUIDByte   byte
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		usersByEmail:   map[string]postgres.User{},
		usersByID:      map[string]postgres.User{},
		orgsBySlug:     map[string]postgres.Organization{},
		orgsByUserID:   map[string][]postgres.ListOrganizationsByUserRow{},
		refreshByHash:  map[string]postgres.AuthRefreshToken{},
		refreshByID:    map[string]string{},
		apiKeysByID:    map[string]postgres.ApiKey{},
		apiKeysByOrg:   map[string][]postgres.ApiKey{},
		apiKeysByPref:  map[string]postgres.ApiKey{},
		totpByUserID:   map[string]postgres.UserTotpCredential{},
		auditLogsByOrg: map[string][]postgres.AuditLog{},
	}
}

func (r *fakeRepo) Begin(context.Context) (Tx, error) { return r, nil }
func (r *fakeRepo) Commit(context.Context) error      { return nil }
func (r *fakeRepo) Rollback(context.Context) error    { return nil }

func (r *fakeRepo) CreateAPIKey(_ context.Context, arg postgres.CreateAPIKeyParams) (postgres.ApiKey, error) {
	key := postgres.ApiKey{
		ID:              r.newUUID(),
		OrganizationID:  arg.OrganizationID,
		CreatedByUserID: arg.CreatedByUserID,
		Name:            arg.Name,
		KeyPrefix:       arg.KeyPrefix,
		KeyHash:         arg.KeyHash,
		Scopes:          append([]string(nil), arg.Scopes...),
		CreatedAt:       pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	r.apiKeysByID[key.ID.String()] = key
	r.apiKeysByPref[key.KeyPrefix] = key
	r.apiKeysByOrg[key.OrganizationID.String()] = append(r.apiKeysByOrg[key.OrganizationID.String()], key)
	return key, nil
}

func (r *fakeRepo) CreateAuditLog(_ context.Context, arg postgres.CreateAuditLogParams) (postgres.AuditLog, error) {
	log := postgres.AuditLog{
		ID:             r.newUUID(),
		OrganizationID: arg.OrganizationID,
		ActorType:      arg.ActorType,
		ActorUserID:    arg.ActorUserID,
		ActorApiKeyID:  arg.ActorApiKeyID,
		Action:         arg.Action,
		TargetType:     arg.TargetType,
		TargetID:       arg.TargetID,
		Metadata:       append([]byte(nil), arg.Metadata...),
		OccurredAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	r.auditLogsByOrg[arg.OrganizationID.String()] = append(r.auditLogsByOrg[arg.OrganizationID.String()], log)
	return log, nil
}

func (r *fakeRepo) CreateAuthRefreshToken(_ context.Context, arg postgres.CreateAuthRefreshTokenParams) (postgres.AuthRefreshToken, error) {
	session := postgres.AuthRefreshToken{
		ID:        r.newUUID(),
		UserID:    arg.UserID,
		TokenHash: arg.TokenHash,
		ExpiresAt: arg.ExpiresAt,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	r.refreshByHash[arg.TokenHash] = session
	r.refreshByID[session.ID.String()] = arg.TokenHash
	return session, nil
}

func (r *fakeRepo) CreateMembership(_ context.Context, arg postgres.CreateMembershipParams) (postgres.Membership, error) {
	for _, existing := range r.memberships {
		if existing.OrganizationID == arg.OrganizationID && existing.UserID == arg.UserID {
			return postgres.Membership{}, errors.New("duplicate membership")
		}
	}
	membership := postgres.Membership{
		ID:             r.newUUID(),
		OrganizationID: arg.OrganizationID,
		UserID:         arg.UserID,
		Role:           arg.Role,
		CreatedAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	r.memberships = append(r.memberships, membership)
	user := r.usersByID[arg.UserID.String()]
	org := r.orgsBySlug[r.slugByOrgID(arg.OrganizationID.String())]
	r.orgsByUserID[arg.UserID.String()] = append(r.orgsByUserID[arg.UserID.String()], postgres.ListOrganizationsByUserRow{
		ID:        org.ID,
		Slug:      org.Slug,
		Name:      org.Name,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
		Role:      membership.Role,
	})
	_ = user
	return membership, nil
}

func (r *fakeRepo) CreateOrganization(_ context.Context, arg postgres.CreateOrganizationParams) (postgres.Organization, error) {
	if _, exists := r.orgsBySlug[arg.Slug]; exists {
		return postgres.Organization{}, errors.New("duplicate organization slug")
	}

	org := postgres.Organization{
		ID:        r.newUUID(),
		Slug:      arg.Slug,
		Name:      arg.Name,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	r.orgsBySlug[arg.Slug] = org
	return org, nil
}

func (r *fakeRepo) CreateUser(_ context.Context, arg postgres.CreateUserParams) (postgres.User, error) {
	if _, exists := r.usersByEmail[arg.Email]; exists {
		return postgres.User{}, errors.New("duplicate email")
	}

	user := postgres.User{
		ID:            r.newUUID(),
		Email:         arg.Email,
		PasswordHash:  arg.PasswordHash,
		DisplayName:   arg.DisplayName,
		IsSystemAdmin: arg.IsSystemAdmin,
		CreatedAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	r.usersByEmail[arg.Email] = user
	r.usersByID[user.ID.String()] = user
	return user, nil
}

func (r *fakeRepo) GetAuthRefreshTokenByTokenHash(_ context.Context, tokenHash string) (postgres.AuthRefreshToken, error) {
	session, ok := r.refreshByHash[tokenHash]
	if !ok {
		return postgres.AuthRefreshToken{}, pgx.ErrNoRows
	}
	return session, nil
}

func (r *fakeRepo) GetMembership(_ context.Context, arg postgres.GetMembershipParams) (postgres.Membership, error) {
	for _, membership := range r.memberships {
		if membership.OrganizationID == arg.OrganizationID && membership.UserID == arg.UserID {
			return membership, nil
		}
	}
	return postgres.Membership{}, pgx.ErrNoRows
}

func (r *fakeRepo) GetOrganizationByID(_ context.Context, id pgtype.UUID) (postgres.Organization, error) {
	for _, org := range r.orgsBySlug {
		if org.ID == id {
			return org, nil
		}
	}
	return postgres.Organization{}, pgx.ErrNoRows
}

func (r *fakeRepo) GetOrganizationBySlug(_ context.Context, slug string) (postgres.Organization, error) {
	org, ok := r.orgsBySlug[slug]
	if !ok {
		return postgres.Organization{}, pgx.ErrNoRows
	}
	return org, nil
}

func (r *fakeRepo) GetUserByEmail(_ context.Context, email string) (postgres.User, error) {
	user, ok := r.usersByEmail[email]
	if !ok {
		return postgres.User{}, pgx.ErrNoRows
	}
	return user, nil
}

func (r *fakeRepo) GetUserByID(_ context.Context, id pgtype.UUID) (postgres.User, error) {
	user, ok := r.usersByID[id.String()]
	if !ok {
		return postgres.User{}, pgx.ErrNoRows
	}
	return user, nil
}

func (r *fakeRepo) GetUserTOTPCredentialByUserID(_ context.Context, userID pgtype.UUID) (postgres.UserTotpCredential, error) {
	credential, ok := r.totpByUserID[userID.String()]
	if !ok {
		return postgres.UserTotpCredential{}, pgx.ErrNoRows
	}
	return credential, nil
}

func (r *fakeRepo) ListAPIKeysByOrganization(_ context.Context, organizationID pgtype.UUID) ([]postgres.ApiKey, error) {
	return append([]postgres.ApiKey(nil), r.apiKeysByOrg[organizationID.String()]...), nil
}

func (r *fakeRepo) ListAuditLogsByOrganization(_ context.Context, arg postgres.ListAuditLogsByOrganizationParams) ([]postgres.AuditLog, error) {
	return append([]postgres.AuditLog(nil), r.auditLogsByOrg[arg.OrganizationID.String()]...), nil
}

func (r *fakeRepo) ListOrganizationsByUser(_ context.Context, userID pgtype.UUID) ([]postgres.ListOrganizationsByUserRow, error) {
	return append([]postgres.ListOrganizationsByUserRow(nil), r.orgsByUserID[userID.String()]...), nil
}

func (r *fakeRepo) ListUsersByOrganization(_ context.Context, organizationID pgtype.UUID) ([]postgres.ListUsersByOrganizationRow, error) {
	rows := make([]postgres.ListUsersByOrganizationRow, 0)
	for _, membership := range r.memberships {
		if membership.OrganizationID != organizationID {
			continue
		}
		user := r.usersByID[membership.UserID.String()]
		rows = append(rows, postgres.ListUsersByOrganizationRow{
			ID:            user.ID,
			Email:         user.Email,
			PasswordHash:  user.PasswordHash,
			DisplayName:   user.DisplayName,
			IsSystemAdmin: user.IsSystemAdmin,
			CreatedAt:     user.CreatedAt,
			UpdatedAt:     user.UpdatedAt,
			Role:          membership.Role,
		})
	}
	return rows, nil
}

func (r *fakeRepo) RevokeAPIKey(_ context.Context, id pgtype.UUID) (postgres.ApiKey, error) {
	key, ok := r.apiKeysByID[id.String()]
	if !ok {
		return postgres.ApiKey{}, pgx.ErrNoRows
	}
	key.RevokedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	r.apiKeysByID[id.String()] = key
	r.apiKeysByPref[key.KeyPrefix] = key
	return key, nil
}

func (r *fakeRepo) RotateAuthRefreshToken(_ context.Context, arg postgres.RotateAuthRefreshTokenParams) (postgres.AuthRefreshToken, error) {
	tokenHash, ok := r.refreshByID[arg.ID.String()]
	if !ok {
		return postgres.AuthRefreshToken{}, pgx.ErrNoRows
	}

	session := r.refreshByHash[tokenHash]
	session.UsedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	session.ReplacedByTokenID = arg.ReplacedByTokenID
	r.refreshByHash[tokenHash] = session
	return session, nil
}

func (r *fakeRepo) TouchAPIKeyLastUsed(_ context.Context, id pgtype.UUID) (postgres.ApiKey, error) {
	key, ok := r.apiKeysByID[id.String()]
	if !ok {
		return postgres.ApiKey{}, pgx.ErrNoRows
	}
	key.LastUsedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	r.apiKeysByID[id.String()] = key
	r.apiKeysByPref[key.KeyPrefix] = key
	return key, nil
}

func (r *fakeRepo) UpdateMembershipRole(_ context.Context, arg postgres.UpdateMembershipRoleParams) (postgres.Membership, error) {
	for i, membership := range r.memberships {
		if membership.OrganizationID == arg.OrganizationID && membership.UserID == arg.UserID {
			membership.Role = arg.Role
			r.memberships[i] = membership
			return membership, nil
		}
	}
	return postgres.Membership{}, pgx.ErrNoRows
}

func (r *fakeRepo) UpsertUserTOTPCredential(_ context.Context, arg postgres.UpsertUserTOTPCredentialParams) (postgres.UserTotpCredential, error) {
	now := time.Now()
	credential := postgres.UserTotpCredential{
		UserID:           arg.UserID,
		SecretCiphertext: append([]byte(nil), arg.SecretCiphertext...),
		SecretNonce:      append([]byte(nil), arg.SecretNonce...),
		CreatedAt:        pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt:        pgtype.Timestamptz{Time: now, Valid: true},
	}
	r.totpByUserID[arg.UserID.String()] = credential
	return credential, nil
}

func (r *fakeRepo) DeleteUserTOTPCredential(_ context.Context, userID pgtype.UUID) error {
	delete(r.totpByUserID, userID.String())
	return nil
}

func (r *fakeRepo) EnableUserTOTPCredential(_ context.Context, userID pgtype.UUID) (postgres.UserTotpCredential, error) {
	credential, ok := r.totpByUserID[userID.String()]
	if !ok {
		return postgres.UserTotpCredential{}, pgx.ErrNoRows
	}
	credential.EnabledAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	credential.UpdatedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	r.totpByUserID[userID.String()] = credential
	return credential, nil
}

func (r *fakeRepo) GetAPIKeyByID(_ context.Context, id pgtype.UUID) (postgres.ApiKey, error) {
	key, ok := r.apiKeysByID[id.String()]
	if !ok {
		return postgres.ApiKey{}, pgx.ErrNoRows
	}
	return key, nil
}

func (r *fakeRepo) GetAPIKeyByPrefix(_ context.Context, keyPrefix string) (postgres.ApiKey, error) {
	key, ok := r.apiKeysByPref[keyPrefix]
	if !ok {
		return postgres.ApiKey{}, pgx.ErrNoRows
	}
	return key, nil
}

func (r *fakeRepo) newUUID() pgtype.UUID {
	r.nextUUIDByte++
	var bytes [16]byte
	bytes[15] = r.nextUUIDByte
	return pgtype.UUID{Bytes: bytes, Valid: true}
}

func (r *fakeRepo) slugByOrgID(id string) string {
	for slug, org := range r.orgsBySlug {
		if org.ID.String() == id {
			return slug
		}
	}
	return ""
}
