package test_config

import (
	"fmt"
	"github.com/nixys/nxs-backup/interfaces"
)

type Opts struct {
	InitErr  error
	Done     chan error
	FileJobs interfaces.Jobs
	DBJobs   interfaces.Jobs
	ExtJobs  interfaces.Jobs
}

type testConfig struct {
	initErr  error
	done     chan error
	fileJobs interfaces.Jobs
	dbJobs   interfaces.Jobs
	extJobs  interfaces.Jobs
}

func Init(o Opts) *testConfig {
	return &testConfig{
		initErr:  o.InitErr,
		done:     o.Done,
		fileJobs: o.FileJobs,
		dbJobs:   o.DBJobs,
		extJobs:  o.ExtJobs,
	}
}

func (tc *testConfig) Run() {

	if tc.initErr != nil {
		fmt.Printf("The configuration have next errors:\n%v\n", tc.initErr)
	} else {
		fmt.Printf("The configuration is correct\n\n")
	}

	if len(tc.extJobs) > 0 {
		fmt.Println("List of external jobs:")
		for _, job := range tc.extJobs {
			fmt.Printf("  %s\n", job.GetName())
		}
	} else {
		fmt.Println("No external jobs")
	}
	if len(tc.dbJobs) > 0 {
		fmt.Println("List of databases jobs:")
		for _, job := range tc.dbJobs {
			fmt.Printf("  %s\n", job.GetName())
		}
	} else {
		fmt.Println("No databases jobs")
	}
	if len(tc.fileJobs) > 0 {
		fmt.Println("List of files jobs:")
		for _, job := range tc.fileJobs {
			fmt.Printf("  %s\n", job.GetName())
		}
	} else {
		fmt.Println("No files jobs")
	}
	tc.done <- tc.initErr
}
