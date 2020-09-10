# Changelog
All notable changes to this project will be documented in this file.

## [Unreleased]

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
