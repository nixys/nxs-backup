package ctx

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/go-multierror"

	"nxs-backup/interfaces"
	"nxs-backup/modules/backup/desc_files"
	"nxs-backup/modules/backup/external"
	"nxs-backup/modules/backup/inc_files"
	"nxs-backup/modules/backup/mongodump"
	"nxs-backup/modules/backup/mysql"
	"nxs-backup/modules/backup/mysql_xtrabackup"
	"nxs-backup/modules/backup/psql"
	"nxs-backup/modules/backup/psql_basebackup"
	"nxs-backup/modules/backup/redis"
	"nxs-backup/modules/connectors/mongo_connect"
	"nxs-backup/modules/connectors/mysql_connect"
	"nxs-backup/modules/connectors/psql_connect"
	"nxs-backup/modules/connectors/redis_connect"
	"nxs-backup/modules/storage"
)

var AllowedJobTypes = []string{
	"desc_files",
	"inc_files",
	"mysql",
	"mysql_xtrabackup",
	"postgresql",
	"postgresql_basebackup",
	"mongodb",
	"redis",
	"external",
}

func jobsInit(cfgJobs []jobCfg, storages map[string]interfaces.Storage) ([]interfaces.Job, error) {
	var errs *multierror.Error
	var jobs []interfaces.Job

	for _, j := range cfgJobs {

		// jobs validation
		if len(j.JobName) == 0 {
			errs = multierror.Append(errs, fmt.Errorf("empty job name is unacceptable"))
			continue
		}
		jobStorages, needToMakeBackup, stErrs := initJobStorages(storages, j)
		if len(stErrs) > 0 {
			errs = multierror.Append(errs, stErrs...)
			continue
		}

		switch j.JobType {
		case AllowedJobTypes[0]:
			var sources []desc_files.SourceParams
			for _, src := range j.Sources {
				sources = append(sources, desc_files.SourceParams{
					Name:        src.Name,
					Targets:     src.Targets,
					Excludes:    src.Excludes,
					Gzip:        src.Gzip,
					SaveAbsPath: src.SaveAbsPath,
				})
			}

			job, err := desc_files.Init(desc_files.JobParams{
				Name:             j.JobName,
				TmpDir:           j.TmpDir,
				NeedToMakeBackup: needToMakeBackup,
				SafetyBackup:     j.SafetyBackup,
				DeferredCopying:  j.DeferredCopying,
				Storages:         jobStorages,
				Sources:          sources,
			})
			if err != nil {
				errs = multierror.Append(errs, err)
				continue
			}

			jobs = append(jobs, job)

		case AllowedJobTypes[1]:
			var sources []inc_files.SourceParams
			for _, src := range j.Sources {
				sources = append(sources, inc_files.SourceParams{
					Name:        src.Name,
					Targets:     src.Targets,
					Excludes:    src.Excludes,
					Gzip:        src.Gzip,
					SaveAbsPath: src.SaveAbsPath,
				})
			}

			job, err := inc_files.Init(inc_files.JobParams{
				Name:            j.JobName,
				TmpDir:          j.TmpDir,
				SafetyBackup:    j.SafetyBackup,
				DeferredCopying: j.DeferredCopying,
				Storages:        jobStorages,
				Sources:         sources,
			})
			if err != nil {
				errs = multierror.Append(errs, err)
				continue
			}

			jobs = append(jobs, job)

		case AllowedJobTypes[2]:
			var sources []mysql.SourceParams

			for _, src := range j.Sources {
				var extraKeys []string
				if len(src.ExtraKeys) > 0 {
					extraKeys = strings.Split(src.ExtraKeys, " ")
				}

				sources = append(sources, mysql.SourceParams{
					ConnectParams: mysql_connect.Params{
						AuthFile: src.Connect.MySQLAuthFile,
						User:     src.Connect.DBUser,
						Passwd:   src.Connect.DBPassword,
						Host:     src.Connect.DBHost,
						Port:     src.Connect.DBPort,
						Socket:   src.Connect.Socket,
					},
					Name:      src.Name,
					TargetDBs: src.TargetDBs,
					Excludes:  src.Excludes,
					Gzip:      src.Gzip,
					IsSlave:   src.IsSlave,
					ExtraKeys: extraKeys,
				})
			}

			job, err := mysql.Init(mysql.JobParams{
				Name:             j.JobName,
				TmpDir:           j.TmpDir,
				NeedToMakeBackup: needToMakeBackup,
				SafetyBackup:     j.SafetyBackup,
				DeferredCopying:  j.DeferredCopying,
				Storages:         jobStorages,
				Sources:          sources,
			})
			if err != nil {
				errs = multierror.Append(errs, err)
				continue
			}
			jobs = append(jobs, job)

		case AllowedJobTypes[3]:
			var sources []mysql_xtrabackup.SourceParams

			for _, src := range j.Sources {
				var extraKeys []string
				if len(src.ExtraKeys) > 0 {
					extraKeys = strings.Split(src.ExtraKeys, " ")
				}

				sources = append(sources, mysql_xtrabackup.SourceParams{
					ConnectParams: mysql_connect.Params{
						AuthFile: src.Connect.MySQLAuthFile,
						User:     src.Connect.DBUser,
						Passwd:   src.Connect.DBPassword,
						Host:     src.Connect.DBHost,
						Port:     src.Connect.DBPort,
						Socket:   src.Connect.Socket,
					},
					Name:      src.Name,
					TargetDBs: src.TargetDBs,
					Excludes:  src.Excludes,
					Gzip:      src.Gzip,
					IsSlave:   src.IsSlave,
					Prepare:   src.PrepareXtrabackup,
					ExtraKeys: extraKeys,
				})
			}

			job, err := mysql_xtrabackup.Init(mysql_xtrabackup.JobParams{
				Name:             j.JobName,
				TmpDir:           j.TmpDir,
				NeedToMakeBackup: needToMakeBackup,
				SafetyBackup:     j.SafetyBackup,
				DeferredCopying:  j.DeferredCopying,
				Storages:         jobStorages,
				Sources:          sources,
			})
			if err != nil {
				errs = multierror.Append(errs, err)
				continue
			}
			jobs = append(jobs, job)

		case AllowedJobTypes[4]:
			var sources []psql.SourceParams

			for _, src := range j.Sources {
				var extraKeys []string
				if len(src.ExtraKeys) > 0 {
					extraKeys = strings.Split(src.ExtraKeys, " ")
				}

				sources = append(sources, psql.SourceParams{
					ConnectParams: psql_connect.Params{
						User:        src.Connect.DBUser,
						Passwd:      src.Connect.DBPassword,
						Host:        src.Connect.DBHost,
						Port:        src.Connect.DBPort,
						Socket:      src.Connect.Socket,
						SSLMode:     src.Connect.SSLMode,
						SSLRootCert: src.Connect.SSlRootCert,
						SSLCrl:      src.Connect.SSlCrl,
					},
					Name:      src.Name,
					TargetDBs: src.TargetDBs,
					Excludes:  src.Excludes,
					Gzip:      src.Gzip,
					IsSlave:   src.IsSlave,
					ExtraKeys: extraKeys,
				})
			}

			job, err := psql.Init(psql.JobParams{
				Name:             j.JobName,
				TmpDir:           j.TmpDir,
				NeedToMakeBackup: needToMakeBackup,
				SafetyBackup:     j.SafetyBackup,
				DeferredCopying:  j.DeferredCopying,
				Storages:         jobStorages,
				Sources:          sources,
			})
			if err != nil {
				errs = multierror.Append(errs, err)
				continue
			}
			jobs = append(jobs, job)

		case AllowedJobTypes[5]:
			var sources []psql_basebackup.SourceParams

			for _, src := range j.Sources {
				var extraKeys []string
				if len(src.ExtraKeys) > 0 {
					extraKeys = strings.Split(src.ExtraKeys, " ")
				}

				sources = append(sources, psql_basebackup.SourceParams{
					ConnectParams: psql_connect.Params{
						User:        src.Connect.DBUser,
						Passwd:      src.Connect.DBPassword,
						Host:        src.Connect.DBHost,
						Port:        src.Connect.DBPort,
						Socket:      src.Connect.Socket,
						SSLMode:     src.Connect.SSLMode,
						SSLRootCert: src.Connect.SSlRootCert,
						SSLCrl:      src.Connect.SSlCrl,
					},
					Name:      src.Name,
					Gzip:      src.Gzip,
					IsSlave:   src.IsSlave,
					ExtraKeys: extraKeys,
				})
			}

			job, err := psql_basebackup.Init(psql_basebackup.JobParams{
				Name:             j.JobName,
				TmpDir:           j.TmpDir,
				NeedToMakeBackup: needToMakeBackup,
				SafetyBackup:     j.SafetyBackup,
				DeferredCopying:  j.DeferredCopying,
				Storages:         jobStorages,
				Sources:          sources,
			})
			if err != nil {
				errs = multierror.Append(errs, err)
				continue
			}
			jobs = append(jobs, job)

		case AllowedJobTypes[6]:
			var sources []mongodump.SourceParams

			for _, src := range j.Sources {
				var extraKeys []string
				if len(src.ExtraKeys) > 0 {
					extraKeys = strings.Split(src.ExtraKeys, " ")
				}

				sources = append(sources, mongodump.SourceParams{
					ConnectParams: mongo_connect.Params{
						User:   src.Connect.DBUser,
						Passwd: src.Connect.DBPassword,
						Host:   src.Connect.DBHost,
						Port:   src.Connect.DBPort,
						RSName: src.Connect.MongoRSName,
						RSAddr: src.Connect.MongoRSAddr,
					},
					Name:               src.Name,
					Gzip:               src.Gzip,
					ExtraKeys:          extraKeys,
					TargetDBs:          src.TargetDBs,
					TargetCollections:  src.TargetCollections,
					ExcludeDBs:         src.ExcludeDBs,
					ExcludeCollections: src.ExcludeCollections,
				})
			}

			job, err := mongodump.Init(mongodump.JobParams{
				Name:             j.JobName,
				TmpDir:           j.TmpDir,
				NeedToMakeBackup: needToMakeBackup,
				SafetyBackup:     j.SafetyBackup,
				DeferredCopying:  j.DeferredCopying,
				Storages:         jobStorages,
				Sources:          sources,
			})
			if err != nil {
				errs = multierror.Append(errs, err)
				continue
			}
			jobs = append(jobs, job)

		case AllowedJobTypes[7]:
			var sources []redis.SourceParams

			for _, src := range j.Sources {
				sources = append(sources, redis.SourceParams{
					ConnectParams: redis_connect.Params{
						Passwd: src.Connect.DBPassword,
						Host:   src.Connect.DBHost,
						Port:   src.Connect.DBPort,
						Socket: src.Connect.Socket,
					},
					Name: src.Name,
					Gzip: src.Gzip,
				})
			}

			job, err := redis.Init(redis.JobParams{
				Name:             j.JobName,
				TmpDir:           j.TmpDir,
				NeedToMakeBackup: needToMakeBackup,
				SafetyBackup:     j.SafetyBackup,
				DeferredCopying:  j.DeferredCopying,
				Storages:         jobStorages,
				Sources:          sources,
			})
			if err != nil {
				errs = multierror.Append(errs, err)
				continue
			}
			jobs = append(jobs, job)

		case AllowedJobTypes[8]:
			job, err := external.Init(external.JobParams{
				Name:             j.JobName,
				DumpCmd:          j.DumpCmd,
				NeedToMakeBackup: needToMakeBackup,
				SafetyBackup:     j.SafetyBackup,
				SkipBackupRotate: j.SkipBackupRotate,
				Storages:         jobStorages,
			})
			if err != nil {
				errs = multierror.Append(errs, err)
				continue
			}
			jobs = append(jobs, job)

		default:
			errs = multierror.Append(errs, fmt.Errorf("unknown job type \"%s\". Allowd types: %s", j.JobType, strings.Join(AllowedJobTypes, ", ")))
			continue
		}
	}

	return jobs, errs.ErrorOrNil()
}

func initJobStorages(storages map[string]interfaces.Storage, job jobCfg) (jobStorages interfaces.Storages, needToMakeBackup bool, errs []error) {

	for _, stOpts := range job.StoragesOptions {

		// storages validation
		s, ok := storages[stOpts.StorageName]
		if !ok {
			errs = append(errs, fmt.Errorf("%s: unknown storage name: %s", job.JobName, stOpts.StorageName))
			continue
		}

		if stOpts.Retention.Days < 0 || stOpts.Retention.Weeks < 0 || stOpts.Retention.Months < 0 {
			errs = append(errs, fmt.Errorf("%s: retention period can't be negative", job.JobName))
		}

		st := s.Clone()
		st.SetBackupPath(stOpts.BackupPath)
		st.SetRetention(storage.Retention(stOpts.Retention))

		if storage.GetNeedToMakeBackup(stOpts.Retention.Days, stOpts.Retention.Weeks, stOpts.Retention.Months) {
			needToMakeBackup = true
		}

		jobStorages = append(jobStorages, st)
	}

	// sorting storages for installing local as last
	if len(jobStorages) > 1 {
		sort.Sort(jobStorages)
	}

	return
}
