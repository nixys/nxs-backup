# Nxs-backup

Nxs-backup is an open source backup software for most popular GNU/Linux distributions. Features of Nxs-backup include
amongst others:

* Support of the most popular storages: local, s3, ssh(sftp), ftp, cifs(smb), nfs, webdav
* Database backups, such as MySQL(logical/physical), PostgreSQL(logical/physical), MongoDB, Redis
* Possibility to specify extra options for collecting database dumps to fine-tune backup process and minimize load on
  the server
* Incremental files backups
* Easy to read and maintain configuration files with clear transparent structure
* Built-in generator of the configuration files to expedite initial setup
* Support of user-defined custom scripts to extend functionality
* Possibility to restore backups with standard tools (no extra software including Nxs-backup is required)
* Email notifications about status and errors during backup process

The source code of Nxs-backup is available at https://github.com/nixys/go-nxs-backup under the license.
Additionally, Nxs-backup offers binary package repositories for the major Linux distributions (Debian, CentOS).

## Getting started

### Understanding Jobs, Type, Sources and Storages

In order to make nxs-backup as ﬂexible as possible, the directions given to nxs-backup are speciﬁed in several pieces.
The main instruction is the job resource, which deﬁnes a job. A backup job generally consists of a Type, a Sources and
Storages.
The Type deﬁnes what type of backup shall run (e.g. MySQL "physical" backups), the Sources defines the target and
exceptions (for each job at least one target must be specified), the Storages define storages where to store backups and
at what quantity (for each job at least one storage must be specified). Work with remote storage is performed by local
mounting of the FS with special tools.

### Setting Up Nxs-backup Conﬁguration Files

Nxs-backup conﬁguration ﬁles are usually located in the */etc/nxs-backup/* directory. The default configuration has only
one configuration file *nxs-backup.conf* and the *conf.d* subdirectory that stores files with descriptions of jobs (one
file per job). Config files are in YAML format. For details, see Settings.

### Generate your Configurations Files for job

You can generate your conﬁguration ﬁle for a job by running the script with the command ***generate*** and *-S*/*
--storages* (list of storages), *-T*/*--type* (type of backup), *-P*/*--path* (path to generated file) options. The
script will generate conﬁguration ﬁle for the job and print result:

 ```bash
# nxs-backup generate -S local scp -T mysql -P /etc/nxs-backup/conf.d/mysql.conf
nxs-backup: Successfully generated '/etc/nxs-backup/conf.d/mysql.conf' configuration file!
```

### Testing your Conﬁguration Files

You can test if conﬁguration is correct by running the script with the ***-t*** option and
optional *-c*/*--config* (path to main conf file). The script will process the conﬁg ﬁle and print any error
messages and then terminate:

```bash
# nxs-backup -t
nxs-backup: The configuration file '/etc/nxs-backup/nxs-backup.conf' syntax is ok!
```

### Start your jobs

You cat start your jobs by running the script with the command ***start*** and optional *-c*/*--config* (path to main
conf file). The script will execute the job passed by the argument. It should be noted that there are several reserved
job names:

+ `all` - simulates the sequential execution of *files*, *databases*, *external* job (default value)
+ `files` - random execution of all jobs with the types *desc_files*, *inc_files*
+ `databases` - random execution of all jobs with the types *mysql*, *mysql_xtrabackup*, *postgresql*, *
  postgresql_basebackup*, *mongodb*, *redis*
+ `external` - random execution of all jobs with the type *external*

```bash
# nxs-backup start all
```

## Settings

### `main`

Nxs-backup main settings block description.

| Name                      | Description                                                                            | Value                                |
|---------------------------|----------------------------------------------------------------------------------------|--------------------------------------|
| `server_name`             | The name of the server on which the nxs-backup is started                              | `""`                                 |
| `project_name`            | The name of the project, used for notifications (optional)                             | `""`                                 |
| `notifications.nxs_alert` | Contains [nxs-alert notification channel parameters](#nxs-alert-parameters)            | `{}`                                 |
| `notifications.mail`      | Contains [email notification channel parameters](#email-parameters)                    | `{}`                                 |
| `storage_connects`        | Contains list of [remote storages connections](#storage-connection-options)            | `[]`                                 |
| `jobs`                    | Contains list of [backup jobs](#backup-job-options)                                    | `[]`                                 |
| `include_jobs_configs`    | Contains list of filepaths or glob patterns to [job config files](#backup-job-options) | `["conf.d/*.conf"]`                  |
| `waiting_timeout`         | Time to waite in minutes for another nxs-backup to be completed (optional)             | `0`                                  |
| `logfile`                 | Path to log file                                                                       | `/var/log/nxs-backup/nxs-backup.log` |
| `loglevel`                | Level of messages to be logged. [Supported levels](#notification-levels)               | `info`                               |

#### Nxs-alert parameters

| Name            | Description                                                                      | Value                                        |
|-----------------|----------------------------------------------------------------------------------|----------------------------------------------|
| `enabled`       | Enables notification channel                                                     | `false`                                      |
| `auth_key`      | Nxs-alert auth key                                                               | `""`                                         |
| `nxs_alert_url` | Contains URL of the nxs-alert service                                            | `"https://nxs-alert.nixys.ru/v2/alert/pool"` |
| `message_level` | Level of messages to be notified about. [Supported levels](#notification-levels) | `"warning"`                                  |

#### Email parameters

| Name            | Description                                                                      | Value       |
|-----------------|----------------------------------------------------------------------------------|-------------|
| `enabled`       | Enables notification channel                                                     | `false`     |
| `mail_from`     | Mailbox on behalf of which mails will be sent                                    | `""`        |
| `smtp_server`   | SMTP host. If not specified email will be sent using `/usr/sbin/sendmail`        | `""`        |
| `smtp_port`     | SMTP port                                                                        | `465`       |
| `smtp_user`     | SMTP user login                                                                  | `""`        |
| `smtp_password` | SMTP user password                                                               | `""`        |
| `recipients`    | List of notifications recipients emails                                          | `[]`        |
| `message_level` | Level of messages to be notified about. [Supported levels](#notification-levels) | `"warning"` |

#### Notification levels

| Name      | Description                                                          |
|-----------|----------------------------------------------------------------------|
| `debug`   | The most detailed information about the backup process               |
| `info`    | General information about the backup process                         |
| `warning` | Information about the backup process that requires special attention |
| `error`   | Only critical information about failures in the backup process       |

#### Storage connection options

Nxs-backup storage connect settings block description.

| Name            | Description                                                                           | Value |
|-----------------|---------------------------------------------------------------------------------------|-------|
| `name`          | Unique storage name                                                                   | `""`  |
| `s3_params`     | Connection parameters for [S3 storage type](#s3-connection-params) (optional)         | `{}`  |
| `scp_params`    | Connection parameters for [scp/sftp storage type](#sftp-connection-params) (optional) | `{}`  |
| `ftp_params`    | Connection parameters for [ftp storage type](#ftp-connection-params) (optional)       | `{}`  |
| `nfs_params`    | Connection parameters for [nfs storage type](#nfs-connection-params) (optional)       | `{}`  |
| `smb_params`    | Connection parameters for [smb/cifs storage type](#smb-connection-params) (optional)  | `{}`  |
| `webdav_params` | Connection parameters for [webdav storage type](#webdav-connection-params) (optional) | `{}`  |

#### S3 connection params

| Name                | Description    | Value |
|---------------------|----------------|-------|
| `bucket_name`       | S3 bucket name | `""`  |
| `endpoint`          | S3 endpoint    | `""`  |
| `region`            | S3 region      | `""`  |
| `access_key_id`     | S3 access key  | `""`  |
| `secret_access_key` | S3 secret key  | `""`  |

#### SFTP connection params

| Name                 | Description                                 | Value |
|----------------------|---------------------------------------------|-------|
| `host`               | SSH host                                    | `""`  |
| `port`               | SSH port (optional)                         | `22`  |
| `user`               | SSH user                                    | `""`  |
| `password`           | SSH password                                | `""`  |
| `key_file`           | Path to SSH private key instead of password | `""`  |
| `connection_timeout` | SSH connection timeout seconds (optional)   | `10`  |

#### FTP connection params

| Name                 | Description                                        | Value |
|----------------------|----------------------------------------------------|-------|
| `host`               | FTP host                                           | `""`  |
| `port`               | FTP port (optional)                                | `21`  |
| `user`               | FTP user                                           | `""`  |
| `password`           | FTP password                                       | `""`  |
| `connect_count`      | Count of FTP connections opens to sever (optional) | `5`   |
| `connection_timeout` | FTP connection timeout seconds (optional)          | `10`  |

#### NFS connection params

| Name     | Description                                     | Value  |
|----------|-------------------------------------------------|--------|
| `host`   | NFS host                                        | `""`   |
| `port`   | NFS port (optional)                             | `111`  |
| `target` | Path on NFS server where backups will be stored | `""`   |
| `UID`    | UID of NFS server user (optional)               | `1000` |
| `GID`    | GID of NFS server user (optional)               | `1000` |

#### SMB connection params

| Name                 | Description                               | Value     |
|----------------------|-------------------------------------------|-----------|
| `host`               | SMB host                                  | `""`      |
| `port`               | SMB port (optional)                       | `445`     |
| `user`               | SMB user (optional)                       | `"Guest"` |
| `password`           | SMB password (optional)                   | `""`      |
| `share`              | SMB share name                            | `5`       |
| `domain`             | SMB domain (optional)                     | `5`       |
| `connection_timeout` | SMB connection timeout seconds (optional) | `10`      |

#### WebDav connection params

| Name                 | Description                                  | Value |
|----------------------|----------------------------------------------|-------|
| `url`                | WebDav URL                                   | `""`  |
| `username`           | WebDav user                                  | `""`  |
| `password`           | WebDav password                              | `""`  |
| `oauth_token`        | WebDav OAuth token (optional)                | `""`  |
| `connection_timeout` | WebDav connection timeout seconds (optional) | `10`  |

### Backup job options

Nxs-backup job settings block description.

| Name                 | Description                                                                                                                                                                                                                                                                     | Value   |
|----------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------|
| `job`                | Job name. This value is used to run the specific job                                                                                                                                                                                                                            | `""`    |
| `type`               | Backup type. [Supported backup types](#backup-types)                                                                                                                                                                                                                            | `""`    |
| `tmp_dir`            | A local path to the directory for temporary backups files                                                                                                                                                                                                                       | `""`    |
| `safety_backup`      | Delete outdated backups after creating a new one. **IMPORTANT** Using of this option requires more disk space.<br> Perform sure there is enough free space on the device where temporary backups stores                                                                         | `false` |
| `deferred_copying`   | Determines that copying of backups to remote storages occurs after creation of all temporary backups defined in the task.<br> **IMPORTANT** Using of this option requires more disk space. Perform sure there is enough free space on the device where temporary backups stores | `false` |
| `sources`            | Specify a list of [source objects](#source-parameters) for backup                                                                                                                                                                                                               | `[]`    |
| `storages_options`   | Specify a list of [storages](#storage-options) to store backups                                                                                                                                                                                                                 | `[]`    |
| `dump_cmd`           | Full command to run an external script. **Only for *external* backup type**                                                                                                                                                                                                     | `""`    |
| `skip_backup_rotate` | Skip backup rotation on storages. **Only for *external* backup type**                                                                                                                                                                                                           | `false` |

Option `skip_backup_rotate` may be used if creation of a local copy is not required. For example, in case when script
copying data to a remote server, rotation of backups may be skipped with this option.

#### Source parameters

| Name                  | Description                                                                                                                                                                      | Value   |
|-----------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------|
| `name`                | Used to differentiate backups in the target directory                                                                                                                            | `""`    |
| `connect`             | Defines a [set of parameters](#database-connection-params) for connecting to the database. **Only for [*databases*](#database-types) types**                                     | `{}`    |
| `targets`             | List of directories/files to be backed up. Glob patterns are supported                                                                                                           | `[]`    |
| `target_dbs`          | List of databases to be backed up. Use the keyword **all** for backup all databases. **Only for [*databases*](#database-types) types**                                           | `[]`    |
| `target_collections`  | List of collections to be backed up. Use the keyword **all** for backup all collections in all dbs. **Only for *mongodb* type**                                                  | `[]`    |
| `excludes`            | List of databases/schemas/tables or directories/files to be excluded from backup. Glob patterns are supported for [*file*](#file-types) types                                    | `[]`    |
| `exclude_dbs`         | List of databases to be excluded from backup. **Only for *mongodb* type**                                                                                                        | `[]`    |
| `exclude_collections` | List of collections to be excluded from backup. **Only for *mongodb* type**                                                                                                      | `[]`    |
| `db_extra_keys`       | Special parameters for the collecting database backups. **Only for [*databases*](#database-types) types**                                                                        | `""`    |
| `gzip`                | Whether you need to compress the backup file                                                                                                                                     | `false` |
| `save_abs_path`       | Whether you need to save absolute path in tar archives **Only for [*file*](#file-types) types**                                                                                  | `true`  |
| `prepare_xtrabackup`  | Whether you need to make [xtrabackup prepare](https://www.percona.com/doc/percona-xtrabackup/2.2/xtrabackup_bin/preparing_the_backup.html). **Only for *mysql_xtrabackup* type** | `true`  |

#### Database connection params

| Name                        | Description                                       | Value       |
|-----------------------------|---------------------------------------------------|-------------|
| `db_host`                   | DB host                                           | `""`        |
| `db_port`                   | DB port                                           | `""`        |
| `socket`                    | Path to DB socket                                 | `""`        |
| `db_user`                   | DB user                                           | `""`        |
| `db_password`               | DB password                                       | `""`        |
| `mysql_auth_file`           | Path to MySQL auth file                           | `""`        |
| `psql_ssl_mode`             | PostgreSQL SSL mode option                        | `"require"` |
| `mongo_replica_set_name`    | MongoDB replicaset name                           | `""`        |
| `mongo_replica_set_address` | Comma separated list of MongoDB replicaset hosts  | `""`        |

You may use either `auth_file` or `db_host` or `socket` options. Options priority follows:
`auth_file` → `db_host` → `socket`

#### Storage options

| Name           | Description                                                                           | Value |
|----------------|---------------------------------------------------------------------------------------|-------|
| `storage_name` | The name of storage, defined in main config. ***local* storage available by default** | `""`  |
| `backup_path`  | Path to directory for storing backups                                                 | `""`  |
| `retention`    | Defines [retention](#storage-retention) for backups on current storage                | `{}`  |

#### Storage retention

| Name    | Description                                                                                                                                                                          | Value |
|---------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------|
| `days`  | Days to store backups                                                                                                                                                                | `7`   |
| `weeks` | Weeks to store backups                                                                                                                                                               | `5`   |
| `month` | Months to store backups. For *inc_files* backup type determines how many months of incremental copies<br> will be stored relative to the current month. Can take values from 0 to 12 | `12`  |

#### Backup types

##### Database types

| Name                    | Description                |
|-------------------------|----------------------------|
| `mysql`                 | MySQL logical backup       |
| `mysql_xtrabackup`      | MySQL physical backup      |
| `postgresql`            | PostgreSQL logical backup  |
| `postgresql_basebackup` | PostgreSQL physical backup |
| `mongodb`               | MongoDB backup             |
| `redis`                 | Redis backup               |

##### File types

| Name                    | Description                |
|-------------------------|----------------------------|
| `desc_files`            | Files discrete backup      |
| `inc_files`             | Files incremental backup   |

##### Other types

| Name                    | Description                |
|-------------------------|----------------------------|
| `external`              | External backup script     |

## Useful information

### Desc files nxs-backup module

Identical to creating a backup using `tar`.

### Incremental files nxs-backup module

Identical to creating a backup using `tar`.

Incremental copies of files are made according to the following scheme:
![Incremental backup scheme](https://image.ibb.co/dtLn2p/nxs_inc_backup_scheme_last_version.jpg)

At the beginning of the year or on the first start of nxs-backup, a full initial backup is created. Then at the
beginning of each month - an incremental monthly copy from a yearly copy is created. Inside each month there are
incremental ten-day copies. Within each ten-day copy incremental day copies are created.

In this case, since now the tar file is in the PAX format, when you deploy the incremental backup, you do not need to
specify the path to inc-files. All the info is stored in the PAX header of the `GNU.dumpdir` directory inside the
archive.
Therefore, the commands to restore a backup for a specific date are the following:

* First, unpack the full year copy with the follow command:

```bash
tar xf /path/to/full/year/backup
```

* Then alternately unpack the monthly, ten-day, day incremental backups, specifying a special key -G, for example:

```bash
tar xGf /path/to/monthly/backup
tar xGf /path/to/ten-day/backup
tar xGf /path/to/day/backup
```

### MySQL(logical) nxs-backup module

Works on top of `mysqldump`, so for the correct work of the module you have to install compatible **mysql-client**.

### MySQL(physical) nxs-backup module

Works on top of `xtrabackup`, so for the correct work of the module you have to install compatible **
percona-xtrabackup**. *Supports only backup of local instance*.

### PostgreSQL(logical) nxs-backup module

Works on top of `pg_dump`, so for the correct work of the module you have to install compatible **postgresql-client**.

### PostgreSQL(physical) nxs-backup module

Works on top of `pg_basebackup`, so for the correct work of the module you have to install compatible **
postgresql-client**.

### MongoDB nxs-backup module

Works on top of `mongodump`, so for the correct work of the module you have to install compatible **
mongodb-clients**.

### Redis nxs-backup module

Works on top of `redis-cli with --rdb option`, so for the correct work of the module you have to install compatible **
redis-tools**.

### External nxs-backup module

In this module, an external script is executed passed to the program via the key "dump_cmd".  
By default at the completion of this command, it is expected that:

* A complete backup file with data will be collected
* The stdout will send data in json format, like:

```json
{
  "full_path": "/abs/path/to/backup.file"
}
```

IMPORTANT:

* make sure that there is no unnecessary information in stdout
* the successfully completed program must exit with 0

If the module used with the `skip_backup_rotate` parameter, the standard output is expected as a result of running
the command. For example, when executing the command "rsync -Pavz /local/source /remote/destination" the result is expected to be a
standard output to stdout.
