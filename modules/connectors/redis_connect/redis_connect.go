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

	connUrl.User = url.UserPassword("", params.Passwd)

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
	rdb = redis.NewClient(opt)

	err = rdb.Ping(context.Background()).Err()

	// this is not a bug, this is strange behavior of redis-cli uri usage
	connUrl.User = url.User(params.Passwd)
	dsn = connUrl.String()

	return
}
