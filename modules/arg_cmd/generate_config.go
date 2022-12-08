package arg_cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	appctx "github.com/nixys/nxs-go-appctx/v2"
	"gopkg.in/yaml.v3"

	"nxs-backup/ctx"
)

type jobCfgYml struct {
	JobName         string            `yaml:"job_name"`
	JobType         string            `yaml:"type"`
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

func GenerateConfig(appCtx *appctx.AppContext) error {

	var errs *multierror.Error

	cc := appCtx.CustomCtx().(*ctx.Ctx)
	params := cc.CmdParams.(*ctx.GenerateCmd)

	job := jobCfgYml{
		JobName:         fmt.Sprintf("PROJECT-%s", params.Type),
		JobType:         params.Type,
		DeferredCopying: false,
		SafetyBackup:    false,
		TmpDir:          "/var/nxs-backup/dump_tmp",
	}
	cfgName := params.Type + ".conf"

	switch params.Type {
	case ctx.AllowedJobTypes[0]:
		job.StoragesOptions = genStorageOpts(params.Storages, false)
		job.Sources = []sourceYaml{
			{
				Name: "desc_files",
				Gzip: true,
				Targets: []string{
					"/var/www/html/www.site.ru",
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
	case ctx.AllowedJobTypes[1]:
		job.StoragesOptions = genStorageOpts(params.Storages, true)
		job.Sources = []sourceYaml{
			{
				Name: "inc_files",
				Gzip: true,
				Targets: []string{
					"/var/www/html/www.site.ru",
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
	case ctx.AllowedJobTypes[2]:
		job.StoragesOptions = genStorageOpts(params.Storages, false)
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
	case ctx.AllowedJobTypes[3]:
		job.StoragesOptions = genStorageOpts(params.Storages, false)
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
				IsSlave:   false,
				TargetDBs: []string{"all"},
				Excludes: []string{
					"mysql",
					"information_schema",
					"performance_schema",
					"sys",
				},
				PrepareXtrabackup: true,
				ExtraKeys:         "--datadir=/path/to/mysql/data",
			},
		}
	case ctx.AllowedJobTypes[4]:
		job.StoragesOptions = genStorageOpts(params.Storages, false)
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
	case ctx.AllowedJobTypes[5]:
		job.StoragesOptions = genStorageOpts(params.Storages, false)
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
	case ctx.AllowedJobTypes[6]:
		job.StoragesOptions = genStorageOpts(params.Storages, false)
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
	case ctx.AllowedJobTypes[7]:
		job.StoragesOptions = genStorageOpts(params.Storages, false)
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
	case ctx.AllowedJobTypes[8]:
		job.StoragesOptions = genStorageOpts(params.Storages, false)
		job.DumpCmd = "/path/to/backup_script.sh"
		job.TmpDir = ""
	default:
		errs = multierror.Append(fmt.Errorf("Unknown job type. Allowed types: %s ", strings.Join(ctx.AllowedJobTypes, ", ")))
	}

	if errs != nil {
		return errs
	}

	// update storage connections
	if err := updateStorageConnects(cc.ConfigPath, params.Storages); err != nil {
		return err
	}

	// create config file

	var cfgPath string
	if params.OutPath == "" {
		cfgPath = path.Join(path.Dir(cc.ConfigPath), "conf.d", cfgName)
	} else {
		cfgPath = params.OutPath
	}
	file, err := os.Create(cfgPath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	e := yaml.NewEncoder(file)
	e.SetIndent(2)
	defer func() { _ = e.Close() }()

	if err = e.Encode(&job); err != nil {
		return err
	}

	fmt.Printf("Successfully added new sample config file: %s\n", cfgPath)

	return nil
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
	content, err := ioutil.ReadFile(cfgPath)
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
	allowedStorageTypes := []string{
		"s3",
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
		case allowedStorageTypes[0]:
			st.S3Params = &s3Params{
				BucketName:      "my_bucket",
				AccessKeyID:     "my_access_key",
				SecretAccessKey: "my_secret_key",
				Endpoint:        "my.s3.endpoint",
				Region:          "my-s3-region",
			}
		case allowedStorageTypes[1]:
			st.ScpParams = &sftpParams{
				Host:     "my_ssh_host",
				Port:     22,
				User:     "my_ssh_user",
				Password: "my_ssh_password",
			}
		case allowedStorageTypes[2]:
			st.SftpParams = &sftpParams{
				Host:     "my_ssh_host",
				Port:     22,
				User:     "my_ssh_user",
				Password: "my_ssh_password",
			}
		case allowedStorageTypes[3]:
			st.FtpParams = &ftpParams{
				Host:     "my_ftp_host",
				Port:     21,
				User:     "my_ftp_user",
				Password: "my_ftp_pass",
			}
		case allowedStorageTypes[4]:
			st.SmbParams = &smbParams{
				Host:     "my_smb_host",
				Port:     445,
				User:     "my_smb_user",
				Password: "my_smb_pass",
				Share:    "my_smb_share_path",
				Domain:   "my_smb_domain",
			}
		case allowedStorageTypes[5]:
			st.NfsParams = &nfsParams{
				Host:   "my_nfs_host",
				Port:   111,
				Target: "my_nfs_target_path",
				UID:    1000,
				GID:    1000,
			}
		case allowedStorageTypes[6]:
			st.WebDavParams = &webDavParams{
				URL:        "my_webdav_url",
				Username:   "my_webdav_user",
				Password:   "my_webdav_pass",
				OAuthToken: "my_webdav_oauth_token",
			}
		default:
			return nil, fmt.Errorf("Unknown strage type. Supported types: %s ", strings.Join(allowedStorageTypes, ", "))
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
