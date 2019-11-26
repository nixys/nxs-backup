# Changelog
All notable changes to this project will be documented in this file.

## [Unreleased]

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