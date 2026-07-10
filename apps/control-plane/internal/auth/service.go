package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
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
)

type QueryRepository interface {
	CreateAuthRefreshToken(ctx context.Context, arg postgres.CreateAuthRefreshTokenParams) (postgres.AuthRefreshToken, error)
	CreateMembership(ctx context.Context, arg postgres.CreateMembershipParams) (postgres.Membership, error)
	CreateOrganization(ctx context.Context, arg postgres.CreateOrganizationParams) (postgres.Organization, error)
	CreateUser(ctx context.Context, arg postgres.CreateUserParams) (postgres.User, error)
	GetAuthRefreshTokenByTokenHash(ctx context.Context, tokenHash string) (postgres.AuthRefreshToken, error)
	GetUserByEmail(ctx context.Context, email string) (postgres.User, error)
	GetUserByID(ctx context.Context, id pgtype.UUID) (postgres.User, error)
	ListOrganizationsByUser(ctx context.Context, userID pgtype.UUID) ([]postgres.ListOrganizationsByUserRow, error)
	RotateAuthRefreshToken(ctx context.Context, arg postgres.RotateAuthRefreshTokenParams) (postgres.AuthRefreshToken, error)
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
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
}

type Service struct {
	repo    Repository
	hasher  PasswordHasher
	tokens  *TokenManager
	timeNow func() time.Time
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
}

type RefreshInput struct {
	RefreshToken string
}

type UserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	DisplayName   string `json:"display_name"`
	IsSystemAdmin bool   `json:"is_system_admin"`
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

	return &Service{
		repo:    newStoreRepository(store),
		hasher:  PasswordHasher{},
		tokens:  tokenManager,
		timeNow: time.Now,
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
		OrganizationID: userOrOrgUUID(org.ID),
		UserID:         userOrOrgUUID(user.ID),
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

	if err := tx.Commit(ctx); err != nil {
		return AuthResult{}, fmt.Errorf("commit refresh tx: %w", err)
	}

	return authResult, nil
}

func (s *Service) Me(ctx context.Context, accessToken string) (AuthResult, error) {
	userID, err := s.tokens.ParseAccessToken(accessToken)
	if err != nil {
		return AuthResult{}, ErrUnauthorized
	}

	pgUserID := pgtype.UUID{}
	if err := pgUserID.Scan(userID); err != nil {
		return AuthResult{}, ErrUnauthorized
	}

	user, err := s.repo.GetUserByID(ctx, pgUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AuthResult{}, ErrUnauthorized
		}
		return AuthResult{}, fmt.Errorf("get current user: %w", err)
	}

	orgs, err := s.organizationsForUser(ctx, s.repo, user.ID)
	if err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		User:          userInfoFromModel(user),
		Organizations: orgs,
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

	return AuthResult{
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessExpiry,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshExpiry,
		User:                  userInfoFromModel(user),
		Organizations:         orgs,
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

func userInfoFromModel(user postgres.User) UserInfo {
	return UserInfo{
		ID:            user.ID.String(),
		Email:         user.Email,
		DisplayName:   user.DisplayName,
		IsSystemAdmin: user.IsSystemAdmin,
	}
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

func userOrOrgUUID(id pgtype.UUID) pgtype.UUID {
	return id
}
