package mysql_connect

import (
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"gopkg.in/ini.v1"
)

type Params struct {
	AuthFile string // Path to auth file
	User     string // Username
	Passwd   string // Password (requires User)
	Host     string // Network host
	Port     string // Network port
	Socket   string // Socket path
}

func GetConnectAndCnfFile(conn Params, sectionName string) (*sqlx.DB, *ini.File, error) {

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

	db, err := sqlx.Connect("mysql", cfg.FormatDSN())

	return db, dumpAuthCfg, err
}
