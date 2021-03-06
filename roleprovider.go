package kubetoken

import (
	"bytes"
	"fmt"
	"strings"

	ldap "gopkg.in/ldap.v2"
)

// ADRoleProvider speaks Active Directory flavoured LDAP to retrieve the
// roles available to a specific user.
type ADRoleProvider struct {
	LDAPCreds
}

func userdn(user string) string {
	return fmt.Sprintf(binddn(user), escapeDN(user))
}

func binddn(user string) string {
	if strings.HasSuffix(user, "-bot") {
		return "CN=%s,OU=bots,OU=people," + SearchBase
	}
	return "CN=%s,OU=people," + SearchBase
}

func (r *ADRoleProvider) FetchRolesForUser(user string) ([]string, error) {
	return fetchRolesForUser(&r.LDAPCreds, userdn(user))
}

func fetchRolesForUser(creds *LDAPCreds, userdn string) ([]string, error) {
	conn, err := creds.Bind()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// find all the kube- roles
	filter := fmt.Sprintf("(&(cn=kube-*-*-*-dl-*)(member:1.2.840.113556.1.4.1941:=%s))", userdn)
	kubeRoles := ldap.NewSearchRequest(
		"OU=access,OU=groups,"+SearchBase,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter,
		[]string{"cn"},
		nil,
	)
	sr, err := conn.Search(kubeRoles)
	if err != nil {
		return nil, err
	}

	var roles []string
	for _, e := range sr.Entries {
		role := e.GetAttributeValue("cn")
		roles = append(roles, role)
	}
	return roles, nil
}

// escapeDN returns a string with characters escaped to safely injected into a DN.
// Intended as a complement to ldap.EscapeFilter, which escapes ldap filter strings.
// Made with reference to https://www.owasp.org/index.php/LDAP_Injection_Prevention_Cheat_Sheet
// and http://www.rlmueller.net/CharactersEscaped.htm
func escapeDN(unsafe string) string {
	var buf bytes.Buffer
	for _, r := range unsafe {
		switch r {
		case '/', '\\', '#', ',', ';', '<', '>', '+', '=':
			buf.WriteRune('\\')
			fallthrough
		default:
			buf.WriteRune(r)
		}
	}
	return buf.String()
}
