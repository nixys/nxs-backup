package psql_connect

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Params struct {
	User        string // Username
	Passwd      string // Password (requires User)
	Host        string // Network host
	Port        string // Network port
	Socket      string // Socket path
	Database    string // Database name
	SSLMode     string // SSL mode
	SSLRootCert string // SSL root cert path
	SSLCrl      string // SSL crl path
}

func GetConnUrl(params Params) *url.URL {

	connUrl := url.URL{}
	opts := url.Values{}

	connUrl.User = url.UserPassword(params.User, params.Passwd)
	connUrl.Path = "/"

	if params.Socket != "" {
		connUrl.Host = ""
		connUrl.Scheme = "postgres"
		opts.Add("host", strings.ReplaceAll(params.Socket, "/.s.PGSQL.5432", ""))
	} else {
		connUrl.Scheme = "postgres"
		connUrl.Host = fmt.Sprintf("%s:%s", params.Host, params.Port)
	}

	opts.Add("sslmode", params.SSLMode)

	if params.SSLRootCert != "" {
		opts.Add("sslrootcert", params.SSLRootCert)
	}
	if params.SSLCrl != "" {
		opts.Add("sslcrl", params.SSLCrl)
	}
	if params.Database != "" {
		connUrl.Path += params.Database
	}

	connUrl.RawQuery = opts.Encode()

	return &connUrl
}

func GetConnect(connUrl *url.URL) (*sqlx.DB, error) {
	return sqlx.Connect("postgres", connUrl.String())
}
