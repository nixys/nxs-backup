package psql_connect

import (
	"fmt"
	"net/url"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Params struct {
	User    string // Username
	Passwd  string // Password (requires User)
	Host    string // Network host
	Port    string // Network port
	Socket  string // Socket path
	SSLMode string // Socket path
}

func GetConnect(params Params) (*sqlx.DB, *url.URL, error) {

	connUrl := url.URL{}
	opts := url.Values{}

	connUrl.User = url.UserPassword(params.User, params.Passwd)
	if params.Socket != "" {
		connUrl.Scheme = "unix"
		connUrl.Host = params.Socket
	} else {
		connUrl.Scheme = "postgres"
		connUrl.Host = fmt.Sprintf("%s:%s", params.Host, params.Port)
	}

	opts.Add("sslmode", params.SSLMode)
	connUrl.RawQuery = opts.Encode()

	db, err := sqlx.Connect("postgres", connUrl.String())

	return db, &connUrl, err
}
