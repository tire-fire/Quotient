package api

import (
	"testing"

	"github.com/dbaseqp/Quotient/engine/config"

	"github.com/stretchr/testify/assert"
)

// "resource_access.roles" must read the roles of the client that verified the
// token, not the realm roles: a realm role named like an admin group must not
// grant admin. A reload that changes OIDCClientID does not change that client.
func TestOIDCResourceAccessClaimReadsClientRoles(t *testing.T) {
	conf := &config.ConfigSettings{OIDCSettings: config.OIDCAuthConfig{
		OIDCEnabled:     true,
		OIDCClientID:    "other",
		OIDCGroupClaim:  "resource_access.roles",
		OIDCAdminGroups: []string{"quotient-admin"},
		OIDCTeamGroups:  []string{"team*"},
	}}

	var claims IDTokenClaims
	claims.PreferredUsername = "blue1"
	claims.RealmAccess.Roles = []string{"quotient-admin", "offline_access"}
	claims.ResourceAccess = map[string]struct {
		Roles []string `json:"roles"`
	}{
		"quotient": {Roles: []string{"team01"}},
		"other":    {Roles: []string{"quotient-admin"}},
	}

	username, groups, roles := extractUserInfoFromClaims(conf, "quotient", &claims)
	assert.Equal(t, "blue1", username)
	assert.Equal(t, []string{"team01"}, groups)
	assert.Equal(t, []string{"team"}, roles)
}
