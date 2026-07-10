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

func (r *storeRepository) GetAuthRefreshTokenByTokenHash(ctx context.Context, tokenHash string) (postgres.AuthRefreshToken, error) {
	return r.store.Queries.GetAuthRefreshTokenByTokenHash(ctx, tokenHash)
}

func (r *storeRepository) GetUserByEmail(ctx context.Context, email string) (postgres.User, error) {
	return r.store.Queries.GetUserByEmail(ctx, email)
}

func (r *storeRepository) GetUserByID(ctx context.Context, id pgtype.UUID) (postgres.User, error) {
	return r.store.Queries.GetUserByID(ctx, id)
}

func (r *storeRepository) ListOrganizationsByUser(ctx context.Context, userID pgtype.UUID) ([]postgres.ListOrganizationsByUserRow, error) {
	return r.store.Queries.ListOrganizationsByUser(ctx, userID)
}

func (r *storeRepository) RotateAuthRefreshToken(ctx context.Context, arg postgres.RotateAuthRefreshTokenParams) (postgres.AuthRefreshToken, error) {
	return r.store.Queries.RotateAuthRefreshToken(ctx, arg)
}

func (tx *storeTx) Commit(ctx context.Context) error {
	return tx.tx.Commit(ctx)
}

func (tx *storeTx) Rollback(ctx context.Context) error {
	return tx.tx.Rollback(ctx)
}
