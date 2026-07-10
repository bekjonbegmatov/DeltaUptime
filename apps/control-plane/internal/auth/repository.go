package auth

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"deltauptime/apps/control-plane/internal/database"
	"deltauptime/packages/database/postgres"
)

type storeRepository struct {
	store *database.Store
}

type storeTx struct {
	*postgres.Queries
	tx pgx.Tx
}

func newStoreRepository(store *database.Store) Repository {
	return &storeRepository{store: store}
}

func (r *storeRepository) Begin(ctx context.Context) (Tx, error) {
	tx, err := r.store.Pool().BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin pgx tx: %w", err)
	}
	return &storeTx{
		Queries: postgres.New(tx),
		tx:      tx,
	}, nil
}

func (r *storeRepository) CreateAPIKey(ctx context.Context, arg postgres.CreateAPIKeyParams) (postgres.ApiKey, error) {
	return r.store.Queries.CreateAPIKey(ctx, arg)
}

func (r *storeRepository) CreateAuthWebAuthnCredential(ctx context.Context, arg postgres.CreateAuthWebAuthnCredentialParams) (postgres.AuthWebauthnCredential, error) {
	return r.store.Queries.CreateAuthWebAuthnCredential(ctx, arg)
}

func (r *storeRepository) CreateAuthWebAuthnSession(ctx context.Context, arg postgres.CreateAuthWebAuthnSessionParams) (postgres.AuthWebauthnSession, error) {
	return r.store.Queries.CreateAuthWebAuthnSession(ctx, arg)
}

func (r *storeRepository) CreateAuditLog(ctx context.Context, arg postgres.CreateAuditLogParams) (postgres.AuditLog, error) {
	return r.store.Queries.CreateAuditLog(ctx, arg)
}

func (r *storeRepository) CreateAuthRefreshToken(ctx context.Context, arg postgres.CreateAuthRefreshTokenParams) (postgres.AuthRefreshToken, error) {
	return r.store.Queries.CreateAuthRefreshToken(ctx, arg)
}

func (r *storeRepository) CreateMembership(ctx context.Context, arg postgres.CreateMembershipParams) (postgres.Membership, error) {
	return r.store.Queries.CreateMembership(ctx, arg)
}

func (r *storeRepository) CreateOrganization(ctx context.Context, arg postgres.CreateOrganizationParams) (postgres.Organization, error) {
	return r.store.Queries.CreateOrganization(ctx, arg)
}

func (r *storeRepository) CreateUser(ctx context.Context, arg postgres.CreateUserParams) (postgres.User, error) {
	return r.store.Queries.CreateUser(ctx, arg)
}

func (r *storeRepository) DeleteAuthWebAuthnSession(ctx context.Context, id pgtype.UUID) error {
	return r.store.Queries.DeleteAuthWebAuthnSession(ctx, id)
}

func (r *storeRepository) DeleteUserTOTPCredential(ctx context.Context, userID pgtype.UUID) error {
	return r.store.Queries.DeleteUserTOTPCredential(ctx, userID)
}

func (r *storeRepository) EnableUserTOTPCredential(ctx context.Context, userID pgtype.UUID) (postgres.UserTotpCredential, error) {
	return r.store.Queries.EnableUserTOTPCredential(ctx, userID)
}

func (r *storeRepository) GetAPIKeyByID(ctx context.Context, id pgtype.UUID) (postgres.ApiKey, error) {
	return r.store.Queries.GetAPIKeyByID(ctx, id)
}

func (r *storeRepository) GetAPIKeyByPrefix(ctx context.Context, keyPrefix string) (postgres.ApiKey, error) {
	return r.store.Queries.GetAPIKeyByPrefix(ctx, keyPrefix)
}

func (r *storeRepository) GetAuthRefreshTokenByTokenHash(ctx context.Context, tokenHash string) (postgres.AuthRefreshToken, error) {
	return r.store.Queries.GetAuthRefreshTokenByTokenHash(ctx, tokenHash)
}

func (r *storeRepository) GetAuthWebAuthnSessionByID(ctx context.Context, id pgtype.UUID) (postgres.AuthWebauthnSession, error) {
	return r.store.Queries.GetAuthWebAuthnSessionByID(ctx, id)
}

func (r *storeRepository) GetMembership(ctx context.Context, arg postgres.GetMembershipParams) (postgres.Membership, error) {
	return r.store.Queries.GetMembership(ctx, arg)
}

func (r *storeRepository) GetOrganizationByID(ctx context.Context, id pgtype.UUID) (postgres.Organization, error) {
	return r.store.Queries.GetOrganizationByID(ctx, id)
}

func (r *storeRepository) GetOrganizationBySlug(ctx context.Context, slug string) (postgres.Organization, error) {
	return r.store.Queries.GetOrganizationBySlug(ctx, slug)
}

func (r *storeRepository) GetUserByEmail(ctx context.Context, email string) (postgres.User, error) {
	return r.store.Queries.GetUserByEmail(ctx, email)
}

func (r *storeRepository) GetUserByID(ctx context.Context, id pgtype.UUID) (postgres.User, error) {
	return r.store.Queries.GetUserByID(ctx, id)
}

func (r *storeRepository) GetUserTOTPCredentialByUserID(ctx context.Context, userID pgtype.UUID) (postgres.UserTotpCredential, error) {
	return r.store.Queries.GetUserTOTPCredentialByUserID(ctx, userID)
}

func (r *storeRepository) ListAPIKeysByOrganization(ctx context.Context, organizationID pgtype.UUID) ([]postgres.ApiKey, error) {
	return r.store.Queries.ListAPIKeysByOrganization(ctx, organizationID)
}

func (r *storeRepository) ListAuditLogsByOrganization(ctx context.Context, arg postgres.ListAuditLogsByOrganizationParams) ([]postgres.AuditLog, error) {
	return r.store.Queries.ListAuditLogsByOrganization(ctx, arg)
}

func (r *storeRepository) ListAuthWebAuthnCredentialsByUserID(ctx context.Context, userID pgtype.UUID) ([]postgres.AuthWebauthnCredential, error) {
	return r.store.Queries.ListAuthWebAuthnCredentialsByUserID(ctx, userID)
}

func (r *storeRepository) ListOrganizationsByUser(ctx context.Context, userID pgtype.UUID) ([]postgres.ListOrganizationsByUserRow, error) {
	return r.store.Queries.ListOrganizationsByUser(ctx, userID)
}

func (r *storeRepository) ListUsersByOrganization(ctx context.Context, organizationID pgtype.UUID) ([]postgres.ListUsersByOrganizationRow, error) {
	return r.store.Queries.ListUsersByOrganization(ctx, organizationID)
}

func (r *storeRepository) RevokeAPIKey(ctx context.Context, id pgtype.UUID) (postgres.ApiKey, error) {
	return r.store.Queries.RevokeAPIKey(ctx, id)
}

func (r *storeRepository) RotateAuthRefreshToken(ctx context.Context, arg postgres.RotateAuthRefreshTokenParams) (postgres.AuthRefreshToken, error) {
	return r.store.Queries.RotateAuthRefreshToken(ctx, arg)
}

func (r *storeRepository) SetUserWebAuthnHandle(ctx context.Context, arg postgres.SetUserWebAuthnHandleParams) (postgres.User, error) {
	return r.store.Queries.SetUserWebAuthnHandle(ctx, arg)
}

func (r *storeRepository) TouchAPIKeyLastUsed(ctx context.Context, id pgtype.UUID) (postgres.ApiKey, error) {
	return r.store.Queries.TouchAPIKeyLastUsed(ctx, id)
}

func (r *storeRepository) UpdateAuthWebAuthnCredential(ctx context.Context, arg postgres.UpdateAuthWebAuthnCredentialParams) (postgres.AuthWebauthnCredential, error) {
	return r.store.Queries.UpdateAuthWebAuthnCredential(ctx, arg)
}

func (r *storeRepository) UpdateMembershipRole(ctx context.Context, arg postgres.UpdateMembershipRoleParams) (postgres.Membership, error) {
	return r.store.Queries.UpdateMembershipRole(ctx, arg)
}

func (r *storeRepository) UpsertUserTOTPCredential(ctx context.Context, arg postgres.UpsertUserTOTPCredentialParams) (postgres.UserTotpCredential, error) {
	return r.store.Queries.UpsertUserTOTPCredential(ctx, arg)
}

func (tx *storeTx) Commit(ctx context.Context) error {
	return tx.tx.Commit(ctx)
}

func (tx *storeTx) Rollback(ctx context.Context) error {
	return tx.tx.Rollback(ctx)
}
