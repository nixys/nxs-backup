# Changelog
All notable changes to this project will be documented in this file.

## [Unreleased]

## [2.5.6] - 2021-08-05
### Fix
- Fixed decremental backups schedule bugs.

## [2.5.5] - 2021-06-24
### Fix
- Added `--single-transaction` flag for mysql backup

## [2.5.4] - 2021-04-21
### Fix
- Fixed error "Can't mount remote 's3' storage: shell-init: error retrieving current directory: getcwd: cannot access parent directories: No such file or directory" for s3 storage with many targets
- Fixed error "Can't umount remote 'local' storage:shell-init: error retrieving current directory: getcwd: cannot access parent directories: No such file or directory"

## [2.5.3] - 2021-04-05
### Fix
- Fixed handling of the error with incorrect connection to mysql.

## [2.5.2] - 2021-03-22
### Fix
- Fixed storing of decremental backups.

## [2.5.1] - 2021-02-26
### Fix
- Fixed `postgresql` backup with some messages in output of pg_dump.

## [2.5.0] - 2021-01-29
### Adds
- Added the `inc_months_to_store` parameter for incremental copies, allowing you to specify how many months with copies will be stored.
- Fixed backup file extension for `postgresql` backup type

## [2.4.6] - 2021-01-15
### Fix
- Fixed creating of the .inc files for `inc_files` backups type with `scp` storage 

## [2.4.5] - 2020-12-23
### Fix
- Fixed creating of the symlinks for `inc_files` backups type with `scp` storage 
- Fixed error "Missing required key:''tmp_dir''" for type=inc_files ([issue/22](https://github.com/nixys/nxs-backup/issues/22))

## [2.4.4] - 2020-12-17
### Fix
- Fixed `scp` mount fot `inc_files` backups type

## [2.4.3] - 2020-12-01
### Fix
- Fixed `scp` mount without `remote_mount_point` parameter

## [2.4.2] - 2020-10-08
### Adds
- The STDERR output of the External Block has been added to the logs.

## [2.4.1] - 2020-10-07
### Fix
- Fixed building for Debian 10 Buster.  

## [2.4.0] - 2020-10-02
### Adds
- The capabilities of the External block are extended by the parameter `skip_backup_rotate`.  
- Added support of Debian 10 Buster.  

## [2.3.0] - 2020-09-15
### Adds
- Added possibility to use already mounted by ssh the same remote resource.

## [2.2.1] - 2020-09-13
### Fix
- Fixed creation of of files and mpngodb backups

## [2.2.0] - 2020-09-10
### Adds
- Added possibility of deferred copying of created temporary backups

## [2.1.13] - 2020-08-04
### Adds
- Fixed error with unmount of local storage

## [2.1.12] - 2020-08-03
### Adds
- Added timeout for waiting of completion already ran script
- Added the ability to create new backups on remote storages safely

## [2.1.11] - 2020-07-29
### Adds
- Scp and nfs mount point option

## [2.1.10] - 2020-01-10
### Fix
- Typo correction in the mount module

## [2.1.9] - 2019-12-10
### Fix
- Improved algorihtm for exclude files in desc backups

## [2.1.8] - 2019-12-06
### Fix
- Fixed a problem with the desc-backup module, when an object deleted after checks for the presence of a file on disk and archive collection process was completed. Now this exception will be handled.

## [2.1.7] - 2019-12-03
### Adds
- Ability to recursively exclude files

## [2.1.6] - 2019-11-28
### Change
- Support for Debian 7,8 has been stopped
- Update code for version Python 3.6

## [2.1.5] - 2019-11-26
### Change
- Support for Centos 6 has been stopped

## [2.1.4] - 2019-11-26
### Change
- Change Python version to 3.6 for CentOS 7

## [2.1.3] - 2019-11-01
### Change
- Change version for build

## [2.1.2] - 2019-11-01
### Fixes
- Fixed a problem with the desc-backup module, when an object deleted after listing files and archive collection process was completed. Now the desc-backup module checks for the presence of a file on disk before adding it to the archive.

## [2.1.1] - 2018-10-18
### Fixes
- Fixed database jobs processing
- Removed packages installation from code

## [2.1.0] - 2018-10-17
### Adds
- Ability to upload backup on custom S3 server
- Ability to use external SMTP servers
- Handling unexpected exceptions

### Changes
- Improved algorihtm for create incremental files backups
- Improved control for duplicated nxs-backup processes
- Disabled log messages bufferization
- Renamed type `mysql_xtradb` to `mysql_xtrabackup`

### Fixes
- Rpm packages build
- Unexpected program termination due to umount error
- Prelink breaks nxs-backup binary
- Closing log-file after nxs-backup termination
