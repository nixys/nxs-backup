package redis_connect

import (
	"context"
	"net/url"

	"github.com/go-redis/redis/v8"
)

type Params struct {
	Passwd string // Password
	Host   string // Network host
	Port   string // Network port
	Socket string // Socket path
}

// GetConnectAndDSN returns connect to mongo instance and dsn string
func GetConnectAndDSN(params Params) (rdb *redis.Client, dsn string, err error) {

	connUrl := url.URL{}
	opts := url.Values{}

	if params.Socket != "" {
		connUrl.Scheme = "unix"
		connUrl.Path = params.Socket
	} else {
		connUrl.Scheme = "redis"
		var host string
		if params.Host != "" {
			host = params.Host
		}
		if params.Port != "" {
			host += ":" + params.Port
		}
		connUrl.Host = host
	}

	connUrl.RawQuery = opts.Encode()

	var opt *redis.Options
	opt, err = redis.ParseURL(connUrl.String())
	if err != nil {
		return
	}
	opt.Password = params.Passwd
	rdb = redis.NewClient(opt)

	err = rdb.Ping(context.Background()).Err()

	// redis-cli treats a single userinfo segment (no colon) as a legacy,
	// version-agnostic `AUTH <password>`, whereas a `user:password` segment
	// makes it send a 2-arg `AUTH user password` that pre-6.0 Redis (no ACL
	// support) rejects. So the DSN below intentionally differs from how
	// opt.Password above is set for the go-redis client.
	if params.Passwd != "" {
		connUrl.User = url.User(params.Passwd)
	}
	dsn = connUrl.String()

	return
}
