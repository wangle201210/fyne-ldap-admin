package dao

import (
	"crypto/tls"
	"fmt"
	"github.com/go-ldap/ldap/v3"
)

func GetLdap(server, port, username, password string) (*ldap.Conn, error) {
	l, err := ldap.DialURL(fmt.Sprintf("%s:%s", server, port), ldap.DialWithTLSConfig(&tls.Config{InsecureSkipVerify: true}))
	if err != nil {
		return nil, err
	}
	_, err = l.SimpleBind(&ldap.SimpleBindRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		return nil, err
	}
	return l, nil
}

func Search(l *ldap.Conn, baseDN, filter string, control *ldap.ControlPaging) (entries []*ldap.Entry, err error) {
	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter,
		nil, // 您可以根据需要调整属性
		[]ldap.Control{control},
	)
	sr, err := l.Search(searchRequest)
	if err != nil {
		return
	}
	for _, c := range sr.Controls {
		if ctrl, ok := c.(*ldap.ControlPaging); ok {
			control.SetCookie(ctrl.Cookie)
			break
		}
	}
	entries = sr.Entries
	return
}
