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

func newTestService(t *testing.T, repo Repository) *Service {
	t.Helper()

	tokens, err := NewTokenManager("access-secret", "refresh-secret", 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("NewTokenManager: %v", err)
	}

	return &Service{
		repo:    repo,
		hasher:  PasswordHasher{},
		tokens:  tokens,
		timeNow: time.Now,
	}
}

type fakeRepo struct {
	usersByEmail  map[string]postgres.User
	usersByID     map[string]postgres.User
	orgsBySlug    map[string]postgres.Organization
	orgsByUserID  map[string][]postgres.ListOrganizationsByUserRow
	refreshByHash map[string]postgres.AuthRefreshToken
	refreshByID   map[string]string
	memberships   []postgres.Membership
	nextUUIDByte  byte
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		usersByEmail:  map[string]postgres.User{},
		usersByID:     map[string]postgres.User{},
		orgsBySlug:    map[string]postgres.Organization{},
		orgsByUserID:  map[string][]postgres.ListOrganizationsByUserRow{},
		refreshByHash: map[string]postgres.AuthRefreshToken{},
		refreshByID:   map[string]string{},
	}
}

func (r *fakeRepo) Begin(context.Context) (Tx, error) { return r, nil }
func (r *fakeRepo) Commit(context.Context) error      { return nil }
func (r *fakeRepo) Rollback(context.Context) error    { return nil }

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
	membership := postgres.Membership{
		ID:             r.newUUID(),
		OrganizationID: arg.OrganizationID,
		UserID:         arg.UserID,
		Role:           arg.Role,
		CreatedAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	r.memberships = append(r.memberships, membership)
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

func (r *fakeRepo) ListOrganizationsByUser(_ context.Context, userID pgtype.UUID) ([]postgres.ListOrganizationsByUserRow, error) {
	return append([]postgres.ListOrganizationsByUserRow(nil), r.orgsByUserID[userID.String()]...), nil
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

func (r *fakeRepo) newUUID() pgtype.UUID {
	r.nextUUIDByte++
	var bytes [16]byte
	bytes[15] = r.nextUUIDByte
	return pgtype.UUID{Bytes: bytes, Valid: true}
}
