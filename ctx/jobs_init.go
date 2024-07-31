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
	metricsData *metrics.Data
	mainLim     *limitsConf
	jobs        []jobConf
	storages    map[string]interfaces.Storage
}

func jobsInit(o jobsOpts) ([]interfaces.Job, error) {
	var (
		errs *multierror.Error
		job  interfaces.Job
		jobs []interfaces.Job
	)

	for _, j := range o.jobs {
		var (
			needToMakeBackup bool
			withStorageRate  bool
			diskRate         int64
			nrl              int64
			stErrs           = 0
			err              error
			jobStorages      interfaces.Storages
		)

		if len(j.Name) == 0 {
			errs = multierror.Append(errs, fmt.Errorf("Empty job name is unacceptable "))
			continue
		}

		if misc.Contains([]string{"files", "databases", "external"}, j.Name) {
			errs = multierror.Append(errs, fmt.Errorf("A job cannot have the name `%s` reserved", j.Name))
			continue
		}

		diskRate, err = getRateLimit(o.mainLim.DiskRate)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("%s The job `%s` won't be use limit defined on job level for its storages", err, j.Name))
		}

		if j.Limits != nil {
			if j.Limits.NetRate != nil {
				nrl, err = getRateLimit(j.Limits.NetRate)
				if err != nil {
					errs = multierror.Append(errs, fmt.Errorf("%s The job `%s` won't be use limit defined on job level for its storages", err, j.Name))
				} else {
					withStorageRate = true
				}
			}
			if j.Limits.DiskRate != nil {
				diskRate, err = getRateLimit(j.Limits.DiskRate)
				if err != nil {
					errs = multierror.Append(errs, fmt.Errorf("%s The job `%s` won't be use limit defined on job level for its storages", err, j.Name))
				}
			}
		}

		for _, opt := range j.StoragesOptions {

			// storages validation
			s, ok := o.storages[opt.StorageName]
			if !ok {
				stErrs++
				errs = multierror.Append(errs, fmt.Errorf("Failed to set storage `%s` for job `%s`: storage not available ", opt.StorageName, j.Name))
				continue
			}

			if opt.Retention.Days < 0 || opt.Retention.Weeks < 0 || opt.Retention.Months < 0 {
				stErrs++
				errs = multierror.Append(errs, fmt.Errorf("Failed to set storage `%s` for job `%s`: retention period can't be negative ", opt.StorageName, j.Name))
				continue
			}

			st := s.Clone()
			stParams := storage.Params{
				BackupPath:    opt.BackupPath,
				RotateEnabled: opt.EnableRotate,
				Retention:     storage.Retention(opt.Retention),
			}
			if opt.StorageName == "local" {
				stParams.RateLimit = diskRate
			} else if withStorageRate {
				stParams.RateLimit = nrl
			}
			st.Configure(stParams)

			if storage.IsNeedToBackup(opt.Retention.Days, opt.Retention.Weeks, opt.Retention.Months) {
				needToMakeBackup = true
			}

			jobStorages = append(jobStorages, st)
		}

		// sorting storages for installing local as last
		if len(jobStorages) > 1 {
			sort.Sort(jobStorages)
		}

		switch j.Type {
		case misc.DescFiles:
			var sources []desc_files.SourceParams

			for _, src := range j.Sources {
				sources = append(sources, desc_files.SourceParams{
					Name:        src.Name,
					Targets:     src.Targets,
					Excludes:    src.Excludes,
					SaveAbsPath: src.SaveAbsPath,
					Gzip:        isGzip(src.Gzip, j.Gzip),
				})
			}

			job, err = desc_files.Init(desc_files.JobParams{
				Name:             j.Name,
				TmpDir:           j.TmpDir,
				NeedToMakeBackup: needToMakeBackup,
				SafetyBackup:     j.SafetyBackup,
				DeferredCopying:  j.DeferredCopying,
				DiskRateLimit:    diskRate,
				Storages:         jobStorages,
				Sources:          sources,
				Metrics:          o.metricsData,
			})

		case misc.IncFiles:
			var sources []inc_files.SourceParams

			for _, src := range j.Sources {
				sources = append(sources, inc_files.SourceParams{
					Name:        src.Name,
					Targets:     src.Targets,
					Excludes:    src.Excludes,
					SaveAbsPath: src.SaveAbsPath,
					Gzip:        isGzip(src.Gzip, j.Gzip),
				})
			}

			job, err = inc_files.Init(inc_files.JobParams{
				Name:            j.Name,
				TmpDir:          j.TmpDir,
				SafetyBackup:    j.SafetyBackup,
				DeferredCopying: j.DeferredCopying,
				DiskRateLimit:   diskRate,
				Storages:        jobStorages,
				Sources:         sources,
				Metrics:         o.metricsData,
			})

		case misc.Mysql:
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
					IsSlave:   src.IsSlave,
					ExtraKeys: extraKeys,
					Gzip:      isGzip(src.Gzip, j.Gzip),
				})
			}

			job, err = mysql.Init(mysql.JobParams{
				Name:             j.Name,
				TmpDir:           j.TmpDir,
				NeedToMakeBackup: needToMakeBackup,
				SafetyBackup:     j.SafetyBackup,
				DeferredCopying:  j.DeferredCopying,
				DiskRateLimit:    diskRate,
				Storages:         jobStorages,
				Sources:          sources,
				Metrics:          o.metricsData,
			})

		case misc.MysqlXtrabackup:
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
					IsSlave:   src.IsSlave,
					Prepare:   src.PrepareXtrabackup,
					ExtraKeys: extraKeys,
					Gzip:      isGzip(src.Gzip, j.Gzip),
				})
			}

			job, err = mysql_xtrabackup.Init(mysql_xtrabackup.JobParams{
				Name:             j.Name,
				TmpDir:           j.TmpDir,
				NeedToMakeBackup: needToMakeBackup,
				SafetyBackup:     j.SafetyBackup,
				DeferredCopying:  j.DeferredCopying,
				DiskRateLimit:    diskRate,
				Storages:         jobStorages,
				Sources:          sources,
				Metrics:          o.metricsData,
			})

		case misc.Postgresql:
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
					IsSlave:   src.IsSlave,
					ExtraKeys: extraKeys,
					Gzip:      isGzip(src.Gzip, j.Gzip),
				})
			}

			job, err = psql.Init(psql.JobParams{
				Name:             j.Name,
				TmpDir:           j.TmpDir,
				NeedToMakeBackup: needToMakeBackup,
				SafetyBackup:     j.SafetyBackup,
				DeferredCopying:  j.DeferredCopying,
				DiskRateLimit:    diskRate,
				Storages:         jobStorages,
				Sources:          sources,
				Metrics:          o.metricsData,
			})

		case misc.PostgresqlBasebackup:
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
					IsSlave:   src.IsSlave,
					ExtraKeys: extraKeys,
					Gzip:      isGzip(src.Gzip, j.Gzip),
				})
			}

			job, err = psql_basebackup.Init(psql_basebackup.JobParams{
				Name:             j.Name,
				TmpDir:           j.TmpDir,
				NeedToMakeBackup: needToMakeBackup,
				SafetyBackup:     j.SafetyBackup,
				DeferredCopying:  j.DeferredCopying,
				DiskRateLimit:    diskRate,
				Storages:         jobStorages,
				Sources:          sources,
				Metrics:          o.metricsData,
			})

		case misc.MongoDB:
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
					ExtraKeys:          extraKeys,
					TargetDBs:          src.TargetDBs,
					TargetCollections:  src.TargetCollections,
					ExcludeDBs:         src.ExcludeDBs,
					ExcludeCollections: src.ExcludeCollections,
					Gzip:               isGzip(src.Gzip, j.Gzip),
				})
			}

			job, err = mongodump.Init(mongodump.JobParams{
				Name:             j.Name,
				TmpDir:           j.TmpDir,
				NeedToMakeBackup: needToMakeBackup,
				SafetyBackup:     j.SafetyBackup,
				DeferredCopying:  j.DeferredCopying,
				DiskRateLimit:    diskRate,
				Storages:         jobStorages,
				Sources:          sources,
				Metrics:          o.metricsData,
			})

		case misc.Redis:
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
					Gzip: isGzip(src.Gzip, j.Gzip),
				})
			}

			job, err = redis.Init(redis.JobParams{
				Name:             j.Name,
				TmpDir:           j.TmpDir,
				NeedToMakeBackup: needToMakeBackup,
				SafetyBackup:     j.SafetyBackup,
				DeferredCopying:  j.DeferredCopying,
				DiskRateLimit:    diskRate,
				Storages:         jobStorages,
				Sources:          sources,
				Metrics:          o.metricsData,
			})

		case misc.External:
			if j.SkipBackupRotate {
				errs = multierror.Append(errs, fmt.Errorf("Used deprecated option `skip_backup_rotate` for job \"%s\". Use `storages_options[].enable_rotate` instead. ", j.Name))
			}
			job, err = external.Init(external.JobParams{
				Name:             j.Name,
				DumpCmd:          j.DumpCmd,
				NeedToMakeBackup: needToMakeBackup,
				SafetyBackup:     j.SafetyBackup,
				SkipBackupRotate: j.SkipBackupRotate,
				DiskRateLimit:    diskRate,
				Storages:         jobStorages,
				Metrics:          o.metricsData,
				Gzip:             j.Gzip,
			})

		default:
			errs = multierror.Append(errs, fmt.Errorf("Unknown job type \"%s\". Allowd types: %s ", j.Type, strings.Join(misc.AllowedBackupTypesList(), ", ")))
			continue
		}

		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("Failed to init job `%s` with error: %w ", j.Name, err))
		} else {
			jobs = append(jobs, job)
		}

	}

	return jobs, errs.ErrorOrNil()
}

func isGzip(sgz *bool, jgz bool) bool {
	if sgz != nil {
		return *sgz
	} else {
		return jgz
	}
}
