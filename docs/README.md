# Documentation

## Supported versions and requirements

nxs-backup can be run on any GNU/Linux distribution with a kernel above 2.6. The set of dependencies depends on what
exactly you want to back up.

### Files backups

To make backups of your files, you have to ensure that you have **GNU tar** of whatever version is available on your OS.
More about incremental files backup you can find in our [documentation](INCREMENTAL_FILES_BACKUP.md)

### MySQL/Mariadb/Percona backups

For regular backups is used `mysqldump`. Therefore, you have to ensure that you have a version of `mysqldump` that is
compatible with your database.

For physical files backups is used Percona `xtrabackup`. So, you have to ensure that you have a compatible with your
database version of Percona `xtrabackup`. *Supports only backup of local database instance*.

### PostgreSQL backups

For regular and physical backups is used `pg_dump`. You have to ensure that you have a version of `pg_dump` that is
compatible with your database version.

For physical files backups is used `pg_basebackup`. So, you have to ensure that you have a compatible with your
database version of Percona `pg_basebackup`.

### MongoDB backups

For backups of MongoDB is used `mongodump` tool. You have to ensure that you have a version of `mongodump` that is
compatible with your database version.

### Redis backups

For backups of Redis is used `redis-cli` tool. You have to ensure that you have a version of `redis-cli` that is
compatible with your Redis version.


## Configure

You can find example of configuration files for different deployment kinds [here](example/README.md).

## Settings

Full list of setting parameters available in [our documentation](settings/README.md).

## Useful information

Some useful info can be found [here](USEFUL_INFO.md).

## Restore from backup

As built-in backups restoring tools are under development. You can discover a few tricks
in [our documentation](restore)
