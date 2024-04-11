package cmd_handler

import (
	"fmt"
	"github.com/nixys/nxs-backup/interfaces"
)

type TestConfig struct {
	initErr  error
	done     chan error
	fileJobs interfaces.Jobs
	dbJobs   interfaces.Jobs
	extJobs  interfaces.Jobs
}

func InitTestConfig(ie error, dc chan error, fj, dj, ej interfaces.Jobs) *TestConfig {
	return &TestConfig{
		initErr:  ie,
		done:     dc,
		fileJobs: fj,
		dbJobs:   dj,
		extJobs:  ej,
	}
}

func (tc *TestConfig) Run() {

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
