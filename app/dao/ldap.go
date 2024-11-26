package dao

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/go-ldap/ldap/v3"
)

// LDAPPool represents a pool of LDAP connections
type LDAPPool struct {
	connections chan *ldap.Conn
	config      *LDAPConfig
}

type LDAPConfig struct {
	Server   string
	Port     string
	Username string
	Password string
	PoolSize int
	Timeout  time.Duration
}

// NewLDAPPool creates a new LDAP connection pool
func NewLDAPPool(config *LDAPConfig) (*LDAPPool, error) {
	if config.PoolSize <= 0 {
		config.PoolSize = 5
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	pool := &LDAPPool{
		connections: make(chan *ldap.Conn, config.PoolSize),
		config:      config,
	}

	for i := 0; i < config.PoolSize; i++ {
		conn, err := pool.createConnection()
		if err != nil {
			return nil, fmt.Errorf("failed to create initial connection: %v", err)
		}
		pool.connections <- conn
	}

	return pool, nil
}

// GetConnection gets a connection from the pool
func (p *LDAPPool) GetConnection() (*ldap.Conn, error) {
	select {
	case conn := <-p.connections:
		if err := p.validateConnection(conn); err != nil {
			_ = conn.Close()
			conn, err = p.createConnection()
			if err != nil {
				return nil, err
			}
		}
		return conn, nil
	case <-time.After(p.config.Timeout):
		return nil, fmt.Errorf("timeout waiting for connection")
	}
}

// ReleaseConnection returns a connection to the pool
func (p *LDAPPool) ReleaseConnection(conn *ldap.Conn) {
	if conn != nil {
		p.connections <- conn
	}
}

// createConnection creates a new LDAP connection with retry mechanism
func (p *LDAPPool) createConnection() (*ldap.Conn, error) {
	var conn *ldap.Conn
	var err error

	for retries := 3; retries > 0; retries-- {
		conn, err = ldap.DialURL(
			fmt.Sprintf("%s:%s", p.config.Server, p.config.Port),
			ldap.DialWithTLSConfig(&tls.Config{InsecureSkipVerify: true}),
			// ldap.DialWithTimeout(p.config.Timeout),
		)
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP server after retries: %v", err)
	}

	err = conn.Bind(p.config.Username, p.config.Password)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("LDAP bind failed: %v", err)
	}

	return conn, nil
}

// validateConnection checks if a connection is still valid
func (p *LDAPPool) validateConnection(conn *ldap.Conn) error {
	// Perform a simple search to validate the connection
	searchRequest := ldap.NewSearchRequest(
		"",
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		"(objectClass=*)",
		[]string{""},
		nil,
	)

	_, err := conn.Search(searchRequest)
	return err
}

// Search performs an LDAP search with improved error handling and attribute filtering
func Search(l *ldap.Conn, baseDN, filter string, control *ldap.ControlPaging, attributes []string) (entries []*ldap.Entry, err error) {
	if attributes == nil {
		attributes = []string{"*"} // Default to all attributes
	}

	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		int(control.PagingSize),
		false,
		filter,
		attributes,
		[]ldap.Control{control},
	)

	var sr *ldap.SearchResult
	sr, err = l.Search(searchRequest)
	if err != nil {
		if ldap.IsErrorWithCode(err, ldap.LDAPResultSizeLimitExceeded) {
			// Handle size limit exceeded gracefully
			return sr.Entries, nil
		}
		return nil, fmt.Errorf("LDAP search failed: %v", err)
	}

	for _, c := range sr.Controls {
		if ctrl, ok := c.(*ldap.ControlPaging); ok {
			control.SetCookie(ctrl.Cookie)
			break
		}
	}

	return sr.Entries, nil
}

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

func SearchOld(l *ldap.Conn, baseDN, filter string, control *ldap.ControlPaging) (entries []*ldap.Entry, err error) {
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
