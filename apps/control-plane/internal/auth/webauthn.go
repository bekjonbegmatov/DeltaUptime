package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"deltauptime/packages/database/postgres"
)

const (
	webAuthnFlowRegistration = "registration"
	webAuthnFlowLogin        = "login"
)

type webAuthnUser struct {
	id          []byte
	name        string
	displayName string
	credentials []webauthn.Credential
}

func newWebAuthnManager(publicPanelURL string) (*webauthn.WebAuthn, error) {
	parsed, err := url.Parse(strings.TrimSpace(publicPanelURL))
	if err != nil {
		return nil, fmt.Errorf("parse PUBLIC_PANEL_URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Hostname() == "" {
		return nil, fmt.Errorf("%w: PUBLIC_PANEL_URL must include scheme and host", ErrInvalidInput)
	}

	origin := parsed.Scheme + "://" + parsed.Host
	return webauthn.New(&webauthn.Config{
		RPID:          parsed.Hostname(),
		RPDisplayName: "DeltaUptime",
		RPOrigins:     []string{origin},
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementPreferred,
			UserVerification: protocol.VerificationRequired,
		},
		AttestationPreference: protocol.PreferNoAttestation,
	})
}

func (s *Service) BeginWebAuthnRegistration(ctx context.Context, principal Principal) (WebAuthnRegistrationBeginResult, error) {
	if principal.Type != PrincipalTypeUser {
		return WebAuthnRegistrationBeginResult{}, ErrUnauthorized
	}

	userID, err := parseUUID(principal.User.ID)
	if err != nil {
		return WebAuthnRegistrationBeginResult{}, err
	}

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return WebAuthnRegistrationBeginResult{}, fmt.Errorf("begin webauthn registration tx: %w", err)
	}
	defer rollbackTx(ctx, tx)

	user, err := tx.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WebAuthnRegistrationBeginResult{}, ErrUnauthorized
		}
		return WebAuthnRegistrationBeginResult{}, fmt.Errorf("get webauthn registration user: %w", err)
	}

	user, err = s.ensureWebAuthnUserHandle(ctx, tx, user)
	if err != nil {
		return WebAuthnRegistrationBeginResult{}, err
	}

	waUser, _, err := s.webAuthnUserFromModel(ctx, tx, user)
	if err != nil {
		return WebAuthnRegistrationBeginResult{}, err
	}

	options, sessionData, err := s.webauthn.BeginRegistration(waUser)
	if err != nil {
		return WebAuthnRegistrationBeginResult{}, fmt.Errorf("begin webauthn registration: %w", err)
	}

	payload, err := json.Marshal(sessionData)
	if err != nil {
		return WebAuthnRegistrationBeginResult{}, fmt.Errorf("marshal webauthn registration session: %w", err)
	}

	session, err := tx.CreateAuthWebAuthnSession(ctx, postgres.CreateAuthWebAuthnSessionParams{
		UserID:      user.ID,
		Flow:        webAuthnFlowRegistration,
		SessionData: payload,
		ExpiresAt:   pgtype.Timestamptz{Time: sessionData.Expires, Valid: true},
	})
	if err != nil {
		return WebAuthnRegistrationBeginResult{}, fmt.Errorf("create webauthn registration session: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return WebAuthnRegistrationBeginResult{}, fmt.Errorf("commit webauthn registration tx: %w", err)
	}

	return WebAuthnRegistrationBeginResult{
		SessionID: session.ID.String(),
		Options:   options,
	}, nil
}

func (s *Service) FinishWebAuthnRegistration(ctx context.Context, principal Principal, input WebAuthnFinishInput) (WebAuthnCredentialInfo, error) {
	if principal.Type != PrincipalTypeUser {
		return WebAuthnCredentialInfo{}, ErrUnauthorized
	}
	if len(input.Credential) == 0 {
		return WebAuthnCredentialInfo{}, fmt.Errorf("%w: credential is required", ErrInvalidInput)
	}

	userID, err := parseUUID(principal.User.ID)
	if err != nil {
		return WebAuthnCredentialInfo{}, err
	}
	sessionID, err := parseUUID(input.SessionID)
	if err != nil {
		return WebAuthnCredentialInfo{}, err
	}

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return WebAuthnCredentialInfo{}, fmt.Errorf("begin webauthn registration finish tx: %w", err)
	}
	defer rollbackTx(ctx, tx)

	session, err := tx.GetAuthWebAuthnSessionByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WebAuthnCredentialInfo{}, ErrUnauthorized
		}
		return WebAuthnCredentialInfo{}, fmt.Errorf("get webauthn registration session: %w", err)
	}
	if session.Flow != webAuthnFlowRegistration || session.UserID != userID || !session.ExpiresAt.Valid || s.timeNow().After(session.ExpiresAt.Time) {
		return WebAuthnCredentialInfo{}, ErrUnauthorized
	}

	user, err := tx.GetUserByID(ctx, userID)
	if err != nil {
		return WebAuthnCredentialInfo{}, fmt.Errorf("get webauthn registration finish user: %w", err)
	}

	waUser, _, err := s.webAuthnUserFromModel(ctx, tx, user)
	if err != nil {
		return WebAuthnCredentialInfo{}, err
	}

	var sessionData webauthn.SessionData
	if err := json.Unmarshal(session.SessionData, &sessionData); err != nil {
		return WebAuthnCredentialInfo{}, fmt.Errorf("unmarshal webauthn registration session: %w", err)
	}

	parsedResponse, err := protocol.ParseCredentialCreationResponseBytes(input.Credential)
	if err != nil {
		return WebAuthnCredentialInfo{}, fmt.Errorf("%w: parse webauthn registration response", ErrInvalidInput)
	}

	credential, err := s.webauthn.CreateCredential(waUser, sessionData, parsedResponse)
	if err != nil {
		return WebAuthnCredentialInfo{}, fmt.Errorf("%w: validate webauthn registration", ErrInvalidInput)
	}

	serializedCredential, err := json.Marshal(credential)
	if err != nil {
		return WebAuthnCredentialInfo{}, fmt.Errorf("marshal webauthn credential: %w", err)
	}
	ciphertext, nonce, err := s.secrets.Encrypt(serializedCredential)
	if err != nil {
		return WebAuthnCredentialInfo{}, fmt.Errorf("encrypt webauthn credential: %w", err)
	}

	storedCredential, err := tx.CreateAuthWebAuthnCredential(ctx, postgres.CreateAuthWebAuthnCredentialParams{
		UserID:               user.ID,
		CredentialID:         credential.ID,
		CredentialCiphertext: ciphertext,
		CredentialNonce:      nonce,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return WebAuthnCredentialInfo{}, fmt.Errorf("%w: credential already registered", ErrInvalidInput)
		}
		return WebAuthnCredentialInfo{}, fmt.Errorf("store webauthn credential: %w", err)
	}

	if err := tx.DeleteAuthWebAuthnSession(ctx, session.ID); err != nil {
		return WebAuthnCredentialInfo{}, fmt.Errorf("delete webauthn registration session: %w", err)
	}

	if err := s.audit(ctx, tx, auditEntry{
		ActorType:   string(PrincipalTypeUser),
		ActorUserID: user.ID,
		Action:      "auth.webauthn.register",
		TargetType:  "webauthn_credential",
		TargetID:    storedCredential.ID.String(),
		Metadata: map[string]any{
			"credential_id": credentialIDString(storedCredential.CredentialID),
		},
	}); err != nil {
		return WebAuthnCredentialInfo{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return WebAuthnCredentialInfo{}, fmt.Errorf("commit webauthn registration finish tx: %w", err)
	}

	return webAuthnCredentialInfoFromModel(storedCredential), nil
}

func (s *Service) BeginWebAuthnLogin(ctx context.Context, email string) (WebAuthnLoginBeginResult, error) {
	email = normalizeEmail(email)
	if email == "" {
		return WebAuthnLoginBeginResult{}, fmt.Errorf("%w: email is required", ErrInvalidInput)
	}

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WebAuthnLoginBeginResult{}, ErrInvalidCredentials
		}
		return WebAuthnLoginBeginResult{}, fmt.Errorf("get webauthn login user: %w", err)
	}

	waUser, _, err := s.webAuthnUserFromModel(ctx, s.repo, user)
	if err != nil {
		return WebAuthnLoginBeginResult{}, err
	}
	if len(waUser.credentials) == 0 {
		return WebAuthnLoginBeginResult{}, ErrInvalidCredentials
	}

	options, sessionData, err := s.webauthn.BeginLogin(waUser, webauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		return WebAuthnLoginBeginResult{}, fmt.Errorf("begin webauthn login: %w", err)
	}

	payload, err := json.Marshal(sessionData)
	if err != nil {
		return WebAuthnLoginBeginResult{}, fmt.Errorf("marshal webauthn login session: %w", err)
	}

	session, err := s.repo.CreateAuthWebAuthnSession(ctx, postgres.CreateAuthWebAuthnSessionParams{
		UserID:      user.ID,
		Flow:        webAuthnFlowLogin,
		SessionData: payload,
		ExpiresAt:   pgtype.Timestamptz{Time: sessionData.Expires, Valid: true},
	})
	if err != nil {
		return WebAuthnLoginBeginResult{}, fmt.Errorf("create webauthn login session: %w", err)
	}

	return WebAuthnLoginBeginResult{
		SessionID: session.ID.String(),
		Options:   options,
	}, nil
}

func (s *Service) FinishWebAuthnLogin(ctx context.Context, input WebAuthnFinishInput) (AuthResult, error) {
	if len(input.Credential) == 0 {
		return AuthResult{}, fmt.Errorf("%w: credential is required", ErrInvalidInput)
	}

	sessionID, err := parseUUID(input.SessionID)
	if err != nil {
		return AuthResult{}, err
	}

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return AuthResult{}, fmt.Errorf("begin webauthn login finish tx: %w", err)
	}
	defer rollbackTx(ctx, tx)

	session, err := tx.GetAuthWebAuthnSessionByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AuthResult{}, ErrInvalidCredentials
		}
		return AuthResult{}, fmt.Errorf("get webauthn login session: %w", err)
	}
	if session.Flow != webAuthnFlowLogin || !session.ExpiresAt.Valid || s.timeNow().After(session.ExpiresAt.Time) {
		return AuthResult{}, ErrInvalidCredentials
	}

	user, err := tx.GetUserByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AuthResult{}, ErrInvalidCredentials
		}
		return AuthResult{}, fmt.Errorf("get webauthn login user: %w", err)
	}

	waUser, storedCredentials, err := s.webAuthnUserFromModel(ctx, tx, user)
	if err != nil {
		return AuthResult{}, err
	}
	if len(waUser.credentials) == 0 {
		return AuthResult{}, ErrInvalidCredentials
	}

	var sessionData webauthn.SessionData
	if err := json.Unmarshal(session.SessionData, &sessionData); err != nil {
		return AuthResult{}, fmt.Errorf("unmarshal webauthn login session: %w", err)
	}

	parsedResponse, err := protocol.ParseCredentialRequestResponseBytes(input.Credential)
	if err != nil {
		return AuthResult{}, fmt.Errorf("%w: parse webauthn login response", ErrInvalidInput)
	}

	credential, err := s.webauthn.ValidateLogin(waUser, sessionData, parsedResponse)
	if err != nil {
		return AuthResult{}, ErrInvalidCredentials
	}

	storedCredential, ok := storedCredentials[string(credential.ID)]
	if !ok {
		return AuthResult{}, ErrInvalidCredentials
	}

	serializedCredential, err := json.Marshal(credential)
	if err != nil {
		return AuthResult{}, fmt.Errorf("marshal updated webauthn credential: %w", err)
	}
	ciphertext, nonce, err := s.secrets.Encrypt(serializedCredential)
	if err != nil {
		return AuthResult{}, fmt.Errorf("encrypt updated webauthn credential: %w", err)
	}

	if _, err := tx.UpdateAuthWebAuthnCredential(ctx, postgres.UpdateAuthWebAuthnCredentialParams{
		ID:                   storedCredential.ID,
		CredentialCiphertext: ciphertext,
		CredentialNonce:      nonce,
	}); err != nil {
		return AuthResult{}, fmt.Errorf("update webauthn credential: %w", err)
	}

	orgs, err := s.organizationsForUser(ctx, tx, user.ID)
	if err != nil {
		return AuthResult{}, err
	}

	authResult, err := s.issueSession(ctx, tx, user, orgs)
	if err != nil {
		return AuthResult{}, err
	}

	if err := tx.DeleteAuthWebAuthnSession(ctx, session.ID); err != nil {
		return AuthResult{}, fmt.Errorf("delete webauthn login session: %w", err)
	}

	if err := s.audit(ctx, tx, auditEntry{
		ActorType:   string(PrincipalTypeUser),
		ActorUserID: user.ID,
		Action:      "auth.webauthn.login",
		TargetType:  "webauthn_credential",
		TargetID:    storedCredential.ID.String(),
		Metadata: map[string]any{
			"credential_id": credentialIDString(storedCredential.CredentialID),
		},
	}); err != nil {
		return AuthResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return AuthResult{}, fmt.Errorf("commit webauthn login finish tx: %w", err)
	}

	return authResult, nil
}

func (u webAuthnUser) WebAuthnID() []byte {
	return append([]byte(nil), u.id...)
}

func (u webAuthnUser) WebAuthnName() string {
	return u.name
}

func (u webAuthnUser) WebAuthnDisplayName() string {
	return u.displayName
}

func (u webAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return append([]webauthn.Credential(nil), u.credentials...)
}

func (s *Service) ensureWebAuthnUserHandle(ctx context.Context, repo QueryRepository, user postgres.User) (postgres.User, error) {
	if len(user.WebauthnUserHandle) > 0 {
		return user, nil
	}

	handle := make([]byte, 32)
	if _, err := rand.Read(handle); err != nil {
		return postgres.User{}, fmt.Errorf("generate webauthn user handle: %w", err)
	}

	updatedUser, err := repo.SetUserWebAuthnHandle(ctx, postgres.SetUserWebAuthnHandleParams{
		ID:                 user.ID,
		WebauthnUserHandle: handle,
	})
	if err != nil {
		return postgres.User{}, fmt.Errorf("set webauthn user handle: %w", err)
	}
	return updatedUser, nil
}

func (s *Service) webAuthnUserFromModel(ctx context.Context, repo QueryRepository, user postgres.User) (webAuthnUser, map[string]postgres.AuthWebauthnCredential, error) {
	user, err := s.ensureWebAuthnUserHandle(ctx, repo, user)
	if err != nil {
		return webAuthnUser{}, nil, err
	}

	rows, err := repo.ListAuthWebAuthnCredentialsByUserID(ctx, user.ID)
	if err != nil {
		return webAuthnUser{}, nil, fmt.Errorf("list webauthn credentials: %w", err)
	}

	credentials := make([]webauthn.Credential, 0, len(rows))
	stored := make(map[string]postgres.AuthWebauthnCredential, len(rows))
	for _, row := range rows {
		plaintext, err := s.secrets.Decrypt(row.CredentialCiphertext, row.CredentialNonce)
		if err != nil {
			return webAuthnUser{}, nil, fmt.Errorf("decrypt webauthn credential: %w", err)
		}

		var credential webauthn.Credential
		if err := json.Unmarshal(plaintext, &credential); err != nil {
			return webAuthnUser{}, nil, fmt.Errorf("unmarshal webauthn credential: %w", err)
		}

		credentials = append(credentials, credential)
		stored[string(row.CredentialID)] = row
	}

	return webAuthnUser{
		id:          append([]byte(nil), user.WebauthnUserHandle...),
		name:        user.Email,
		displayName: user.DisplayName,
		credentials: credentials,
	}, stored, nil
}

func webAuthnCredentialInfoFromModel(model postgres.AuthWebauthnCredential) WebAuthnCredentialInfo {
	info := WebAuthnCredentialInfo{
		ID:        credentialIDString(model.CredentialID),
		CreatedAt: model.CreatedAt.Time,
	}
	if model.LastUsedAt.Valid {
		info.LastUsedAt = model.LastUsedAt.Time
	}
	return info
}

func credentialIDString(id []byte) string {
	return base64.RawURLEncoding.EncodeToString(id)
}
