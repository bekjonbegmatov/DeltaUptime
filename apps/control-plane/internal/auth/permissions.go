package auth

type Permission string

const (
	PermissionOrganizationRead Permission = "organization:read"
	PermissionMembershipRead   Permission = "membership:read"
	PermissionMembershipWrite  Permission = "membership:write"
	PermissionAPIKeyRead       Permission = "api_key:read"
	PermissionAPIKeyWrite      Permission = "api_key:write"
	PermissionAuditRead        Permission = "audit:read"
	PermissionTOTPWrite        Permission = "totp:write"
)

var rolePermissions = map[string]map[Permission]struct{}{
	"owner": {
		PermissionOrganizationRead: {},
		PermissionMembershipRead:   {},
		PermissionMembershipWrite:  {},
		PermissionAPIKeyRead:       {},
		PermissionAPIKeyWrite:      {},
		PermissionAuditRead:        {},
		PermissionTOTPWrite:        {},
	},
	"admin": {
		PermissionOrganizationRead: {},
		PermissionMembershipRead:   {},
		PermissionMembershipWrite:  {},
		PermissionAPIKeyRead:       {},
		PermissionAPIKeyWrite:      {},
		PermissionAuditRead:        {},
		PermissionTOTPWrite:        {},
	},
	"operator": {
		PermissionOrganizationRead: {},
		PermissionMembershipRead:   {},
		PermissionAuditRead:        {},
		PermissionTOTPWrite:        {},
	},
	"viewer": {
		PermissionOrganizationRead: {},
		PermissionMembershipRead:   {},
		PermissionAuditRead:        {},
		PermissionTOTPWrite:        {},
	},
	"billing": {
		PermissionOrganizationRead: {},
		PermissionAPIKeyRead:       {},
		PermissionAuditRead:        {},
		PermissionTOTPWrite:        {},
	},
}

func permissionsForRole(role string) map[Permission]struct{} {
	base, ok := rolePermissions[role]
	if !ok {
		return map[Permission]struct{}{}
	}

	permissions := make(map[Permission]struct{}, len(base))
	for permission := range base {
		permissions[permission] = struct{}{}
	}
	return permissions
}
