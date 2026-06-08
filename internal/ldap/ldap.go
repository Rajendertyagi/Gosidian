// Package ldap authenticates web-login users against an external LDAP / Active
// Directory directory using a search-then-bind flow: bind as a service
// account, search for the user's DN, then re-bind as that DN with the supplied
// password. It deliberately exposes only Authenticate — gosidian keeps its own
// account records (auto-provisioned on first LDAP login), so no attribute
// mapping beyond locating the entry is needed.
package ldap

import (
	"crypto/tls"
	"errors"
	"fmt"

	goldap "github.com/go-ldap/ldap/v3"
)

// Config holds the [ldap] settings required to authenticate. UserFilter must
// contain exactly one %s placeholder for the (escaped) username.
type Config struct {
	URL          string
	StartTLS     bool
	SkipVerify   bool
	BindDN       string
	BindPassword string
	UserBaseDN   string
	UserFilter   string
}

// Client authenticates against a directory. Build with New; the zero value is
// not usable. A nil *Client is safe to call and reports "disabled".
type Client struct {
	cfg Config
}

// New validates the config and returns a Client. It does not dial — connection
// errors surface lazily on Authenticate so a directory outage doesn't block
// startup (local auth keeps working).
func New(cfg Config) (*Client, error) {
	if cfg.URL == "" {
		return nil, errors.New("ldap: url required")
	}
	if cfg.UserBaseDN == "" {
		return nil, errors.New("ldap: user_base_dn required")
	}
	if cfg.UserFilter == "" {
		cfg.UserFilter = "(uid=%s)"
	}
	return &Client{cfg: cfg}, nil
}

func (c *Client) dial() (*goldap.Conn, error) {
	tlsCfg := &tls.Config{InsecureSkipVerify: c.cfg.SkipVerify} //nolint:gosec // opt-in, dev-only
	conn, err := goldap.DialURL(c.cfg.URL, goldap.DialWithTLSConfig(tlsCfg))
	if err != nil {
		return nil, err
	}
	if c.cfg.StartTLS {
		if err := conn.StartTLS(tlsCfg); err != nil {
			conn.Close()
			return nil, err
		}
	}
	return conn, nil
}

// Authenticate verifies username+password against the directory. Returns nil on
// success. Transport/config failures are wrapped (so callers can log them) and
// kept distinct from a clean credential rejection; the login handler maps both
// to a generic 401 for the end user.
func (c *Client) Authenticate(username, password string) error {
	if c == nil {
		return errors.New("ldap: disabled")
	}
	if username == "" || password == "" {
		return errors.New("ldap: empty credentials")
	}

	conn, err := c.dial()
	if err != nil {
		return fmt.Errorf("ldap: dial: %w", err)
	}
	defer conn.Close()

	// 1) Bind as the service account (anonymous if no bind_dn) to search.
	if c.cfg.BindDN != "" {
		if err := conn.Bind(c.cfg.BindDN, c.cfg.BindPassword); err != nil {
			return fmt.Errorf("ldap: service bind: %w", err)
		}
	}

	// 2) Locate the user's DN. SizeLimit 2 so an ambiguous filter is rejected.
	filter := fmt.Sprintf(c.cfg.UserFilter, goldap.EscapeFilter(username))
	res, err := conn.Search(goldap.NewSearchRequest(
		c.cfg.UserBaseDN, goldap.ScopeWholeSubtree, goldap.NeverDerefAliases,
		2, 0, false, filter, []string{"dn"}, nil,
	))
	if err != nil {
		return fmt.Errorf("ldap: search: %w", err)
	}
	if len(res.Entries) != 1 {
		return errors.New("ldap: user not found or ambiguous")
	}
	userDN := res.Entries[0].DN

	// 3) Re-bind as the user on a fresh connection to verify the password.
	uconn, err := c.dial()
	if err != nil {
		return fmt.Errorf("ldap: dial(user): %w", err)
	}
	defer uconn.Close()
	if err := uconn.Bind(userDN, password); err != nil {
		return errors.New("ldap: invalid credentials")
	}
	return nil
}
