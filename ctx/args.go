package ctx

import (
	"fmt"
	"os"
	"strings"

	"github.com/alexflint/go-arg"

	"github.com/nixys/nxs-backup/misc"
)

type command string

const (
	start     command = "start"
	server    command = "server"
	generate  command = "generate"
	update    command = "update"
	lsBackups command = "ls_backups"
	testCfg   command = "test_cfg"
	unknown   command = "unknown"
)

// ArgsParams contains parameters read from command line, command parameters and command handler
type ArgsParams struct {
	ConfigPath string
	Cmd        command
	CmdParams  interface{}
	Arg        *arg.Parser
}

type StartCmd struct {
	JobName string `arg:"positional" help:"Name of job or jobs group to run [default: all]" default:"all" placeholder:"JOB_NAME/GROUP_NAME"`
}

// ServerCmd "Running the nxs-backup in server mode"
type ServerCmd struct{}

type GenerateCmd struct {
	Type     misc.BackupType   `arg:"-T,--backup-type,required" help:"Type of backup"`
	Storages map[string]string `arg:"-S,--storage-types" help:"Storages names with type. Example: -S minio=s3 aws=s3"`
	OutPath  string            `arg:"-O,--out-path" help:"Path to the generated configuration file" placeholder:"PATH"`
}

type ListCmd struct {
	Backups *StartCmd `arg:"subcommand:backups"`
}

type UpdateCmd struct {
	Version string `arg:"-V,--set-version" help:"Use the specific version to update. Example: -V 3.2.0-rc0" default:"3"`
}

type args struct {
	Start    *StartCmd    `arg:"subcommand:start"`
	Server   *ServerCmd   `arg:"subcommand:server"`
	Generate *GenerateCmd `arg:"subcommand:generate"`
	Update   *UpdateCmd   `arg:"subcommand:update"`
	List     *ListCmd     `arg:"subcommand:ls"`
	ConfPath string       `arg:"-c,--config" help:"Path to config file" default:"/etc/nxs-backup/nxs-backup.conf" placeholder:"PATH"`
	TestConf bool         `arg:"-t,--test-config" help:"Check if configuration correct"`
}

// ReadArgs reads arguments from command line
func ReadArgs() (p ArgsParams, err error) {

	var a args

	curArgs := arg.MustParse(&a)

	p.ConfigPath = a.ConfPath

	if a.TestConf {
		p.Cmd = testCfg
		return
	}

	subCmds := curArgs.SubcommandNames()
	if len(subCmds) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "Command not defined")
		curArgs.WriteHelp(os.Stderr)
		return p, misc.ErrArg
	}

	cmd := getCmd(strings.Join(subCmds, "_"))
	if cmd == unknown {
		_, _ = fmt.Fprintln(os.Stderr, "Unknown command")
		curArgs.WriteHelp(os.Stderr)
		return p, misc.ErrArg
	}
	p.Cmd = cmd
	p.CmdParams = curArgs.Subcommand()
	p.Arg = curArgs

	return
}

func (args) Version() string {
	return "nxs-backup " + misc.VERSION
}

func getCmd(argCmd string) command {
	switch command(argCmd) {
	case start:
		return start
	case server:
		return server
	case generate:
		return generate
	case update:
		return update
	case lsBackups:
		return lsBackups
	case testCfg:
		return testCfg
	default:
		return unknown
	}
}
