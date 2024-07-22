package list_backups

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/hashicorp/go-multierror"

	"github.com/nixys/nxs-backup/interfaces"
	"github.com/nixys/nxs-backup/misc"
)

type Opts struct {
	JobName  string
	InitErr  error
	Done     chan error
	FileJobs interfaces.Jobs
	DBJobs   interfaces.Jobs
	ExtJobs  interfaces.Jobs
	Jobs     map[string]interfaces.Job
}

type listBackups struct {
	jobName  string
	initErr  error
	done     chan error
	fileJobs interfaces.Jobs
	dbJobs   interfaces.Jobs
	extJobs  interfaces.Jobs
	jobs     map[string]interfaces.Job
}

type treeElement struct {
	name     string
	children []treeElement
}

var (
	bold       = color.New(color.Bold)
	italic     = color.New(color.Italic)
	italicBold = color.New(color.Italic, color.Bold)
)

func Init(o Opts) *listBackups {
	return &listBackups{
		jobName:  o.JobName,
		initErr:  o.InitErr,
		done:     o.Done,
		fileJobs: o.FileJobs,
		dbJobs:   o.DBJobs,
		extJobs:  o.ExtJobs,
		jobs:     o.Jobs,
	}
}

func (lb *listBackups) Run() {
	var err error
	errs := new(multierror.Error)

	defer func() {
		lb.done <- err
	}()

	if lb.initErr != nil {
		color.HiRed("[WARNING!] Backup plan initialised with errors:")
		fmt.Println(lb.initErr)
	}

	if lb.jobName == "external" || lb.jobName == "all" {
		err := printBackups("External", lb.extJobs)
		errs = multierror.Append(err, errs)
	}
	if lb.jobName == "databases" || lb.jobName == "all" {
		err := printBackups("Database", lb.dbJobs)
		errs = multierror.Append(err, errs)
	}
	if lb.jobName == "files" || lb.jobName == "all" {
		err := printBackups("File", lb.fileJobs)
		errs = multierror.Append(err, errs)
	}
	if job, ok := lb.jobs[lb.jobName]; ok {
		err := printBackups("", interfaces.Jobs{job})
		errs = multierror.Append(err, errs)
	}
	if errs.Len() > 0 {
		color.HiRed("[WARNING!] Execution finished with errors.")
		err = misc.ErrExecution
	}
}

func printBackups(bType string, jobs interfaces.Jobs) (err error) {
	var backupsTree []treeElement

	if len(bType) > 0 && len(jobs) > 0 {
		bold.Printf("%s backup jobs\n", bType)
	}

	for _, job := range jobs {
		var jobTargets []treeElement
		jt := job.ListBackups()
		for tName, tOnSt := range jt {
			jobTargetSts := make([]treeElement, 0, len(tOnSt))
			for st, tFiles := range tOnSt {
				jobTargetStFiles := make([]treeElement, 0, len(tFiles.List))
				if tFiles.ListErr == nil {
					for _, f := range tFiles.List {
						jobTargetStFiles = append(jobTargetStFiles, treeElement{
							name: f,
						})
					}
				} else {
					jobTargetStFiles = append(jobTargetStFiles, treeElement{
						name: fmt.Sprintf("Failed to get files from storage. Err: `%v`", tFiles.ListErr),
					})
					err = misc.ErrExecution
				}

				jobTargetSts = append(jobTargetSts, treeElement{
					name:     st,
					children: jobTargetStFiles,
				})
			}
			jobTargets = append(jobTargets, treeElement{
				name:     tName,
				children: jobTargetSts,
			})
		}
		backupsTree = append(backupsTree, treeElement{
			name:     job.GetName(),
			children: jobTargets,
		})
	}

	if len(bType) == 0 && len(jobs) == 1 {
		italicBold.Printf("%s backup job\n", jobs[0].GetName())
		printTree(backupsTree[0].children, "", 1)
	} else {
		printTree(backupsTree, "", 0)
	}

	return
}

func printTree(elements []treeElement, prefix string, lvl int) {
	for i, e := range elements {
		if i == len(elements)-1 {
			fmt.Print(prefix + "└── ")
			printlnWithLevel(lvl, e.name)
			printTree(e.children, prefix+"    ", lvl+1)
		} else {
			fmt.Print(prefix + "├── ")
			printlnWithLevel(lvl, e.name)
			printTree(e.children, prefix+"│   ", lvl+1)
		}
	}
}

func printlnWithLevel(level int, msg string) {
	switch level {
	case 0:
		italicBold.Println(msg)
	case 1:
		bold.Println(msg)
	case 2:
		italic.Println(msg)
	default:
		fmt.Println(msg)
	}
}
