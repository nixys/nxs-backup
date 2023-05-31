package ctx

import (
	"path"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-multierror"
	conf "github.com/nixys/nxs-go-conf"

	"nxs-backup/misc"
)

type confOpts struct {
	ProjectName     string           `conf:"project_name"`
	ServerName      string           `conf:"server_name" conf_extraopts:"default=localhost"`
	Notifications   notifications    `conf:"notifications"`
	Jobs            []jobCfg         `conf:"jobs"`
	StorageConnects []storageConnect `conf:"storage_connects"`
	IncludeCfgs     []string         `conf:"include_jobs_configs"`
	WaitingTimeout  time.Duration    `conf:"waiting_timeout"`

	LogFile  string `conf:"logfile" conf_extraopts:"default=stdout"`
	LogLevel string `conf:"loglevel" conf_extraopts:"default=info"`
	PidFile  string `conf:"pidfile"`
	ConfPath string
}

type notifications struct {
	Mail     mailConf      `conf:"mail"`
	Webhooks []webhookConf `conf:"webhooks"`
}

type mailConf struct {
	Enabled      bool     `conf:"enabled" conf_extraopts:"default=true"`
	From         string   `conf:"mail_from"`
	SmtpServer   string   `conf:"smtp_server"`
	SmtpPort     int      `conf:"smtp_port"`
	SmtpUser     string   `conf:"smtp_user"`
	SmtpPassword string   `conf:"smtp_password"`
	Recipients   []string `conf:"recipients"`
	MessageLevel string   `conf:"message_level" conf_extraopts:"default=err"`
}

type webhookConf struct {
	Enabled           bool                   `conf:"enabled" conf_extraopts:"default=true"`
	WebhookURL        string                 `conf:"webhook_url" conf_extraopts:"required"`
	PayloadMessageKey string                 `conf:"payload_message_key" conf_extraopts:"required"`
	ExtraPayload      map[string]interface{} `conf:"extra_payload"`
	ExtraHeaders      map[string]string      `conf:"extra_headers"`
	InsecureTLS       bool                   `conf:"insecure_tls" conf_extraopts:"default=false"`
	MessageLevel      string                 `conf:"message_level" conf_extraopts:"default=warn"`
}

type jobCfg struct {
	JobName          string        `conf:"job_name" conf_extraopts:"required"`
	JobType          string        `conf:"type" conf_extraopts:"required"`
	TmpDir           string        `conf:"tmp_dir"`
	SafetyBackup     bool          `conf:"safety_backup" conf_extraopts:"default=false"`
	DeferredCopying  bool          `conf:"deferred_copying" conf_extraopts:"default=false"`
	Sources          []sourceCfg   `conf:"sources"`
	StoragesOptions  []storageOpts `conf:"storages_options"`
	DumpCmd          string        `conf:"dump_cmd"`
	SkipBackupRotate bool          `conf:"skip_backup_rotate" conf_extraopts:"default=false"` // used by external
}

type sourceCfg struct {
	Name               string        `conf:"name" conf_extraopts:"required"`
	Connect            sourceConnect `conf:"connect"`
	Targets            []string      `conf:"targets"`
	TargetDBs          []string      `conf:"target_dbs"`
	TargetCollections  []string      `conf:"target_collections"`
	Excludes           []string      `conf:"excludes"`
	ExcludeDBs         []string      `conf:"exclude_dbs"`
	ExcludeCollections []string      `conf:"exclude_collections"`
	ExtraKeys          string        `conf:"db_extra_keys"`
	IsSlave            bool          `conf:"is_slave" conf_extraopts:"default=false"`
	Gzip               bool          `conf:"gzip" conf_extraopts:"default=false"`
	SaveAbsPath        bool          `conf:"save_abs_path" conf_extraopts:"default=true"`
	PrepareXtrabackup  bool          `conf:"prepare_xtrabackup" conf_extraopts:"default=false"`
}

type sourceConnect struct {
	DBHost          string `conf:"db_host"`
	DBPort          string `conf:"db_port"`
	Socket          string `conf:"socket"`
	DBUser          string `conf:"db_user"`
	DBPassword      string `conf:"db_password"`
	MySQLAuthFile   string `conf:"mysql_auth_file"`
	PsqlSSLMode     string `conf:"psql_ssl_mode" conf_extraopts:"default=require"`
	PsqlSSlRootCert string `conf:"psql_ssl_root_cert"`
	PsqlSSlCrl      string `conf:"psql_ssl_crl"`
	MongoRSName     string `conf:"mongo_replica_set_name"`
	MongoRSAddr     string `conf:"mongo_replica_set_address"`
	MongoTLSCAFile  string `conf:"mongo_tls_CA_file"`
	MongoAuthDB     string `conf:"mongo_auth_db"`
}

type storageOpts struct {
	StorageName string    `conf:"storage_name" conf_extraopts:"required"`
	BackupPath  string    `conf:"backup_path" conf_extraopts:"required"`
	Retention   retention `conf:"retention" conf_extraopts:"required"`
}

type retention struct {
	Days   int `conf:"days" conf_extraopts:"default=7"`
	Weeks  int `conf:"weeks" conf_extraopts:"default=5"`
	Months int `conf:"months" conf_extraopts:"default=12"`
}

type storageConnect struct {
	Name         string        `conf:"name" conf_extraopts:"required"`
	S3Params     *s3Params     `conf:"s3_params"`
	ScpParams    *sftpParams   `conf:"scp_params"`
	SftpParams   *sftpParams   `conf:"sftp_params"`
	FtpParams    *ftpParams    `conf:"ftp_params"`
	NfsParams    *nfsParams    `conf:"nfs_params"`
	WebDavParams *webDavParams `conf:"webdav_params"`
	SmbParams    *smbParams    `conf:"smb_params"`
}

type s3Params struct {
	BucketName  string `conf:"bucket_name" conf_extraopts:"required"`
	AccessKeyID string `conf:"access_key_id"`
	SecretKey   string `conf:"secret_access_key"`
	Endpoint    string `conf:"endpoint" conf_extraopts:"required"`
	Region      string `conf:"region" conf_extraopts:"required"`
	Secure      bool   `conf:"secure" conf_extraopts:"default=true"`
}

type sftpParams struct {
	User           string        `conf:"user" conf_extraopts:"required"`
	Host           string        `conf:"host" conf_extraopts:"required"`
	Port           int           `conf:"port" conf_extraopts:"default=22"`
	Password       string        `conf:"password"`
	KeyFile        string        `conf:"key_file"`
	ConnectTimeout time.Duration `conf:"connection_timeout" conf_extraopts:"default=10"`
}

type ftpParams struct {
	Host              string        `conf:"host"  conf_extraopts:"required"`
	User              string        `conf:"user"`
	Password          string        `conf:"password"`
	Port              int           `conf:"port" conf_extraopts:"default=21"`
	ConnectCount      int           `conf:"connect_count" conf_extraopts:"default=5"`
	ConnectionTimeout time.Duration `conf:"connection_timeout" conf_extraopts:"default=10"`
}

type nfsParams struct {
	Host   string `conf:"host"  conf_extraopts:"required"`
	Target string `conf:"target"`
	UID    uint32 `conf:"uid" conf_extraopts:"default=0"`
	GID    uint32 `conf:"gid" conf_extraopts:"default=0"`
}

type webDavParams struct {
	URL               string        `conf:"url" conf_extraopts:"required"`
	Username          string        `conf:"username"`
	Password          string        `conf:"password"`
	OAuthToken        string        `conf:"oauth_token"`
	ConnectionTimeout time.Duration `conf:"connection_timeout" conf_extraopts:"default=10"`
}

type smbParams struct {
	Host              string        `conf:"host" conf_extraopts:"required"`
	Port              int           `conf:"port" conf_extraopts:"default=445"`
	User              string        `conf:"user" conf_extraopts:"default=Guest"`
	Password          string        `conf:"password"`
	Domain            string        `conf:"domain"`
	Share             string        `conf:"share" conf_extraopts:"required"`
	ConnectionTimeout time.Duration `conf:"connection_timeout" conf_extraopts:"default=10"`
}

func confRead(confPath string) (confOpts, error) {

	var c confOpts

	p, err := misc.PathNormalize(confPath)
	if err != nil {
		return c, err
	}

	err = conf.Load(&c, conf.Settings{
		ConfPath:    p,
		ConfType:    conf.ConfigTypeYAML,
		UnknownDeny: true,
	})
	if err != nil {
		return c, err
	}

	c.ConfPath = confPath

	if len(c.IncludeCfgs) > 0 {
		err = c.extraCfgsRead()
		if err != nil {
			return c, err
		}
	}

	return c, nil
}

func (c *confOpts) extraCfgsRead() error {

	for _, pathRegexp := range c.IncludeCfgs {
		var p string

		abs, err := filepath.Abs(pathRegexp)
		if err != nil {
			return err
		}
		cp := path.Clean(pathRegexp)
		if abs != cp {
			p = path.Join(path.Dir(c.ConfPath), cp)
		} else {
			p = cp
		}

		confs, err := filepath.Glob(p)
		if err != nil {
			return err
		}

		var errs *multierror.Error
		for _, cfgFile := range confs {
			var j jobCfg

			err = conf.Load(&j, conf.Settings{
				ConfPath:    cfgFile,
				ConfType:    conf.ConfigTypeYAML,
				UnknownDeny: true,
			})
			if err != nil {
				errs = multierror.Append(errs, err)
			}

			c.Jobs = append(c.Jobs, j)
		}

		if errs != nil {
			return errs
		}
	}

	return nil
}

// validate checks if provided configuration valid
