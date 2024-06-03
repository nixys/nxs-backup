package ctx

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/go-multierror"

	"github.com/nixys/nxs-backup/ds/mongo_connect"
	"github.com/nixys/nxs-backup/ds/mysql_connect"
	"github.com/nixys/nxs-backup/ds/psql_connect"
	"github.com/nixys/nxs-backup/ds/redis_connect"
	"github.com/nixys/nxs-backup/interfaces"
	"github.com/nixys/nxs-backup/misc"
	"github.com/nixys/nxs-backup/modules/backup/desc_files"
	"github.com/nixys/nxs-backup/modules/backup/external"
	"github.com/nixys/nxs-backup/modules/backup/inc_files"
	"github.com/nixys/nxs-backup/modules/backup/mongodump"
	"github.com/nixys/nxs-backup/modules/backup/mysql"
	"github.com/nixys/nxs-backup/modules/backup/mysql_xtrabackup"
	"github.com/nixys/nxs-backup/modules/backup/psql"
	"github.com/nixys/nxs-backup/modules/backup/psql_basebackup"
	"github.com/nixys/nxs-backup/modules/backup/redis"
	"github.com/nixys/nxs-backup/modules/metrics"
	"github.com/nixys/nxs-backup/modules/storage"
)

type jobsOpts struct {
	metricsData    *metrics.Data
	oldMetricsData *metrics.Data
	jobs           []jobCfg
	storages       map[string]interfaces.Storage
}

func jobsInit(o jobsOpts) ([]interfaces.Job, error) {
	var (
		errs *multierror.Error
		jobs []interfaces.Job
	)

	for _, j := range o.jobs {
		var needToMakeBackup bool
		var jobStorages interfaces.Storages
		stErrs := 0

		if len(j.JobName) == 0 {
			errs = multierror.Append(errs, fmt.Errorf("Empty job name is unacceptable "))
			continue
		}

		if misc.Contains([]string{"files", "databases", "external"}, j.JobName) {
			errs = multierror.Append(errs, fmt.Errorf("A job cannot have the name `%s` reserved", j.JobName))
			continue
		}

		for _, opt := range j.StoragesOptions {

			// storages validation
			s, ok := o.storages[opt.StorageName]
			if !ok {
				stErrs++
				errs = multierror.Append(errs, fmt.Errorf("Failed to set storage `%s` for job `%s`: storage not available ", opt.StorageName, j.JobName))
				continue
			}

			if opt.Retention.Days < 0 || opt.Retention.Weeks < 0 || opt.Retention.Months < 0 {
				stErrs++
				errs = multierror.Append(errs, fmt.Errorf("Failed to set storage `%s` for job `%s`: retention period can't be negative ", opt.StorageName, j.JobName))
				continue
			}

			st := s.Clone()
			st.SetBackupPath(opt.BackupPath)
			st.SetRetention(storage.Retention(opt.Retention))

			if storage.GetNeedToMakeBackup(opt.Retention.Days, opt.Retention.Weeks, opt.Retention.Months) {
				needToMakeBackup = true
			}

			jobStorages = append(jobStorages, st)
		}

		// sorting storages for installing local as last
		if len(jobStorages) > 1 {
			sort.Sort(jobStorages)
		}

		switch j.JobType {
		case string(misc.DescFiles):
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
				Metrics:          o.metricsData,
				OldMetrics:       o.oldMetricsData,
			})
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("Failed to init job `%s` with error: %w ", j.JobName, err))
				continue
			}

			jobs = append(jobs, job)

		case string(misc.IncFiles):
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
				Metrics:         o.metricsData,
				OldMetrics:      o.oldMetricsData,
			})
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("Failed to init job `%s` with error: %w ", j.JobName, err))
				continue
			}

			jobs = append(jobs, job)

		case string(misc.Mysql):
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
				Metrics:          o.metricsData,
				OldMetrics:       o.oldMetricsData,
			})
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("Failed to init job `%s` with error: %w ", j.JobName, err))
				continue
			}
			jobs = append(jobs, job)

		case string(misc.MysqlXtrabackup):
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
				Metrics:          o.metricsData,
				OldMetrics:       o.oldMetricsData,
			})
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("Failed to init job `%s` with error: %w ", j.JobName, err))
				continue
			}
			jobs = append(jobs, job)

		case string(misc.Postgresql):
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
						SSLMode:     src.Connect.PsqlSSLMode,
						SSLRootCert: src.Connect.PsqlSSlRootCert,
						SSLCrl:      src.Connect.PsqlSSlCrl,
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
				Metrics:          o.metricsData,
				OldMetrics:       o.oldMetricsData,
			})
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("Failed to init job `%s` with error: %w ", j.JobName, err))
				continue
			}
			jobs = append(jobs, job)

		case string(misc.PostgresqlBasebackup):
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
						SSLMode:     src.Connect.PsqlSSLMode,
						SSLRootCert: src.Connect.PsqlSSlRootCert,
						SSLCrl:      src.Connect.PsqlSSlCrl,
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
				Metrics:          o.metricsData,
				OldMetrics:       o.oldMetricsData,
			})
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("Failed to init job `%s` with error: %w ", j.JobName, err))
				continue
			}
			jobs = append(jobs, job)

		case string(misc.Mongodb):
			var sources []mongodump.SourceParams

			for _, src := range j.Sources {
				var extraKeys []string
				if len(src.ExtraKeys) > 0 {
					extraKeys = strings.Split(src.ExtraKeys, " ")
				}

				sources = append(sources, mongodump.SourceParams{
					ConnectParams: mongo_connect.Params{
						User:      src.Connect.DBUser,
						Passwd:    src.Connect.DBPassword,
						Host:      src.Connect.DBHost,
						Port:      src.Connect.DBPort,
						RSName:    src.Connect.MongoRSName,
						RSAddr:    src.Connect.MongoRSAddr,
						TLSCAFile: src.Connect.MongoTLSCAFile,
						AuthDB:    src.Connect.MongoAuthDB,
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
				Metrics:          o.metricsData,
				OldMetrics:       o.oldMetricsData,
			})
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("Failed to init job `%s` with error: %w ", j.JobName, err))
				continue
			}
			jobs = append(jobs, job)

		case string(misc.Redis):
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
				Metrics:          o.metricsData,
				OldMetrics:       o.oldMetricsData,
			})
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("Failed to init job `%s` with error: %w ", j.JobName, err))
				continue
			}
			jobs = append(jobs, job)

		case string(misc.External):
			job, err := external.Init(external.JobParams{
				Name:             j.JobName,
				DumpCmd:          j.DumpCmd,
				NeedToMakeBackup: needToMakeBackup,
				SafetyBackup:     j.SafetyBackup,
				SkipBackupRotate: j.SkipBackupRotate,
				Storages:         jobStorages,
				Metrics:          o.metricsData,
				OldMetrics:       o.oldMetricsData,
			})
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("Failed to init job `%s` with error: %w ", j.JobName, err))
				continue
			}
			jobs = append(jobs, job)

		default:
			errs = multierror.Append(errs, fmt.Errorf("Unknown job type \"%s\". Allowd types: %s ", j.JobType, strings.Join(misc.AllowedBackupTypesGet(), ", ")))
			continue
		}
	}

	return jobs, errs.ErrorOrNil()
}
