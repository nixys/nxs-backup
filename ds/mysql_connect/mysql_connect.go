package mysql_connect

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"gopkg.in/ini.v1"
	"os"
)

type Params struct {
	AuthFile string // Path to auth file
	User     string // Username
	Passwd   string // Password (requires User)
	Host     string // Network host
	Port     string // Network port
	Socket   string // Socket path
	SSLCA    string
	SSLCert  string
	SSLKey   string
}

func GetConnectAndCnfFile(conn Params, sectionName string) (*sqlx.DB, *ini.File, error) {
	var withTls bool

	dumpAuthCfg := ini.Empty()
	_ = dumpAuthCfg.NewSections(sectionName)

	if conn.AuthFile != "" {
		authCfg, err := ini.LoadSources(ini.LoadOptions{AllowBooleanKeys: true}, conn.AuthFile)
		if err != nil {
			return nil, nil, err
		}

		for _, sName := range []string{"mysql", "client", "mysqldump", ""} {
			s, err := authCfg.GetSection(sName)
			if err != nil {
				continue
			}
			if user := s.Key("user").MustString(""); user != "" {
				conn.User = user
				_, _ = dumpAuthCfg.Section(sectionName).NewKey("user", user)
			}
			if pass := s.Key("password").MustString(""); pass != "" {
				conn.Passwd = pass
				_, _ = dumpAuthCfg.Section(sectionName).NewKey("password", pass)
			}
			if socket := s.Key("socket").MustString(""); socket != "" {
				conn.Socket = socket
				_, _ = dumpAuthCfg.Section(sectionName).NewKey("socket", socket)
			}
			if host := s.Key("host").MustString(""); host != "" {
				conn.Host = host
				_, _ = dumpAuthCfg.Section(sectionName).NewKey("host", host)
			}
			if port := s.Key("port").MustString(""); port != "" {
				conn.Port = port
				_, _ = dumpAuthCfg.Section(sectionName).NewKey("port", port)
			}
			if ca := s.Key("ssl-ca").MustString(""); ca != "" {
				conn.SSLCA = ca
				_, _ = dumpAuthCfg.Section(sectionName).NewKey("ssl-ca", ca)
			}
			if cert := s.Key("ssl-cert").MustString(""); cert != "" {
				conn.SSLCert = cert
				_, _ = dumpAuthCfg.Section(sectionName).NewKey("ssl-cert", cert)
			}
			if key := s.Key("ssl-key").MustString(""); key != "" {
				conn.SSLKey = key
				_, _ = dumpAuthCfg.Section(sectionName).NewKey("ssl-key", key)
			}
			break
		}
	} else {
		if conn.User != "" {
			_, _ = dumpAuthCfg.Section(sectionName).NewKey("user", conn.User)
		}
		if conn.Passwd != "" {
			_, _ = dumpAuthCfg.Section(sectionName).NewKey("password", conn.Passwd)
		}
		if conn.Socket != "" {
			_, _ = dumpAuthCfg.Section(sectionName).NewKey("socket", conn.Socket)
		}
		if conn.Host != "" {
			_, _ = dumpAuthCfg.Section(sectionName).NewKey("host", conn.Host)
		}
		if conn.Port != "" {
			_, _ = dumpAuthCfg.Section(sectionName).NewKey("port", conn.Port)
		}
		if conn.SSLCA != "" {
			_, _ = dumpAuthCfg.Section(sectionName).NewKey("ssl-ca", conn.SSLCA)
		}
		if conn.SSLCert != "" {
			_, _ = dumpAuthCfg.Section(sectionName).NewKey("ssl-cert", conn.SSLCert)
		}
		if conn.SSLKey != "" {
			_, _ = dumpAuthCfg.Section(sectionName).NewKey("ssl-key", conn.SSLKey)
		}
	}

	if conn.SSLCert != "" && conn.SSLKey != "" {
		var caCertPool *x509.CertPool

		if conn.SSLCA != "" {
			caCertPool = x509.NewCertPool()
			caCert, err := os.ReadFile(conn.SSLCA)
			if err != nil {
				return nil, dumpAuthCfg, err
			}
			if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
				return nil, dumpAuthCfg, fmt.Errorf("failed to append ca certs")
			}
			//insecure = false
		}
		cert, err := tls.LoadX509KeyPair(conn.SSLCert, conn.SSLKey)
		if err != nil {
			return nil, dumpAuthCfg, err
		}

		if err = mysql.RegisterTLSConfig("custom", &tls.Config{
			RootCAs:            caCertPool,
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true,
		}); err != nil {
			return nil, dumpAuthCfg, err
		}
		withTls = true
	}

	cfg := mysql.NewConfig()
	cfg.User = conn.User
	cfg.Passwd = conn.Passwd
	if conn.Socket != "" {
		cfg.Net = "unix"
		cfg.Addr = conn.Socket
	} else {
		cfg.Net = "tcp"
		cfg.Addr = fmt.Sprintf("%s:%s", conn.Host, conn.Port)
	}
	if withTls {
		cfg.TLSConfig = "custom"
	}

	db, err := sqlx.Connect("mysql", cfg.FormatDSN())

	return db, dumpAuthCfg, err
}
