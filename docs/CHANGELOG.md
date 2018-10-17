# Changelog
All notable changes to this project will be documented in this file.

## [Unreleased]

## [2.1.0] - 2018-10-17
### Added
- Download backup on custom S3 server
- External SMTP send
- Handling unexpected exceptions

### Changed
- Rpm backages build
- Improved algorihtm for collecting incremental files backups
- Improved control of the number of processes in the system via flock()
- No buffering for log file
- Renamed type mysql_xtradb with mysql_xtrabackup

### Fixed
- Unexpected program termination due to umount error
- Prelink breaks binaries
- Closing the log file when the program ends
