package mongo_connect

import (
	"context"
	"fmt"
	"net/url"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Params struct {
	User   string // Username
	Passwd string // Password (requires User)
	Host   string // Network host
	Port   string // Network port
	RSName string // Replica set name
	RSAddr string // Replica set address (requires RSName)
}

// GetConnectAndHost returns connect to mongo instance and dsn string
func GetConnectAndHost(params Params) (*mongo.Client, string, error) {
	var host string
	connUrl := url.URL{}
	opts := url.Values{}

	connUrl.User = url.UserPassword(params.User, params.Passwd)

	connUrl.Scheme = "mongodb"
	connUrl.Path = "/"

	if params.RSAddr != "" {
		connUrl.Host = params.RSAddr
		opts.Set("replicaSet", params.RSName)
		host = params.RSName + `/` + params.RSAddr
	} else {
		host = fmt.Sprintf("%s:%s", params.Host, params.Port)
		connUrl.Host = host
	}

	connUrl.RawQuery = opts.Encode()

	dsn := connUrl.String()

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(dsn))
	if err != nil {
		return nil, "", err
	}
	if err = client.Ping(context.TODO(), readpref.Primary()); err != nil {
		return nil, "", err
	}

	return client, host, err
}
