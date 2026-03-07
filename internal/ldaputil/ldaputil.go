package ldaputil

import (
	"fmt"

	"github.com/go-ldap/ldap/v3"
)

// Connect establishes an LDAP connection and binds with the given credentials.
func Connect(url, bindDN, bindPassword string) (*ldap.Conn, error) {
	conn, err := ldap.DialURL(url)
	if err != nil {
		return nil, fmt.Errorf("LDAP dial failed: %w", err)
	}
	if err := conn.Bind(bindDN, bindPassword); err != nil {
		conn.Close()
		return nil, fmt.Errorf("LDAP bind failed: %w", err)
	}
	return conn, nil
}

// SearchUsers searches for users matching the given filter under baseDN, returning the requested attributes.
func SearchUsers(conn *ldap.Conn, baseDN, filter string, attributes []string) ([]*ldap.Entry, error) {
	req := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter,
		attributes,
		nil,
	)
	sr, err := conn.Search(req)
	if err != nil {
		return nil, err
	}
	return sr.Entries, nil
}
