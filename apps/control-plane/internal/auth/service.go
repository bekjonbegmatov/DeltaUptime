package auth

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"deltauptime/apps/control-plane/internal/database"
	"deltauptime/packages/database/postgres"
)

var (
	ErrInvalidInput          = errors.New("invalid input")
	ErrEmailTaken            = errors.New("email already exists")
	ErrOrganizationSlugTaken = errors.New("organization slug already exists")
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrRefreshTokenInvalid   = errors.New("refresh token is invalid")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrTOTPRequired          = errors.New("totp code is required")
	ErrTOTPInvalid           = errors.New("invalid totp code")
	ErrPermissionDenied      = errors.New("permission denied")
	ErrAPIKeyInvalid         = errors.New("invalid api key")
	ErrMembershipExists      = errors.New("membership already exists")
)

type QueryRepository interface {
	CreateAPIKey(ctx context.Context, arg postgres.CreateAPIKeyParams) (postgres.ApiKey, error)
	CreateAuditLog(ctx context.Context, arg postgres.CreateAuditLogParams) (postgres.AuditLog, error)
	CreateAuthRefreshToken(ctx context.Context, arg postgres.CreateAuthRefreshTokenParams) (postgres.AuthRefreshToken, error)
	CreateMembership(ctx context.Context, arg postgres.CreateMembershipParams) (postgres.Membership, error)
	CreateOrganization(ctx context.Context, arg postgres.CreateOrganizationParams) (postgres.Organization, error)
	CreateUser(ctx context.Context, arg postgres.CreateUserParams) (postgres.User, error)
	DeleteUserTOTPCredential(ctx context.Context, userID pgtype.UUID) error
	EnableUserTOTPCredential(ctx context.Context, userID pgtype.UUID) (postgres.UserTotpCredential, error)
	GetAPIKeyByID(ctx context.Context, id pgtype.UUID) (postgres.ApiKey, error)
	GetAPIKeyByPrefix(ctx context.Context, keyPrefix string) (postgres.ApiKey, error)
	GetAuthRefreshTokenByTokenHash(ctx context.Context, tokenHash string) (postgres.AuthRefreshToken, error)
	GetMembership(ctx context.Context, arg postgres.GetMembershipParams) (postgres.Membership, error)
	GetOrganizationByID(ctx context.Context, id pgtype.UUID) (postgres.Organization, error)
	GetOrganizationBySlug(ctx context.Context, slug string) (postgres.Organization, error)
	GetUserByEmail(ctx context.Context, email string) (postgres.User, error)
	GetUserByID(ctx context.Context, id pgtype.UUID) (postgres.User, error)
	GetUserTOTPCredentialByUserID(ctx context.Context, userID pgtype.UUID) (postgres.UserTotpCredential, error)
	ListAPIKeysByOrganization(ctx context.Context, organizationID pgtype.UUID) ([]postgres.ApiKey, error)
	ListAuditLogsByOrganization(ctx context.Context, arg postgres.ListAuditLogsByOrganizationParams) ([]postgres.AuditLog, error)
	ListOrganizationsByUser(ctx context.Context, userID pgtype.UUID) ([]postgres.ListOrganizationsByUserRow, error)
	ListUsersByOrganization(ctx context.Context, organizationID pgtype.UUID) ([]postgres.ListUsersByOrganizationRow, error)
	RevokeAPIKey(ctx context.Context, id pgtype.UUID) (postgres.ApiKey, error)
	RotateAuthRefreshToken(ctx context.Context, arg postgres.RotateAuthRefreshTokenParams) (postgres.AuthRefreshToken, error)
	TouchAPIKeyLastUsed(ctx context.Context, id pgtype.UUID) (postgres.ApiKey, error)
	UpdateMembershipRole(ctx context.Context, arg postgres.UpdateMembershipRoleParams) (postgres.Membership, error)
	UpsertUserTOTPCredential(ctx context.Context, arg postgres.UpsertUserTOTPCredentialParams) (postgres.UserTotpCredential, error)
}

type Tx interface {
	QueryRepository
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type Repository interface {
	QueryRepository
	Begin(ctx context.Context) (Tx, error)
}

type Config struct {
	AccessTokenSecret  string
	RefreshTokenSecret string
	SecretsMasterKey   string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
}

type Service struct {
	repo       Repository
	hasher     PasswordHasher
	tokens     *TokenManager
	secrets    *SecretsManager
	totp       *TOTPManager
	timeNow    func() time.Time
	totpIssuer string
}

type RegisterInput struct {
	Email            string
	Password         string
	DisplayName      string
	OrganizationName string
	OrganizationSlug string
}

type LoginInput struct {
	Email    string
	Password string
	TOTPCode string
}

type RefreshInput struct {
	RefreshToken string
}

type MembershipCreateInput struct {
	Email string
	Role  string
}

type MembershipUpdateInput struct {
	UserID string
	Role   string
}

type APIKeyCreateInput struct {
	Name   string
	Scopes []string
}

type UserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	DisplayName   string `json:"display_name"`
	IsSystemAdmin bool   `json:"is_system_admin"`
	TOTPEnabled   bool   `json:"totp_enabled"`
}

type OrganizationInfo struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type AuthResult struct {
	AccessToken           string             `json:"access_token"`
	AccessTokenExpiresAt  time.Time          `json:"access_token_expires_at"`
	RefreshToken          string             `json:"refresh_token,omitempty"`
	RefreshTokenExpiresAt time.Time          `json:"refresh_token_expires_at,omitempty"`
	User                  UserInfo           `json:"user"`
	Organizations         []OrganizationInfo `json:"organizations"`
}

func NewService(store *database.Store, cfg Config) (*Service, error) {
	tokenManager, err := NewTokenManager(cfg.AccessTokenSecret, cfg.RefreshTokenSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	if err != nil {
		return nil, err
	}

	secretsManager, err := NewSecretsManager(cfg.SecretsMasterKey)
	if err != nil {
		return nil, err
	}

	return &Service{
		repo:       newStoreRepository(store),
		hasher:     PasswordHasher{},
		tokens:     tokenManager,
		secrets:    secretsManager,
		totp:       NewTOTPManager("DeltaUptime"),
		timeNow:    time.Now,
		totpIssuer: "DeltaUptime",
	}, nil
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (AuthResult, error) {
	input.Email = normalizeEmail(input.Email)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.OrganizationName = strings.TrimSpace(input.OrganizationName)
	input.OrganizationSlug = normalizeSlug(input.OrganizationSlug)

	if err := validateRegistrationInput(input); err != nil {
		return AuthResult{}, err
	}

	passwordHash, err := s.hasher.HashPassword(input.Password)
	if err != nil {
		return AuthResult{}, fmt.Errorf("hash password: %w", err)
	}

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return AuthResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer rollbackTx(ctx, tx)

	user, err := tx.CreateUser(ctx, postgres.CreateUserParams{
		Email:         input.Email,
		PasswordHash:  passwordHash,
		DisplayName:   input.DisplayName,
		IsSystemAdmin: false,
	})
	if err != nil {
		return AuthResult{}, mapCreateUserError(err)
	}

	org, err := tx.CreateOrganization(ctx, postgres.CreateOrganizationParams{
		Slug: input.OrganizationSlug,
		Name: input.OrganizationName,
	})
	if err != nil {
		return AuthResult{}, mapCreateOrganizationError(err)
	}

	if _, err := tx.CreateMembership(ctx, postgres.CreateMembershipParams{
		OrganizationID: org.ID,
		UserID:         user.ID,
		Role:           "owner",
	}); err != nil {
		return AuthResult{}, fmt.Errorf("create owner membership: %w", err)
	}

	authResult, err := s.issueSession(ctx, tx, user, []OrganizationInfo{{
		ID:   org.ID.String(),
		Slug: org.Slug,
		Name: org.Name,
		Role: "owner",
	}})
	if err != nil {
		return AuthResult{}, err
	}

	if err := s.audit(ctx, tx, auditEntry{
		OrganizationID: org.ID,
		ActorType:      string(PrincipalTypeUser),
		ActorUserID:    user.ID,
		Action:         "auth.register",
		TargetType:     "user",
		TargetID:       user.ID.String(),
		Metadata: map[string]any{
			"organization_slug": org.Slug,
		},
	}); err != nil {
		return AuthResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return AuthResult{}, fmt.Errorf("commit register tx: %w", err)
	}

	return authResult, nil
}

func (s *Service) Login(ctx context.Context, input LoginInput) (AuthResult, error) {
	input.Email = normalizeEmail(input.Email)
	if input.Email == "" || input.Password == "" {
		return AuthResult{}, fmt.Errorf("%w: email and password are required", ErrInvalidInput)
	}

	user, err := s.repo.GetUserByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AuthResult{}, ErrInvalidCredentials
		}
		return AuthResult{}, fmt.Errorf("get user by email: %w", err)
	}

	ok, err := s.hasher.VerifyPassword(user.PasswordHash, input.Password)
	if err != nil {
		return AuthResult{}, fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return AuthResult{}, ErrInvalidCredentials
	}

	if err := s.verifyTOTPIfEnabled(ctx, s.repo, user.ID, input.TOTPCode); err != nil {
		return AuthResult{}, err
	}

	orgs, err := s.organizationsForUser(ctx, s.repo, user.ID)
	if err != nil {
		return AuthResult{}, err
	}

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return AuthResult{}, fmt.Errorf("begin login tx: %w", err)
	}
	defer rollbackTx(ctx, tx)

	authResult, err := s.issueSession(ctx, tx, user, orgs)
	if err != nil {
		return AuthResult{}, err
	}

	if err := s.audit(ctx, tx, auditEntry{
		ActorType:   string(PrincipalTypeUser),
		ActorUserID: user.ID,
		Action:      "auth.login",
		TargetType:  "user",
		TargetID:    user.ID.String(),
	}); err != nil {
		return AuthResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return AuthResult{}, fmt.Errorf("commit login tx: %w", err)
	}

	return authResult, nil
}

func (s *Service) Refresh(ctx context.Context, input RefreshInput) (AuthResult, error) {
	if strings.TrimSpace(input.RefreshToken) == "" {
		return AuthResult{}, fmt.Errorf("%w: refresh token is required", ErrInvalidInput)
	}

	tokenHash := s.tokens.HashRefreshToken(input.RefreshToken)

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return AuthResult{}, fmt.Errorf("begin refresh tx: %w", err)
	}
	defer rollbackTx(ctx, tx)

	session, err := tx.GetAuthRefreshTokenByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AuthResult{}, ErrRefreshTokenInvalid
		}
		return AuthResult{}, fmt.Errorf("get refresh token: %w", err)
	}

	now := s.timeNow()
	if session.UsedAt.Valid || session.RevokedAt.Valid || !session.ExpiresAt.Valid || now.After(session.ExpiresAt.Time) {
		return AuthResult{}, ErrRefreshTokenInvalid
	}

	user, err := tx.GetUserByID(ctx, session.UserID)
	if err != nil {
		return AuthResult{}, fmt.Errorf("get refresh user: %w", err)
	}

	orgs, err := s.organizationsForUser(ctx, tx, user.ID)
	if err != nil {
		return AuthResult{}, err
	}

	authResult, replacement, err := s.issueSessionWithModel(ctx, tx, user, orgs)
	if err != nil {
		return AuthResult{}, err
	}

	if _, err := tx.RotateAuthRefreshToken(ctx, postgres.RotateAuthRefreshTokenParams{
		ID:                session.ID,
		ReplacedByTokenID: replacement.ID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AuthResult{}, ErrRefreshTokenInvalid
		}
		return AuthResult{}, fmt.Errorf("rotate refresh token: %w", err)
	}

	if err := s.audit(ctx, tx, auditEntry{
		ActorType:   string(PrincipalTypeUser),
		ActorUserID: user.ID,
		Action:      "auth.refresh",
		TargetType:  "refresh_token",
		TargetID:    session.ID.String(),
	}); err != nil {
		return AuthResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return AuthResult{}, fmt.Errorf("commit refresh tx: %w", err)
	}

	return authResult, nil
}

func (s *Service) Me(ctx context.Context, accessToken string) (AuthResult, error) {
	principal, err := s.AuthenticateAccessToken(ctx, accessToken)
	if err != nil {
		return AuthResult{}, err
	}

	orgs, err := s.organizationsForPrincipal(ctx, principal)
	if err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		User:          principal.User,
		Organizations: orgs,
	}, nil
}

func (s *Service) AuthenticateAccessToken(ctx context.Context, accessToken string) (Principal, error) {
	userID, err := s.tokens.ParseAccessToken(accessToken)
	if err != nil {
		return Principal{}, ErrUnauthorized
	}

	pgUserID := pgtype.UUID{}
	if err := pgUserID.Scan(userID); err != nil {
		return Principal{}, ErrUnauthorized
	}

	user, err := s.repo.GetUserByID(ctx, pgUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Principal{}, ErrUnauthorized
		}
		return Principal{}, fmt.Errorf("get current user: %w", err)
	}

	return s.principalForUser(ctx, user)
}

func (s *Service) AuthenticateAPIKey(ctx context.Context, rawAPIKey string) (Principal, error) {
	prefix, err := parseAPIKeyPrefix(rawAPIKey)
	if err != nil {
		return Principal{}, ErrAPIKeyInvalid
	}

	apiKey, err := s.repo.GetAPIKeyByPrefix(ctx, prefix)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Principal{}, ErrAPIKeyInvalid
		}
		return Principal{}, fmt.Errorf("get api key: %w", err)
	}
	if apiKey.RevokedAt.Valid || apiKey.KeyHash != s.secrets.Hash(rawAPIKey) {
		return Principal{}, ErrAPIKeyInvalid
	}

	if _, err := s.repo.TouchAPIKeyLastUsed(ctx, apiKey.ID); err != nil {
		return Principal{}, fmt.Errorf("touch api key: %w", err)
	}

	principal := Principal{
		Type:              PrincipalTypeAPIKey,
		APIKeyID:          apiKey.ID.String(),
		OrganizationRoles: map[string]string{apiKey.OrganizationID.String(): "api_key"},
		Permissions: map[string]map[Permission]struct{}{
			apiKey.OrganizationID.String(): scopesToPermissions(apiKey.Scopes),
		},
	}

	return principal, nil
}

func (s *Service) SetupTOTP(ctx context.Context, principal Principal) (TOTPSetup, error) {
	userID, err := parseUUID(principal.User.ID)
	if err != nil {
		return TOTPSetup{}, err
	}

	secret, err := s.totp.GenerateSecret()
	if err != nil {
		return TOTPSetup{}, err
	}

	ciphertext, nonce, err := s.secrets.Encrypt([]byte(secret))
	if err != nil {
		return TOTPSetup{}, fmt.Errorf("encrypt totp secret: %w", err)
	}

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return TOTPSetup{}, fmt.Errorf("begin totp setup tx: %w", err)
	}
	defer rollbackTx(ctx, tx)

	if _, err := tx.UpsertUserTOTPCredential(ctx, postgres.UpsertUserTOTPCredentialParams{
		UserID:           userID,
		SecretCiphertext: ciphertext,
		SecretNonce:      nonce,
	}); err != nil {
		return TOTPSetup{}, fmt.Errorf("upsert totp credential: %w", err)
	}

	if err := s.audit(ctx, tx, auditEntry{
		ActorType:   string(PrincipalTypeUser),
		ActorUserID: userID,
		Action:      "auth.totp.setup",
		TargetType:  "user",
		TargetID:    principal.User.ID,
	}); err != nil {
		return TOTPSetup{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return TOTPSetup{}, fmt.Errorf("commit totp setup tx: %w", err)
	}

	return TOTPSetup{
		Secret:     secret,
		OTPAuthURL: s.totp.OTPAuthURL(principal.User.Email, secret),
	}, nil
}

func (s *Service) EnableTOTP(ctx context.Context, principal Principal, code string) (UserInfo, error) {
	userID, err := parseUUID(principal.User.ID)
	if err != nil {
		return UserInfo{}, err
	}

	credential, err := s.repo.GetUserTOTPCredentialByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserInfo{}, fmt.Errorf("%w: totp setup not started", ErrInvalidInput)
		}
		return UserInfo{}, fmt.Errorf("get totp credential: %w", err)
	}

	secret, err := s.decryptTOTPSecret(credential)
	if err != nil {
		return UserInfo{}, err
	}
	if !s.totp.VerifyCode(secret, code, s.timeNow().UTC()) {
		return UserInfo{}, ErrTOTPInvalid
	}

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return UserInfo{}, fmt.Errorf("begin totp enable tx: %w", err)
	}
	defer rollbackTx(ctx, tx)

	if _, err := tx.EnableUserTOTPCredential(ctx, userID); err != nil {
		return UserInfo{}, fmt.Errorf("enable totp credential: %w", err)
	}

	if err := s.audit(ctx, tx, auditEntry{
		ActorType:   string(PrincipalTypeUser),
		ActorUserID: userID,
		Action:      "auth.totp.enable",
		TargetType:  "user",
		TargetID:    principal.User.ID,
	}); err != nil {
		return UserInfo{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return UserInfo{}, fmt.Errorf("commit totp enable tx: %w", err)
	}

	principal.User.TOTPEnabled = true
	return principal.User, nil
}

func (s *Service) DisableTOTP(ctx context.Context, principal Principal, code string) (UserInfo, error) {
	userID, err := parseUUID(principal.User.ID)
	if err != nil {
		return UserInfo{}, err
	}

	credential, err := s.repo.GetUserTOTPCredentialByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserInfo{}, fmt.Errorf("%w: totp is not enabled", ErrInvalidInput)
		}
		return UserInfo{}, fmt.Errorf("get totp credential: %w", err)
	}
	if !credential.EnabledAt.Valid {
		return UserInfo{}, fmt.Errorf("%w: totp is not enabled", ErrInvalidInput)
	}

	secret, err := s.decryptTOTPSecret(credential)
	if err != nil {
		return UserInfo{}, err
	}
	if !s.totp.VerifyCode(secret, code, s.timeNow().UTC()) {
		return UserInfo{}, ErrTOTPInvalid
	}

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return UserInfo{}, fmt.Errorf("begin totp disable tx: %w", err)
	}
	defer rollbackTx(ctx, tx)

	if err := tx.DeleteUserTOTPCredential(ctx, userID); err != nil {
		return UserInfo{}, fmt.Errorf("delete totp credential: %w", err)
	}

	if err := s.audit(ctx, tx, auditEntry{
		ActorType:   string(PrincipalTypeUser),
		ActorUserID: userID,
		Action:      "auth.totp.disable",
		TargetType:  "user",
		TargetID:    principal.User.ID,
	}); err != nil {
		return UserInfo{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return UserInfo{}, fmt.Errorf("commit totp disable tx: %w", err)
	}

	principal.User.TOTPEnabled = false
	return principal.User, nil
}

func (s *Service) ListOrganizations(ctx context.Context, principal Principal) ([]OrganizationInfo, error) {
	return s.organizationsForPrincipal(ctx, principal)
}

func (s *Service) GetOrganization(ctx context.Context, principal Principal, slug string) (OrganizationInfo, error) {
	org, err := s.repo.GetOrganizationBySlug(ctx, normalizeSlug(slug))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OrganizationInfo{}, ErrUnauthorized
		}
		return OrganizationInfo{}, fmt.Errorf("get organization: %w", err)
	}

	if err := s.requirePermission(principal, org.ID.String(), PermissionOrganizationRead); err != nil {
		return OrganizationInfo{}, err
	}

	role := principal.OrganizationRoles[org.ID.String()]
	return OrganizationInfo{
		ID:   org.ID.String(),
		Slug: org.Slug,
		Name: org.Name,
		Role: role,
	}, nil
}

func (s *Service) ListMemberships(ctx context.Context, principal Principal, slug string) ([]MembershipInfo, error) {
	org, err := s.repo.GetOrganizationBySlug(ctx, normalizeSlug(slug))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUnauthorized
		}
		return nil, fmt.Errorf("get organization: %w", err)
	}

	if err := s.requirePermission(principal, org.ID.String(), PermissionMembershipRead); err != nil {
		return nil, err
	}

	rows, err := s.repo.ListUsersByOrganization(ctx, org.ID)
	if err != nil {
		return nil, fmt.Errorf("list users by organization: %w", err)
	}

	memberships := make([]MembershipInfo, 0, len(rows))
	for _, row := range rows {
		totpEnabled, err := s.userHasEnabledTOTP(ctx, s.repo, row.ID)
		if err != nil {
			return nil, err
		}
		memberships = append(memberships, MembershipInfo{
			User: UserInfo{
				ID:            row.ID.String(),
				Email:         row.Email,
				DisplayName:   row.DisplayName,
				IsSystemAdmin: row.IsSystemAdmin,
				TOTPEnabled:   totpEnabled,
			},
			Role: row.Role,
		})
	}

	return memberships, nil
}

func (s *Service) AddMembership(ctx context.Context, principal Principal, slug string, input MembershipCreateInput) (MembershipInfo, error) {
	org, err := s.repo.GetOrganizationBySlug(ctx, normalizeSlug(slug))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MembershipInfo{}, ErrUnauthorized
		}
		return MembershipInfo{}, fmt.Errorf("get organization: %w", err)
	}

	if err := s.requirePermission(principal, org.ID.String(), PermissionMembershipWrite); err != nil {
		return MembershipInfo{}, err
	}

	input.Email = normalizeEmail(input.Email)
	if _, err := mail.ParseAddress(input.Email); err != nil {
		return MembershipInfo{}, fmt.Errorf("%w: invalid email", ErrInvalidInput)
	}
	if err := validateRole(input.Role); err != nil {
		return MembershipInfo{}, err
	}

	user, err := s.repo.GetUserByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MembershipInfo{}, fmt.Errorf("%w: user must exist before membership can be added", ErrInvalidInput)
		}
		return MembershipInfo{}, fmt.Errorf("get user by email: %w", err)
	}

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return MembershipInfo{}, fmt.Errorf("begin membership create tx: %w", err)
	}
	defer rollbackTx(ctx, tx)

	if _, err := tx.CreateMembership(ctx, postgres.CreateMembershipParams{
		OrganizationID: org.ID,
		UserID:         user.ID,
		Role:           input.Role,
	}); err != nil {
		if isUniqueViolation(err) {
			return MembershipInfo{}, ErrMembershipExists
		}
		return MembershipInfo{}, fmt.Errorf("create membership: %w", err)
	}

	if err := s.audit(ctx, tx, auditEntry{
		OrganizationID: org.ID,
		ActorType:      string(principal.Type),
		ActorUserID:    mustUUID(principal.User.ID),
		ActorAPIKeyID:  mustUUID(principal.APIKeyID),
		Action:         "membership.create",
		TargetType:     "membership",
		TargetID:       user.ID.String(),
		Metadata: map[string]any{
			"role":  input.Role,
			"email": user.Email,
		},
	}); err != nil {
		return MembershipInfo{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return MembershipInfo{}, fmt.Errorf("commit membership create tx: %w", err)
	}

	totpEnabled, err := s.userHasEnabledTOTP(ctx, s.repo, user.ID)
	if err != nil {
		return MembershipInfo{}, err
	}

	return MembershipInfo{
		User: UserInfo{
			ID:            user.ID.String(),
			Email:         user.Email,
			DisplayName:   user.DisplayName,
			IsSystemAdmin: user.IsSystemAdmin,
			TOTPEnabled:   totpEnabled,
		},
		Role: input.Role,
	}, nil
}

func (s *Service) UpdateMembership(ctx context.Context, principal Principal, slug string, input MembershipUpdateInput) (MembershipInfo, error) {
	org, err := s.repo.GetOrganizationBySlug(ctx, normalizeSlug(slug))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MembershipInfo{}, ErrUnauthorized
		}
		return MembershipInfo{}, fmt.Errorf("get organization: %w", err)
	}

	if err := s.requirePermission(principal, org.ID.String(), PermissionMembershipWrite); err != nil {
		return MembershipInfo{}, err
	}
	if err := validateRole(input.Role); err != nil {
		return MembershipInfo{}, err
	}

	userID, err := parseUUID(input.UserID)
	if err != nil {
		return MembershipInfo{}, err
	}

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return MembershipInfo{}, fmt.Errorf("begin membership update tx: %w", err)
	}
	defer rollbackTx(ctx, tx)

	membership, err := tx.UpdateMembershipRole(ctx, postgres.UpdateMembershipRoleParams{
		OrganizationID: org.ID,
		UserID:         userID,
		Role:           input.Role,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MembershipInfo{}, ErrUnauthorized
		}
		return MembershipInfo{}, fmt.Errorf("update membership role: %w", err)
	}

	user, err := tx.GetUserByID(ctx, membership.UserID)
	if err != nil {
		return MembershipInfo{}, fmt.Errorf("get updated membership user: %w", err)
	}

	if err := s.audit(ctx, tx, auditEntry{
		OrganizationID: org.ID,
		ActorType:      string(principal.Type),
		ActorUserID:    mustUUID(principal.User.ID),
		ActorAPIKeyID:  mustUUID(principal.APIKeyID),
		Action:         "membership.update_role",
		TargetType:     "membership",
		TargetID:       user.ID.String(),
		Metadata: map[string]any{
			"role": input.Role,
		},
	}); err != nil {
		return MembershipInfo{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return MembershipInfo{}, fmt.Errorf("commit membership update tx: %w", err)
	}

	totpEnabled, err := s.userHasEnabledTOTP(ctx, s.repo, user.ID)
	if err != nil {
		return MembershipInfo{}, err
	}

	return MembershipInfo{
		User: UserInfo{
			ID:            user.ID.String(),
			Email:         user.Email,
			DisplayName:   user.DisplayName,
			IsSystemAdmin: user.IsSystemAdmin,
			TOTPEnabled:   totpEnabled,
		},
		Role: membership.Role,
	}, nil
}

func (s *Service) CreateAPIKey(ctx context.Context, principal Principal, slug string, input APIKeyCreateInput) (APIKeyCreateResult, error) {
	org, err := s.repo.GetOrganizationBySlug(ctx, normalizeSlug(slug))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return APIKeyCreateResult{}, ErrUnauthorized
		}
		return APIKeyCreateResult{}, fmt.Errorf("get organization: %w", err)
	}

	if err := s.requirePermission(principal, org.ID.String(), PermissionAPIKeyWrite); err != nil {
		return APIKeyCreateResult{}, err
	}

	scopes, err := validateScopes(input.Scopes)
	if err != nil {
		return APIKeyCreateResult{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return APIKeyCreateResult{}, fmt.Errorf("%w: api key name is required", ErrInvalidInput)
	}

	rawKey, prefix, err := generateAPIKey()
	if err != nil {
		return APIKeyCreateResult{}, err
	}
	hash := s.secrets.Hash(rawKey)

	actorUserID := mustUUID(principal.User.ID)
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return APIKeyCreateResult{}, fmt.Errorf("begin api key create tx: %w", err)
	}
	defer rollbackTx(ctx, tx)

	apiKey, err := tx.CreateAPIKey(ctx, postgres.CreateAPIKeyParams{
		OrganizationID:  org.ID,
		CreatedByUserID: actorUserID,
		Name:            name,
		KeyPrefix:       prefix,
		KeyHash:         hash,
		Scopes:          scopes,
	})
	if err != nil {
		return APIKeyCreateResult{}, fmt.Errorf("create api key: %w", err)
	}

	if err := s.audit(ctx, tx, auditEntry{
		OrganizationID: org.ID,
		ActorType:      string(principal.Type),
		ActorUserID:    actorUserID,
		Action:         "api_key.create",
		TargetType:     "api_key",
		TargetID:       apiKey.ID.String(),
		Metadata: map[string]any{
			"name":   apiKey.Name,
			"scopes": apiKey.Scopes,
			"prefix": apiKey.KeyPrefix,
		},
	}); err != nil {
		return APIKeyCreateResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return APIKeyCreateResult{}, fmt.Errorf("commit api key create tx: %w", err)
	}

	return APIKeyCreateResult{
		APIKeyInfo: apiKeyInfoFromModel(apiKey),
		Token:      rawKey,
	}, nil
}

func (s *Service) ListAPIKeys(ctx context.Context, principal Principal, slug string) ([]APIKeyInfo, error) {
	org, err := s.repo.GetOrganizationBySlug(ctx, normalizeSlug(slug))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUnauthorized
		}
		return nil, fmt.Errorf("get organization: %w", err)
	}

	if err := s.requirePermission(principal, org.ID.String(), PermissionAPIKeyRead); err != nil {
		return nil, err
	}

	keys, err := s.repo.ListAPIKeysByOrganization(ctx, org.ID)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}

	items := make([]APIKeyInfo, 0, len(keys))
	for _, key := range keys {
		items = append(items, apiKeyInfoFromModel(key))
	}
	return items, nil
}

func (s *Service) RevokeAPIKey(ctx context.Context, principal Principal, slug, apiKeyID string) (APIKeyInfo, error) {
	org, err := s.repo.GetOrganizationBySlug(ctx, normalizeSlug(slug))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return APIKeyInfo{}, ErrUnauthorized
		}
		return APIKeyInfo{}, fmt.Errorf("get organization: %w", err)
	}

	if err := s.requirePermission(principal, org.ID.String(), PermissionAPIKeyWrite); err != nil {
		return APIKeyInfo{}, err
	}

	keyID, err := parseUUID(apiKeyID)
	if err != nil {
		return APIKeyInfo{}, err
	}

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return APIKeyInfo{}, fmt.Errorf("begin api key revoke tx: %w", err)
	}
	defer rollbackTx(ctx, tx)

	apiKey, err := tx.RevokeAPIKey(ctx, keyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return APIKeyInfo{}, ErrUnauthorized
		}
		return APIKeyInfo{}, fmt.Errorf("revoke api key: %w", err)
	}
	if apiKey.OrganizationID != org.ID {
		return APIKeyInfo{}, ErrUnauthorized
	}

	if err := s.audit(ctx, tx, auditEntry{
		OrganizationID: org.ID,
		ActorType:      string(principal.Type),
		ActorUserID:    mustUUID(principal.User.ID),
		ActorAPIKeyID:  mustUUID(principal.APIKeyID),
		Action:         "api_key.revoke",
		TargetType:     "api_key",
		TargetID:       apiKey.ID.String(),
		Metadata: map[string]any{
			"prefix": apiKey.KeyPrefix,
		},
	}); err != nil {
		return APIKeyInfo{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return APIKeyInfo{}, fmt.Errorf("commit api key revoke tx: %w", err)
	}

	return apiKeyInfoFromModel(apiKey), nil
}

func (s *Service) ListAuditLogs(ctx context.Context, principal Principal, slug string, limit int32) ([]AuditLogInfo, error) {
	org, err := s.repo.GetOrganizationBySlug(ctx, normalizeSlug(slug))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUnauthorized
		}
		return nil, fmt.Errorf("get organization: %w", err)
	}

	if err := s.requirePermission(principal, org.ID.String(), PermissionAuditRead); err != nil {
		return nil, err
	}

	if limit <= 0 || limit > 200 {
		limit = 100
	}

	rows, err := s.repo.ListAuditLogsByOrganization(ctx, postgres.ListAuditLogsByOrganizationParams{
		OrganizationID: org.ID,
		Limit:          limit,
		Offset:         0,
	})
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}

	items := make([]AuditLogInfo, 0, len(rows))
	for _, row := range rows {
		items = append(items, auditLogInfoFromModel(row))
	}
	return items, nil
}

func (s *Service) organizationsForPrincipal(ctx context.Context, principal Principal) ([]OrganizationInfo, error) {
	switch principal.Type {
	case PrincipalTypeUser:
		userID, err := parseUUID(principal.User.ID)
		if err != nil {
			return nil, err
		}
		return s.organizationsForUser(ctx, s.repo, userID)
	case PrincipalTypeAPIKey:
		orgs := make([]OrganizationInfo, 0, len(principal.OrganizationRoles))
		for orgID := range principal.OrganizationRoles {
			pgOrgID, err := parseUUID(orgID)
			if err != nil {
				return nil, err
			}
			org, err := s.repo.GetOrganizationByID(ctx, pgOrgID)
			if err != nil {
				return nil, fmt.Errorf("get organization for api key: %w", err)
			}
			orgs = append(orgs, OrganizationInfo{
				ID:   org.ID.String(),
				Slug: org.Slug,
				Name: org.Name,
				Role: "api_key",
			})
		}
		return orgs, nil
	default:
		return nil, ErrUnauthorized
	}
}

func (s *Service) principalForUser(ctx context.Context, user postgres.User) (Principal, error) {
	orgs, err := s.organizationsForUser(ctx, s.repo, user.ID)
	if err != nil {
		return Principal{}, err
	}

	permissions := make(map[string]map[Permission]struct{}, len(orgs))
	roles := make(map[string]string, len(orgs))
	for _, org := range orgs {
		roles[org.ID] = org.Role
		permissions[org.ID] = permissionsForRole(org.Role)
	}

	totpEnabled, err := s.userHasEnabledTOTP(ctx, s.repo, user.ID)
	if err != nil {
		return Principal{}, err
	}

	return Principal{
		Type: PrincipalTypeUser,
		User: UserInfo{
			ID:            user.ID.String(),
			Email:         user.Email,
			DisplayName:   user.DisplayName,
			IsSystemAdmin: user.IsSystemAdmin,
			TOTPEnabled:   totpEnabled,
		},
		OrganizationRoles: roles,
		Permissions:       permissions,
	}, nil
}

func (s *Service) issueSession(ctx context.Context, repo QueryRepository, user postgres.User, orgs []OrganizationInfo) (AuthResult, error) {
	authResult, _, err := s.issueSessionWithModel(ctx, repo, user, orgs)
	return authResult, err
}

func (s *Service) issueSessionWithModel(ctx context.Context, repo QueryRepository, user postgres.User, orgs []OrganizationInfo) (AuthResult, postgres.AuthRefreshToken, error) {
	accessToken, accessExpiry, err := s.tokens.IssueAccessToken(user.ID.String())
	if err != nil {
		return AuthResult{}, postgres.AuthRefreshToken{}, fmt.Errorf("issue access token: %w", err)
	}

	refreshToken, refreshHash, refreshExpiry, err := s.tokens.GenerateRefreshToken()
	if err != nil {
		return AuthResult{}, postgres.AuthRefreshToken{}, fmt.Errorf("generate refresh token: %w", err)
	}

	session, err := repo.CreateAuthRefreshToken(ctx, postgres.CreateAuthRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: refreshHash,
		ExpiresAt: pgtype.Timestamptz{Time: refreshExpiry, Valid: true},
	})
	if err != nil {
		return AuthResult{}, postgres.AuthRefreshToken{}, fmt.Errorf("create refresh token row: %w", err)
	}

	totpEnabled, err := s.userHasEnabledTOTP(ctx, repo, user.ID)
	if err != nil {
		return AuthResult{}, postgres.AuthRefreshToken{}, err
	}

	return AuthResult{
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessExpiry,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshExpiry,
		User: UserInfo{
			ID:            user.ID.String(),
			Email:         user.Email,
			DisplayName:   user.DisplayName,
			IsSystemAdmin: user.IsSystemAdmin,
			TOTPEnabled:   totpEnabled,
		},
		Organizations: orgs,
	}, session, nil
}

func (s *Service) organizationsForUser(ctx context.Context, repo QueryRepository, userID pgtype.UUID) ([]OrganizationInfo, error) {
	rows, err := repo.ListOrganizationsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list organizations by user: %w", err)
	}

	orgs := make([]OrganizationInfo, 0, len(rows))
	for _, row := range rows {
		orgs = append(orgs, OrganizationInfo{
			ID:   row.ID.String(),
			Slug: row.Slug,
			Name: row.Name,
			Role: row.Role,
		})
	}
	return orgs, nil
}

func (s *Service) verifyTOTPIfEnabled(ctx context.Context, repo QueryRepository, userID pgtype.UUID, code string) error {
	credential, err := repo.GetUserTOTPCredentialByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("get totp credential: %w", err)
	}
	if !credential.EnabledAt.Valid {
		return nil
	}
	if strings.TrimSpace(code) == "" {
		return ErrTOTPRequired
	}

	secret, err := s.decryptTOTPSecret(credential)
	if err != nil {
		return err
	}
	if !s.totp.VerifyCode(secret, code, s.timeNow().UTC()) {
		return ErrTOTPInvalid
	}
	return nil
}

func (s *Service) userHasEnabledTOTP(ctx context.Context, repo QueryRepository, userID pgtype.UUID) (bool, error) {
	credential, err := repo.GetUserTOTPCredentialByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("get totp credential: %w", err)
	}
	return credential.EnabledAt.Valid, nil
}

func (s *Service) decryptTOTPSecret(credential postgres.UserTotpCredential) (string, error) {
	secret, err := s.secrets.Decrypt(credential.SecretCiphertext, credential.SecretNonce)
	if err != nil {
		return "", err
	}
	return string(secret), nil
}

func (s *Service) requirePermission(principal Principal, organizationID string, permission Permission) error {
	perms, ok := principal.Permissions[organizationID]
	if !ok {
		return ErrUnauthorized
	}
	if _, ok := perms[permission]; !ok {
		return ErrPermissionDenied
	}
	return nil
}

type auditEntry struct {
	OrganizationID pgtype.UUID
	ActorType      string
	ActorUserID    pgtype.UUID
	ActorAPIKeyID  pgtype.UUID
	Action         string
	TargetType     string
	TargetID       string
	Metadata       map[string]any
}

func (s *Service) audit(ctx context.Context, repo QueryRepository, entry auditEntry) error {
	metadata, err := json.Marshal(entry.Metadata)
	if err != nil {
		return fmt.Errorf("marshal audit metadata: %w", err)
	}

	_, err = repo.CreateAuditLog(ctx, postgres.CreateAuditLogParams{
		OrganizationID: entry.OrganizationID,
		ActorType:      entry.ActorType,
		ActorUserID:    entry.ActorUserID,
		ActorApiKeyID:  entry.ActorAPIKeyID,
		Action:         entry.Action,
		TargetType:     entry.TargetType,
		TargetID:       entry.TargetID,
		Metadata:       metadata,
	})
	if err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}
	return nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func normalizeSlug(slug string) string {
	slug = strings.ToLower(strings.TrimSpace(slug))
	slug = strings.ReplaceAll(slug, " ", "-")
	return slug
}

func validateRegistrationInput(input RegisterInput) error {
	if _, err := mail.ParseAddress(input.Email); err != nil {
		return fmt.Errorf("%w: invalid email", ErrInvalidInput)
	}
	if len(input.Password) < 10 {
		return fmt.Errorf("%w: password must be at least 10 characters", ErrInvalidInput)
	}
	if input.DisplayName == "" {
		return fmt.Errorf("%w: display_name is required", ErrInvalidInput)
	}
	if input.OrganizationName == "" {
		return fmt.Errorf("%w: organization_name is required", ErrInvalidInput)
	}
	if err := validateSlug(input.OrganizationSlug); err != nil {
		return err
	}
	return nil
}

func validateRole(role string) error {
	switch role {
	case "owner", "admin", "operator", "viewer", "billing":
		return nil
	default:
		return fmt.Errorf("%w: invalid role", ErrInvalidInput)
	}
}

func validateSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("%w: organization_slug is required", ErrInvalidInput)
	}
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return fmt.Errorf("%w: organization_slug may contain only lowercase latin letters, digits, and '-'", ErrInvalidInput)
	}
	return nil
}

func validateScopes(scopes []string) ([]string, error) {
	if len(scopes) == 0 {
		return nil, fmt.Errorf("%w: at least one scope is required", ErrInvalidInput)
	}

	allowed := map[string]struct{}{
		string(PermissionOrganizationRead): {},
		string(PermissionMembershipRead):   {},
		string(PermissionMembershipWrite):  {},
		string(PermissionAPIKeyRead):       {},
		string(PermissionAPIKeyWrite):      {},
		string(PermissionAuditRead):        {},
	}

	normalized := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if _, ok := allowed[scope]; !ok {
			return nil, fmt.Errorf("%w: unsupported scope %q", ErrInvalidInput, scope)
		}
		if !slices.Contains(normalized, scope) {
			normalized = append(normalized, scope)
		}
	}
	return normalized, nil
}

func rollbackTx(ctx context.Context, tx Tx) {
	if tx == nil {
		return
	}
	_ = tx.Rollback(ctx)
}

func mapCreateUserError(err error) error {
	if isUniqueViolation(err) {
		return ErrEmailTaken
	}
	return fmt.Errorf("create user: %w", err)
}

func mapCreateOrganizationError(err error) error {
	if isUniqueViolation(err) {
		return ErrOrganizationSlugTaken
	}
	return fmt.Errorf("create organization: %w", err)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func parseUUID(raw string) (pgtype.UUID, error) {
	id := pgtype.UUID{}
	if err := id.Scan(raw); err != nil {
		return pgtype.UUID{}, fmt.Errorf("%w: invalid uuid", ErrInvalidInput)
	}
	return id, nil
}

func mustUUID(raw string) pgtype.UUID {
	if strings.TrimSpace(raw) == "" {
		return pgtype.UUID{}
	}
	id, err := parseUUID(raw)
	if err != nil {
		return pgtype.UUID{}
	}
	return id
}

func generateAPIKey() (string, string, error) {
	prefix, err := randomHex(4)
	if err != nil {
		return "", "", err
	}
	secret, err := randomHex(18)
	if err != nil {
		return "", "", err
	}
	raw := "duk_" + prefix + "_" + secret
	return raw, prefix, nil
}

func parseAPIKeyPrefix(raw string) (string, error) {
	parts := strings.Split(raw, "_")
	if len(parts) != 3 || parts[0] != "duk" {
		return "", fmt.Errorf("%w: malformed api key", ErrInvalidInput)
	}
	return parts[1], nil
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return fmt.Sprintf("%x", buf), nil
}

func scopesToPermissions(scopes []string) map[Permission]struct{} {
	perms := make(map[Permission]struct{}, len(scopes))
	for _, scope := range scopes {
		perms[Permission(scope)] = struct{}{}
	}
	return perms
}

func apiKeyInfoFromModel(model postgres.ApiKey) APIKeyInfo {
	info := APIKeyInfo{
		ID:        model.ID.String(),
		Name:      model.Name,
		Prefix:    model.KeyPrefix,
		Scopes:    append([]string(nil), model.Scopes...),
		CreatedAt: model.CreatedAt.Time,
	}
	if model.CreatedByUserID.Valid {
		info.CreatedByUserID = model.CreatedByUserID.String()
	}
	if model.LastUsedAt.Valid {
		info.LastUsedAt = model.LastUsedAt.Time
	}
	if model.RevokedAt.Valid {
		info.RevokedAt = model.RevokedAt.Time
	}
	return info
}

func auditLogInfoFromModel(model postgres.AuditLog) AuditLogInfo {
	info := AuditLogInfo{
		ID:         model.ID.String(),
		ActorType:  model.ActorType,
		Action:     model.Action,
		TargetType: model.TargetType,
		TargetID:   model.TargetID,
		Metadata:   append([]byte(nil), model.Metadata...),
		OccurredAt: model.OccurredAt.Time,
	}
	if model.ActorUserID.Valid {
		info.ActorUserID = model.ActorUserID.String()
	}
	if model.ActorApiKeyID.Valid {
		info.ActorAPIKeyID = model.ActorApiKeyID.String()
	}
	return info
}
