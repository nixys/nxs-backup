package generate_config

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"gopkg.in/yaml.v3"

	"github.com/nixys/nxs-backup/misc"
)

type jobCfgYml struct {
	JobName         string            `yaml:"job_name"`
	JobType         misc.BackupType   `yaml:"type"`
	TmpDir          string            `yaml:"tmp_dir,omitempty"`
	SafetyBackup    bool              `yaml:"safety_backup"`
	DeferredCopying bool              `yaml:"deferred_copying"`
	Sources         []sourceYaml      `yaml:"sources"`
	StoragesOptions []storageOptsYaml `yaml:"storages_options"`
	DumpCmd         string            `yaml:"dump_cmd,omitempty"`
}

type sourceYaml struct {
	Name               string         `yaml:"name"`
	Connect            srcConnectYaml `yaml:"connect,omitempty"`
	SpecialKeys        string         `yaml:"special_keys,omitempty"`
	Targets            []string       `yaml:"targets,omitempty"`
	TargetDBs          []string       `yaml:"target_dbs,omitempty"`
	TargetCollections  []string       `yaml:"target_collections,omitempty"`
	Excludes           []string       `yaml:"excludes,omitempty"`
	ExcludeDbs         []string       `yaml:"exclude_dbs,omitempty"`
	ExcludeCollections []string       `yaml:"exclude_collections,omitempty"`
	Gzip               bool           `yaml:"gzip,omitempty"`
	SaveAbsPath        bool           `yaml:"save_abs_path,omitempty"`
	IsSlave            bool           `yaml:"is_slave,omitempty"`
	ExtraKeys          string         `yaml:"db_extra_keys,omitempty"`
	SkipBackupRotate   bool           `yaml:"skip_backup_rotate,omitempty"` // used by external
	PrepareXtrabackup  bool           `yaml:"prepare_xtrabackup,omitempty"`
}

type srcConnectYaml struct {
	AuthFile       string        `yaml:"auth_file,omitempty"`
	DBHost         string        `yaml:"db_host,omitempty"`
	DBPort         string        `yaml:"db_port,omitempty"`
	Socket         string        `yaml:"socket,omitempty"`
	SSLMode        string        `yaml:"psql_ssl_mode,omitempty"`
	DBUser         string        `yaml:"db_user,omitempty"`
	DBPassword     string        `yaml:"db_password,omitempty"`
	MongoRSName    string        `yaml:"mongo_replica_set_name,omitempty"`
	MongoRSAddr    string        `yaml:"mongo_replica_set_address,omitempty"`
	ConnectTimeout time.Duration `yaml:"connection_timeout,omitempty"`
}

type storageOptsYaml struct {
	StorageName string           `yaml:"storage_name"`
	BackupPath  string           `yaml:"backup_path"`
	Retention   cfgRetentionYaml `yaml:"retention"`
}

type cfgRetentionYaml struct {
	Days   int `yaml:"days,omitempty"`
	Weeks  int `yaml:"weeks,omitempty"`
	Months int `yaml:"months,omitempty"`
}

type storageConnect struct {
	Name         string        `yaml:"name"`
	S3Params     *s3Params     `yaml:"s3_params,omitempty"`
	ScpParams    *sftpParams   `yaml:"scp_params,omitempty"`
	SftpParams   *sftpParams   `yaml:"sftp_params,omitempty"`
	FtpParams    *ftpParams    `yaml:"ftp_params,omitempty"`
	NfsParams    *nfsParams    `yaml:"nfs_params,omitempty"`
	WebDavParams *webDavParams `yaml:"webdav_params,omitempty"`
	SmbParams    *smbParams    `yaml:"smb_params,omitempty"`
}

type s3Params struct {
	BucketName      string `yaml:"bucket_name"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	Endpoint        string `yaml:"endpoint"`
	Region          string `yaml:"region"`
}

type sftpParams struct {
	User     string `yaml:"user"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	KeyFile  string `yaml:"key_file"`
}

type ftpParams struct {
	Host     string `yaml:"host" `
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Port     int    `yaml:"port"`
}

type nfsParams struct {
	Host   string `yaml:"host" `
	Target string `yaml:"target"`
	UID    uint32 `yaml:"uid"`
	GID    uint32 `yaml:"gid"`
	Port   int    `yaml:"port"`
}

type webDavParams struct {
	URL        string `yaml:"url"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	OAuthToken string `yaml:"oauth_token"`
}

type smbParams struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Domain   string `yaml:"domain"`
	Share    string `yaml:"share"`
}

type Opts struct {
	Done     chan error
	CfgPath  string
	JobType  misc.BackupType
	OutPath  string
	Arg      *arg.Parser
	Storages map[string]string
}

type generateConfig struct {
	done     chan error
	cfgPath  string
	jobType  misc.BackupType
	outPath  string
	arg      *arg.Parser
	storages map[string]string
}

func Init(o Opts) *generateConfig {
	return &generateConfig{
		done:     o.Done,
		arg:      o.Arg,
		cfgPath:  o.CfgPath,
		jobType:  o.JobType,
		outPath:  o.OutPath,
		storages: o.Storages,
	}
}

func (gc *generateConfig) Run() {

	job := jobCfgYml{
		JobName:         fmt.Sprintf("PROJECT-%s", gc.jobType),
		JobType:         gc.jobType,
		DeferredCopying: false,
		SafetyBackup:    false,
		TmpDir:          "/var/nxs-backup/dump_tmp",
	}
	cfgName := gc.jobType + ".conf"

	switch gc.jobType {
	case misc.DescFiles:
		job.StoragesOptions = genStorageOpts(gc.storages, false)
		job.Sources = []sourceYaml{
			{
				Name: "desc_files",
				Gzip: true,
				Targets: []string{
					"/var/www/html/www.site.io",
					"/some/path/for/backup",
				},
				Excludes: []string{
					"tmp",
					"log",
					"some_extra_exclude",
				},
				SaveAbsPath: true,
			},
		}
	case misc.IncFiles:
		job.StoragesOptions = genStorageOpts(gc.storages, true)
		job.Sources = []sourceYaml{
			{
				Name: "inc_files",
				Gzip: true,
				Targets: []string{
					"/var/www/html/www.site.io",
					"/some/path/for/backup",
				},
				Excludes: []string{
					"tmp",
					"log",
					"some_extra_exclude",
				},
				SaveAbsPath: true,
			},
		}
	case misc.Mysql:
		job.StoragesOptions = genStorageOpts(gc.storages, false)
		job.Sources = []sourceYaml{
			{
				Name: "mysql",
				Gzip: true,
				Connect: srcConnectYaml{
					DBHost:     "mysql",
					DBPort:     "3306",
					DBUser:     "root",
					DBPassword: "rootP@5s",
					Socket:     "",
					AuthFile:   "",
				},
				IsSlave:   false,
				TargetDBs: []string{"all"},
				Excludes: []string{
					"mysql",
					"information_schema",
					"performance_schema",
					"sys",
				},
				ExtraKeys: "--opt --add-drop-database --routines --comments --create-options --quote-names --order-by-primary --hex-blob --single-transaction",
			},
		}
	case misc.MysqlXtrabackup:
		job.StoragesOptions = genStorageOpts(gc.storages, false)
		job.Sources = []sourceYaml{
			{
				Name: "mysql_xtrabackup",
				Gzip: true,
				Connect: srcConnectYaml{
					DBHost:     "mysql",
					DBPort:     "3306",
					DBUser:     "root",
					DBPassword: "rootP@5s",
					Socket:     "",
					AuthFile:   "",
				},
				IsSlave: false,
				Excludes: []string{
					"bd_name.table_to_exclude",
				},
				PrepareXtrabackup: true,
				ExtraKeys:         "--datadir=/path/to/mysql/data",
			},
		}
	case misc.Postgresql:
		job.StoragesOptions = genStorageOpts(gc.storages, false)
		job.Sources = []sourceYaml{
			{
				Name: "psql",
				Gzip: true,
				Connect: srcConnectYaml{
					DBHost:     "psql",
					DBPort:     "5432",
					DBUser:     "postgres",
					DBPassword: "postgresP@5s",
					Socket:     "",
					SSLMode:    "require",
				},
				TargetDBs: []string{"all"},
				Excludes: []string{
					"postgres",
					"demo.information_schema",
				},
				ExtraKeys: "",
			},
		}
	case misc.PostgresqlBasebackup:
		job.StoragesOptions = genStorageOpts(gc.storages, false)
		job.Sources = []sourceYaml{
			{
				Name: "psql_basebackup",
				Gzip: true,
				Connect: srcConnectYaml{
					DBHost:     "psql",
					DBPort:     "5432",
					DBUser:     "repmgr",
					DBPassword: "repmgrP@5s",
					Socket:     "",
					SSLMode:    "require",
				},
				ExtraKeys: "",
			},
		}
	case misc.MongoDB:
		job.StoragesOptions = genStorageOpts(gc.storages, false)
		job.Sources = []sourceYaml{
			{
				Name: "mongodb",
				Gzip: true,
				Connect: srcConnectYaml{
					DBHost:      "mongo1",
					DBPort:      "27017",
					DBUser:      "mongo",
					DBPassword:  "mongoP@5s",
					MongoRSName: "",
					MongoRSAddr: "",
				},
				TargetDBs:         []string{"all"},
				TargetCollections: []string{"all"},
				ExcludeDbs: []string{
					"admin",
					"config",
					"local",
				},
				ExcludeCollections: []string{"sample_mflix.users"},
				ExtraKeys:          "",
			},
		}
	case misc.Redis:
		job.StoragesOptions = genStorageOpts(gc.storages, false)
		job.Sources = []sourceYaml{
			{
				Name: "redis",
				Gzip: true,
				Connect: srcConnectYaml{
					DBHost:     "redis",
					DBPort:     "6379",
					DBPassword: "redisP@5s",
					Socket:     "",
				},
			},
		}
	case misc.External:
		job.StoragesOptions = genStorageOpts(gc.storages, false)
		job.DumpCmd = "/path/to/my_script.sh"
		job.TmpDir = ""
	default:
		printGenCfgErr(gc.done, fmt.Errorf("Unknown backup type. Allowed types: %s ", strings.Join(misc.AllowedBackupTypesList(), ", ")), gc.arg)
		return
	}

	// update storage connections
	if err := updateStorageConnects(gc.cfgPath, gc.storages); err != nil {
		printGenCfgErr(gc.done, err, gc.arg)
		return
	}

	// create config file

	var newCfgPath string
	if gc.outPath == "" {
		newCfgPath = path.Join(path.Dir(gc.cfgPath), "conf.d", string(cfgName))
	} else {
		newCfgPath = gc.outPath
	}
	file, err := os.Create(newCfgPath)
	if err != nil {
		printGenCfgErr(gc.done, err, gc.arg)
		return
	}
	defer func() { _ = file.Close() }()

	e := yaml.NewEncoder(file)
	e.SetIndent(2)
	defer func() { _ = e.Close() }()

	if err = e.Encode(&job); err != nil {
		printGenCfgErr(gc.done, err, gc.arg)
		return
	}

	fmt.Printf("Successfully added new sample config file: %s\n", newCfgPath)

	gc.done <- nil
}

func genStorageOpts(storages map[string]string, incBackup bool) (sts []storageOptsYaml) {

	defaultRetention := cfgRetentionYaml{
		Days:   7,
		Weeks:  5,
		Months: 5,
	}

	if incBackup {
		defaultRetention = cfgRetentionYaml{Months: 12}
	}

	sts = append(sts, storageOptsYaml{
		StorageName: "local",
		BackupPath:  "/var/nxs-backup/dump",
		Retention:   defaultRetention,
	})

	for nm := range storages {
		sts = append(sts, storageOptsYaml{
			StorageName: nm,
			BackupPath:  "/nxs-backup/dump",
			Retention:   defaultRetention,
		})
	}

	return
}

func updateStorageConnects(cfgPath string, storages map[string]string) error {
	content, err := os.ReadFile(cfgPath)
	if err != nil {
		return err
	}

	yamlNode := yaml.Node{}

	err = yaml.Unmarshal(content, &yamlNode)
	if err != nil {
		return err
	}

	stIdx := -1
	for i, k := range yamlNode.Content[0].Content {
		if k.Value == "storage_connects" {
			stIdx = i + 1
			break
		}
	}

	if stIdx == -1 {
		sNode := &yaml.Node{}
		sNode.SetString("storage_connects")
		lNode := &yaml.Node{}
		lNode.Tag = "!!seq"
		lNode.Kind = yaml.SequenceNode
		lNode.Style = 0
		yamlNode.Content[0].Content = append(yamlNode.Content[0].Content, sNode, lNode)
		stIdx = len(yamlNode.Content[0].Content) - 1
	} else if yamlNode.Content[0].Content[stIdx].Style == yaml.FlowStyle {
		yamlNode.Content[0].Content[stIdx].Style = 0
	}

	stNodes, err := getStorageConnects(storages)
	if err != nil {
		return err
	}

	for _, stNode := range stNodes {
		for _, st := range yamlNode.Content[0].Content[stIdx].Content {
			if st.Content[1].Value == stNode.Content[0].Content[1].Value {
				return fmt.Errorf("Storage with name `%s` already present in config ", st.Content[1].Value)
			}
		}
		yamlNode.Content[0].Content[stIdx].Content = append(
			yamlNode.Content[0].Content[stIdx].Content, stNode.Content[0])
	}

	file, err := os.Create(cfgPath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	e := yaml.NewEncoder(file)
	e.SetIndent(2)
	defer func() { _ = e.Close() }()

	return e.Encode(&yamlNode)
}

func getStorageConnects(storages map[string]string) ([]*yaml.Node, error) {
	ast := []string{
		"s3",
		"ssh",
		"scp",
		"sftp",
		"ftp",
		"smb",
		"nfs",
		"webdav",
	}
	var sts []*yaml.Node

	for stName, stType := range storages {
		st := storageConnect{Name: stName}
		//stNode := yaml.Node{}
		switch stType {
		case ast[0]:
			st.S3Params = &s3Params{
				BucketName:      "my_bucket",
				AccessKeyID:     "my_access_key",
				SecretAccessKey: "my_secret_key",
				Endpoint:        "my.s3.endpoint",
				Region:          "my-s3-region",
			}
		case ast[1], ast[2], ast[3]:
			st.ScpParams = &sftpParams{
				Host:     "my_ssh_host",
				Port:     22,
				User:     "my_ssh_user",
				Password: "my_ssh_password",
			}
		case ast[4]:
			st.FtpParams = &ftpParams{
				Host:     "my_ftp_host",
				Port:     21,
				User:     "my_ftp_user",
				Password: "my_ftp_pass",
			}
		case ast[5]:
			st.SmbParams = &smbParams{
				Host:     "my_smb_host",
				Port:     445,
				User:     "my_smb_user",
				Password: "my_smb_pass",
				Share:    "my_smb_share_path",
				Domain:   "my_smb_domain",
			}
		case ast[6]:
			st.NfsParams = &nfsParams{
				Host:   "my_nfs_host",
				Port:   111,
				Target: "my_nfs_target_path",
				UID:    1000,
				GID:    1000,
			}
		case ast[7]:
			st.WebDavParams = &webDavParams{
				URL:        "my_webdav_url",
				Username:   "my_webdav_user",
				Password:   "my_webdav_pass",
				OAuthToken: "my_webdav_oauth_token",
			}
		default:
			return nil, fmt.Errorf("Unknown storage type. Supported types: %s ", strings.Join(ast, ", "))
		}
		data, err := yaml.Marshal(st)
		if err != nil {
			return nil, err
		}
		node := yaml.Node{}
		if err = yaml.Unmarshal(data, &node); err != nil {
			return nil, err
		}
		sts = append(sts, &node)
	}

	return sts, nil
}

func printGenCfgErr(dc chan error, err error, arg *arg.Parser) {
	_, _ = fmt.Fprintf(os.Stderr, "Can't generate config: %v\n", err)
	_ = arg.FailSubcommand(err.Error(), "generate")
	dc <- err
}
