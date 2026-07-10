package auth

import (
	"encoding/json"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
)

type PrincipalType string

const (
	PrincipalTypeUser   PrincipalType = "user"
	PrincipalTypeAPIKey PrincipalType = "api_key"
)

type Principal struct {
	Type              PrincipalType
	User              UserInfo
	APIKeyID          string
	OrganizationRoles map[string]string
	Permissions       map[string]map[Permission]struct{}
}

type TOTPSetup struct {
	Secret     string `json:"secret"`
	OTPAuthURL string `json:"otpauth_url"`
}

type WebAuthnRegistrationBeginResult struct {
	SessionID string                       `json:"session_id"`
	Options   *protocol.CredentialCreation `json:"options"`
}

type WebAuthnLoginBeginResult struct {
	SessionID string                        `json:"session_id"`
	Options   *protocol.CredentialAssertion `json:"options"`
}

type WebAuthnFinishInput struct {
	SessionID  string          `json:"session_id"`
	Credential json.RawMessage `json:"credential"`
}

type WebAuthnCredentialInfo struct {
	ID         string    `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at,omitempty"`
}

type APIKeyInfo struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Prefix          string    `json:"prefix"`
	Scopes          []string  `json:"scopes"`
	LastUsedAt      time.Time `json:"last_used_at,omitempty"`
	RevokedAt       time.Time `json:"revoked_at,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	CreatedByUserID string    `json:"created_by_user_id,omitempty"`
}

type APIKeyCreateResult struct {
	APIKeyInfo
	Token string `json:"token"`
}

type AuditLogInfo struct {
	ID            string          `json:"id"`
	ActorType     string          `json:"actor_type"`
	ActorUserID   string          `json:"actor_user_id,omitempty"`
	ActorAPIKeyID string          `json:"actor_api_key_id,omitempty"`
	Action        string          `json:"action"`
	TargetType    string          `json:"target_type"`
	TargetID      string          `json:"target_id"`
	Metadata      json.RawMessage `json:"metadata"`
	OccurredAt    time.Time       `json:"occurred_at"`
}

type MembershipInfo struct {
	User UserInfo `json:"user"`
	Role string   `json:"role"`
}
