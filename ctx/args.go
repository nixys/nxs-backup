package ctx

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/alexflint/go-arg"
	appctx "github.com/nixys/nxs-go-appctx/v2"

	"nxs-backup/misc"
)

type cmdHandler func(*appctx.AppContext) error

// SubCmds contains Command name and handler
type SubCmds map[string]cmdHandler

// ArgsParams contains parameters read from command line, command parameters and command handler
type ArgsParams struct {
	ConfigPath string
	CmdHandler cmdHandler
	CmdParams  interface{}
}

type StartCmd struct {
	JobName string `arg:"positional" placeholder:"JOB GROUP/NAME" default:"all"`
}

type GenerateCmd struct {
	Type     string            `arg:"-T,--backup-type" help:"Type of backup"`
	Storages map[string]string `arg:"-S,--storage-types" help:"Storages names with type. Example: -S minio=s3 aws=s3"`
	OutPath  string            `arg:"-O,--out-path" help:"Path to the generated configuration file" placeholder:"PATH"`
}

type args struct {
	Start    *StartCmd    `arg:"subcommand:start"`
	Generate *GenerateCmd `arg:"subcommand:generate"`
	ConfPath string       `arg:"-c,--config" help:"Path to config file" default:"/etc/nxs-backup/nxs-backup.conf" placeholder:"PATH"`
	TestConf bool         `arg:"-t,--test-config" help:"Check if configuration correct"`
}

// ReadArgs reads arguments from command line
func ReadArgs(cmds SubCmds) (p ArgsParams) {

	var a args

	curArgs := arg.MustParse(&a)

	if err := checkConfigPath(a.ConfPath); err != nil {
		fmt.Println("Config error:", err)
		os.Exit(1)
	}

	p.ConfigPath = a.ConfPath

	if a.TestConf {
		p.CmdHandler = cmds["testCfg"]
		return
	}

	subCmds := curArgs.SubcommandNames()
	if len(subCmds) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "Command not defined")
		curArgs.WriteHelp(os.Stderr)
		os.Exit(1)
	}
	p.CmdHandler = cmds[subCmds[0]]
	p.CmdParams = curArgs.Subcommand()

	return p
}

func (args) Version() string {
	return "nxs-backup " + misc.VERSION
}

func checkConfigPath(configPath string) error {
	cp, err := misc.PathNormalize(configPath)
	if err != nil {
		return err
	}

	// Check config file exist
	_, err = os.Stat(cp)
	if err == nil {
		return nil
	}

	// If not, default files and directories ask to create
	r := bufio.NewReader(os.Stdin)
	var s string

	fmt.Printf("Can't find config by path '%s'. Would you like to create? (y/N): ", configPath)

	s, _ = r.ReadString('\n')

	s = strings.TrimSpace(s)
	s = strings.ToLower(s)

	// Create new config
	if s == "y" || s == "yes" {
		if err = os.MkdirAll(path.Join(path.Dir(cp), "conf.d"), 0750); err != nil {
			return err
		}
		f, err := os.Create(cp)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		_, err = f.WriteString(emptyConfig())
		if err != nil {
			return err
		}
	}

	return nil
}

func emptyConfig() string {
	return `server_name: localhost
#project_name: My best Project

notifications:
  mail:
    enabled: false
    mail_from: backup@localhost
    smtp_server: ''
    smtp_port: 465
    smtp_user: ''
    smtp_password: ''
    recipients:
    - root@localhost
  nxs_alert:
    enabled: false
    auth_key: ''

storage_connects: []

jobs: []

include_jobs_configs: ["conf.d/*.conf"]

logfile: /var/log/nxs-backup/nxs-backup.log
`
}
