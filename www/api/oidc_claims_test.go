package api

import (
	"testing"

	"github.com/dbaseqp/Quotient/engine/config"
	"golang.org/x/oauth2"

	"github.com/stretchr/testify/assert"
)

// clientRoleClaims carries a realm role and another client's role that both look
// like the admin group, and a team role under the "quotient" client.
func clientRoleClaims() *IDTokenClaims {
	var claims IDTokenClaims
	claims.PreferredUsername = "blue1"
	claims.RealmAccess.Roles = []string{"quotient-admin", "offline_access"}
	claims.ResourceAccess = map[string]struct {
		Roles []string `json:"roles"`
	}{
		"quotient": {Roles: []string{"team01"}},
		"other":    {Roles: []string{"quotient-admin"}},
	}
	return &claims
}

func clientRoleAPI(configClientID, loginClientID string) *API {
	return &API{
		conf: config.NewLive(&config.ConfigSettings{OIDCSettings: config.OIDCAuthConfig{
			OIDCEnabled:     true,
			OIDCClientID:    configClientID,
			OIDCGroupClaim:  "resource_access.roles",
			OIDCAdminGroups: []string{"quotient-admin"},
			OIDCTeamGroups:  []string{"team*"},
		}}),
		oidc: &oidcState{oauth2Config: &oauth2.Config{ClientID: loginClientID}},
	}
}

// "resource_access.roles" must read the roles of this deployment's client, not
// the realm roles: a realm role named like an admin group must not grant admin.
func TestOIDCResourceAccessClaimReadsClientRoles(t *testing.T) {
	a := clientRoleAPI("quotient", "quotient")

	username, groups, roles := a.extractUserInfoFromClaims(clientRoleClaims())
	assert.Equal(t, "blue1", username)
	assert.Equal(t, []string{"team01"}, groups)
	assert.Equal(t, []string{"team"}, roles)
}

// The token was issued to and verified against the client InitOIDC set up, so its
// roles are read under that client even after a reload renames OIDCClientID:
// reading the reloaded id would pick another client's roles, or none.
func TestOIDCResourceAccessClaimUsesLoginClientAfterReload(t *testing.T) {
	a := clientRoleAPI("other", "quotient")

	_, groups, roles := a.extractUserInfoFromClaims(clientRoleClaims())
	assert.Equal(t, []string{"team01"}, groups)
	assert.Equal(t, []string{"team"}, roles)
}
