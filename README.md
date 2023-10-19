# nxs-backup

nxs-backup is a tool for creating and rotating backups locally and on remote storages, compatible with GNU/Linux
distributions.

## Introduction

### Features

* Full data backup
* Discrete and incremental files backups
* Upload and manage backups to the remote storages:
    * S3 (Simple Storage Service that provides object storage through a web interface. Supported by clouds e.g. AWS,
      GCP)
    * ssh (sftp)
    * ftp
    * cifs (smb)
    * nfs
    * WebDAV
* Database backups:
    * Regular backups of MySQL/Mariadb/Percona (5.7/8.0/_all versions_)
    * Xtrabackup (2.4/8.0) of MySQL/Mariadb/Percona (5.7/8.0/all versions)
    * Regular backups of PostgreSQL (9/10/11/12/13/14/15/16/_all versions_)
    * Basebackups of PostgreSQL (9/10/11/12/13/14/15/_all versions_)
    * Backups of MongoDB (3.0/3.2/3.4/3.6/4.0/4.2/4.4/5.0/6.0/7.0/_all versions_)
    * Backups of Redis (_all versions_)
* Fine-tune the database backup process with additional options for optimization purposes
* Notifications via email and webhooks about events in the backup process
* Built-in generator of the configuration files to expedite initial setup
* Easy to read and maintain configuration files with clear transparent structure
* Possibility to restore backups with standard file/database tools (nxs-backup is not required)
* Support of user-defined scripts that extend functionality
* Support of Environment variables in config files

### Who can use the tool?

* System Administrators
* DevOps Engineers
* Developers
* Anybody who need to do regular backups

## IMPORTANT! Read it before start

### Supported versions and requirements

nxs-backup can be run on any GNU/Linux distribution with a kernel above 2.6. The set of dependencies depends on what
exactly you want to back up.

#### Files backups

To make backups of your files, you have to ensure that you have **GNU tar** of whatever version is available on your OS.
More about incremental files backup you can find in our [documentation](docs/INCREMENTAL_FILES_BACKUP.MD)

#### MySQL/Mariadb/Percona backups

For regular backups is used `mysqldump`. Therefore, you have to ensure that you have a version of `mysqldump` that is
compatible with your database.

For physical files backups is used Percona `xtrabackup`. So, you have to ensure that you have a compatible with your
database version of Percona `xtrabackup`. *Supports only backup of local database instance*.

#### PostgreSQL backups

For regular and physical backups is used `pg_dump`. You have to ensure that you have a version of `pg_dump` that is
compatible with your database version.

For physical files backups is used `pg_basebackup`. So, you have to ensure that you have a compatible with your
database version of Percona `pg_basebackup`.

#### MongoDB backups

For backups of MongoDB is used `mongodump` tool. You have to ensure that you have a version of `mongodump` that is
compatible with your database version.

#### Redis backups

For backups of Redis is used `redis-cli` tool. You have to ensure that you have a version of `redis-cli` that is
compatible with your Redis version.

## Quickstart

### Install

### On-premise (bare-metal or virtual machine)

nxs-backup is provided for the following processor architectures: amd64 (x86_64), arm (armv7/armv8), arm64 (aarch64).

To install latest version just download and unpack archive for your CPU architecture.

```bash
curl -L https://github.com/nixys/nxs-backup/releases/latest/download/nxs-backup-amd64.tar.gz -o /tmp/nxs-backup.tar.gz
tar xf /tmp/nxs-backup.tar.gz -C /tmp
sudo mv /tmp/nxs-backup /usr/sbin/nxs-backup
sudo chown root:root /usr/sbin/nxs-backup
```

Then check that installation succesfull:

```bash
sudo nxs-backup --version
```

If you need specific version of nxs-backup, or different architecture, you can find it
on [release page](https://github.com/nixys/nxs-backup/releases).

### Docker-compose

- clone the repo

  ```bash
  $ git clone https://github.com/nixys/go-nxs-backup.git
  ```

- go to docker compose directory

  ```bash
  $ cd nxs-backup/.deploy/docker-compose/
  ```

- update provided `nxs-backup.conf` file with your paramethers (see [Settings](#Settings) section for details)

- Launch the nxs-backup with command:

  ```bash
  docker compose up -d --build
  ```

### Kubernetes

Do the following steps:

- Install [nxs-universal-chart](https://github.com/nixys/nxs-universal-chart) (`Helm 3` is required):
  ```
  helm repo add nixys https://registry.nixys.ru/chartrepo/public
  ```
- Launch nxs-backup with command:
  ```
  helm -n $NAMESPACE_SERVICE_NAME install nxs-backup nixys/universal-chart -f values.yaml
  ```
  Where $NAMESPACE_SERVICE_NAME is namespace with your application launched.

- find example `values.yaml` file in [.deploy/kubernetes](.deploy/kubernetes) path

- update it according [Settings](#Settings) section

- Configure nxs-backup (see [Configure](#configure) section for details)

### Settings

Default configuration file path: `/etc/nxs-backup/nxs-backup.conf`. File represented in yaml.

#### General settings

| Name                     | Description                                                                             | Value                                |
|--------------------------|-----------------------------------------------------------------------------------------|--------------------------------------|
| `server_name`            | The name of the server on which the nxs-backup is started                               | `""`                                 |
| `project_name`           | The name of the project, used for notifications (optional)                              | `""`                                 |
| `notifications.webhooks` | Contains list of [webhook notification channel parameters](#webhook-parameters)         | `[]`                                 |
| `notifications.mail`     | Contains [email notification channel parameters](#email-parameters)                     | `{}`                                 |
| `storage_connects`       | Contains list of [remote storages connections](#storage-connection-options)             | `[]`                                 |
| `jobs`                   | Contains list of [backup jobs](#backup-job-settings)                                    | `[]`                                 |
| `include_jobs_configs`   | Contains list of filepaths or glob patterns to [job config files](#backup-job-settings) | `["conf.d/*.conf"]`                  |
| `waiting_timeout`        | Time to waite in minutes for another nxs-backup to be completed (optional)              | `0`                                  |
| `logfile`                | Path to log file                                                                        | `/var/log/nxs-backup/nxs-backup.log` |
| `loglevel`               | Level of messages to be logged. [Supported levels](#notification-levels)                | `info`                               |

##### Webhook parameters

| Name                  | Description                                                                      | Value       |
|-----------------------|----------------------------------------------------------------------------------|-------------|
| `enabled`             | Enables notification channel                                                     | `true`      |
| `webhook_url`         | Contains URL of the webhook service                                              | `""`        |
| `payload_message_key` | Defines request payload key that will contain notification message               | `""`        |
| `extra_payload`       | Contains struct that contains extra request payload keys                         | `{}`        |
| `extra_headers`       | Contains map of strings with request headers                                     | `{}`        |
| `insecure_tls`        | Allows to skip invalid certificates on webhook service side                      | `false`     |
| `message_level`       | Level of messages to be notified about. [Supported levels](#notification-levels) | `"warning"` |

##### Email parameters

| Name            | Description                                                                      | Value       |
|-----------------|----------------------------------------------------------------------------------|-------------|
| `enabled`       | Enables notification channel                                                     | `true`      |
| `mail_from`     | Mailbox on behalf of which mails will be sent                                    | `""`        |
| `smtp_server`   | SMTP host. If not specified email will be sent using `/usr/sbin/sendmail`        | `""`        |
| `smtp_port`     | SMTP port                                                                        | `465`       |
| `smtp_user`     | SMTP user login                                                                  | `""`        |
| `smtp_password` | SMTP user password                                                               | `""`        |
| `recipients`    | List of notifications recipients emails                                          | `[]`        |
| `message_level` | Level of messages to be notified about. [Supported levels](#notification-levels) | `"warning"` |

##### Notification levels

| Name      | Description                                                          |
|-----------|----------------------------------------------------------------------|
| `debug`   | The most detailed information about the backup process               |
| `info`    | General information about the backup process                         |
| `warning` | Information about the backup process that requires special attention |
| `error`   | Only critical information about failures in the backup process       |

##### Storage connection options

nxs-backup storage connect settings block description.

| Name            | Description                                                                           | Value |
|-----------------|---------------------------------------------------------------------------------------|-------|
| `name`          | Unique storage name                                                                   | `""`  |
| `s3_params`     | Connection parameters for [S3 storage type](#s3-connection-params) (optional)         | `{}`  |
| `scp_params`    | Connection parameters for [scp/sftp storage type](#sftp-connection-params) (optional) | `{}`  |
| `ftp_params`    | Connection parameters for [ftp storage type](#ftp-connection-params) (optional)       | `{}`  |
| `nfs_params`    | Connection parameters for [nfs storage type](#nfs-connection-params) (optional)       | `{}`  |
| `smb_params`    | Connection parameters for [smb/cifs storage type](#smb-connection-params) (optional)  | `{}`  |
| `webdav_params` | Connection parameters for [webdav storage type](#webdav-connection-params) (optional) | `{}`  |

##### S3 connection params

| Name                | Description    | Value |
|---------------------|----------------|-------|
| `bucket_name`       | S3 bucket name | `""`  |
| `endpoint`          | S3 endpoint    | `""`  |
| `region`            | S3 region      | `""`  |
| `access_key_id`     | S3 access key  | `""`  |
| `secret_access_key` | S3 secret key  | `""`  |

##### SFTP connection params

| Name                 | Description                                 | Value |
|----------------------|---------------------------------------------|-------|
| `host`               | SSH host                                    | `""`  |
| `port`               | SSH port (optional)                         | `22`  |
| `user`               | SSH user                                    | `""`  |
| `password`           | SSH password                                | `""`  |
| `key_file`           | Path to SSH private key instead of password | `""`  |
| `connection_timeout` | SSH connection timeout seconds (optional)   | `10`  |

##### FTP connection params

| Name                 | Description                                        | Value |
|----------------------|----------------------------------------------------|-------|
| `host`               | FTP host                                           | `""`  |
| `port`               | FTP port (optional)                                | `21`  |
| `user`               | FTP user                                           | `""`  |
| `password`           | FTP password                                       | `""`  |
| `connect_count`      | Count of FTP connections opens to sever (optional) | `5`   |
| `connection_timeout` | FTP connection timeout seconds (optional)          | `10`  |

##### NFS connection params

| Name     | Description                                     | Value |
|----------|-------------------------------------------------|-------|
| `host`   | NFS host                                        | `""`  |
| `target` | Path on NFS server where backups will be stored | `""`  |
| `UID`    | UID of NFS server user (optional)               | `0`   |
| `GID`    | GID of NFS server user (optional)               | `0`   |

##### SMB connection params

| Name                 | Description                               | Value     |
|----------------------|-------------------------------------------|-----------|
| `host`               | SMB host                                  | `""`      |
| `port`               | SMB port (optional)                       | `445`     |
| `user`               | SMB user (optional)                       | `"Guest"` |
| `password`           | SMB password (optional)                   | `""`      |
| `share`              | SMB share name                            | `""`      |
| `domain`             | SMB domain (optional)                     | `""`      |
| `connection_timeout` | SMB connection timeout seconds (optional) | `10`      |

##### WebDav connection params

| Name                 | Description                                  | Value |
|----------------------|----------------------------------------------|-------|
| `url`                | WebDav URL                                   | `""`  |
| `username`           | WebDav user                                  | `""`  |
| `password`           | WebDav password                              | `""`  |
| `oauth_token`        | WebDav OAuth token (optional)                | `""`  |
| `connection_timeout` | WebDav connection timeout seconds (optional) | `10`  |

#### Backup job settings

| Name                 | Description                                                                                                                                                                                                                                                                     | Value   |
|----------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------|
| `job_name`           | Job name. This value is used to run the specific job                                                                                                                                                                                                                            | `""`    |
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

##### Source parameters

| Name                  | Description                                                                                                                                                                      | Value   |
|-----------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------|
| `name`                | Used to differentiate backups in the target directory                                                                                                                            | `""`    |
| `connect`             | Defines a [set of parameters](#database-connection-parameters) for connecting to the database. **Only for [*databases*](#database-types) types**                                 | `{}`    |
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

##### Database connection parameters

| Name                        | Description                                                                          | Value       |
|-----------------------------|--------------------------------------------------------------------------------------|-------------|
| `db_host`                   | DB host                                                                              | `""`        |
| `db_port`                   | DB port                                                                              | `""`        |
| `socket`                    | Path to DB socket                                                                    | `""`        |
| `db_user`                   | DB user                                                                              | `""`        |
| `db_password`               | DB password                                                                          | `""`        |
| `mysql_auth_file`           | Path to MySQL auth file                                                              | `""`        |
| `psql_ssl_mode`             | PostgreSQL SSL mode option                                                           | `"require"` |
| `psql_ssl_root_cert`        | Path to file containing SSL certificate authority (CA) certificate(s) for PostgreSQL | `""`        |
| `psql_ssl_crl`              | Path to file containing SSL server certificate revocation list (CRL) for PostgreSQL  | `""`        |
| `mongo_replica_set_name`    | MongoDB replicaset name                                                              | `""`        |
| `mongo_replica_set_address` | Comma separated list of MongoDB replicaset hosts                                     | `""`        |

You may use either `auth_file` or `db_host` or `socket` options. Options priority follows:
`auth_file` → `db_host` → `socket`

##### Storage options

| Name           | Description                                                                           | Value |
|----------------|---------------------------------------------------------------------------------------|-------|
| `storage_name` | The name of storage, defined in main config. ***local* storage available by default** | `""`  |
| `backup_path`  | Path to directory for storing backups                                                 | `""`  |
| `retention`    | Defines [retention](#storage-retention) for backups on current storage                | `{}`  |

##### Storage retention

| Name     | Description                                                                                                                                                                          | Value |
|----------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------|
| `days`   | Days to store backups                                                                                                                                                                | `7`   |
| `weeks`  | Weeks to store backups                                                                                                                                                               | `5`   |
| `months` | Months to store backups. For *inc_files* backup type determines how many months of incremental copies<br> will be stored relative to the current month. Can take values from 0 to 12 | `12`  |

##### Backup types

###### Database types

| Name                    | Description                |
|-------------------------|----------------------------|
| `mysql`                 | MySQL logical backup       |
| `mysql_xtrabackup`      | MySQL physical backup      |
| `postgresql`            | PostgreSQL logical backup  |
| `postgresql_basebackup` | PostgreSQL physical backup |
| `mongodb`               | MongoDB backup             |
| `redis`                 | Redis backup               |

###### File types

| Name         | Description              |
|--------------|--------------------------|
| `desc_files` | Files discrete backup    |
| `inc_files`  | Files incremental backup |

###### Other types

| Name       | Description            |
|------------|------------------------|
| `external` | External backup script |

#### Environment variables support

Each parameter in configuration files can be defined using environment variables.  
To do this, use the value of parameter
in the following pattern, where `<ENV_NAME>` is the name of the environment variable.

```yaml
param: ENV:<ENV_NAME>
```

**Example:**

```yaml
...
loglevel: ENV:LOG_LEVEL
...
```

### Configuration files example

Here is example of configs for create backups of files and databases for projects located in directory `/var/www` with
exclude of `bitrix` files.

Main config file at `/etc/nxs-backup/nxs-backup.conf`

```yaml
server_name: project-data-server
project_name: My best Project

loglevel: debug

notifications:
  mail:
    enabled: false
  webhooks:
  - webhook_url: https://hooks.slack.com/services/B04AUP00QRX/OkMtk1cq307silFb3rc13W44
    message_level: error
    payload_message_key: "text"
storage_connects:
- name: s3
  s3_params:
    bucket_name: backups_bucket
    access_key_id: my_s3_ak_id
    secret_access_key: ENV:S3_SECRET_KEY
    endpoint: my.s3.endpoint
    region: my-s3-region
jobs: [ ]
include_jobs_configs: [ "conf.d/*.conf" ]
```

Files backup job config at `/etc/nxs-backup/conf.d/files.conf`

```yaml
job_name: files
type: desc_files
tmp_dir: /var/backups/tmp_dump

sources:
- name: "prod_data"
  save_abs_path: yes
  targets:
  - /var/www/*/data/
  - /var/www/*/uploads/
  - /var/www/*/conf/
  excludes:
  - '**/bitrix**'
  gzip: true

storages_options:
- storage_name: local
  backup_path: /var/backups/files
  retention:
    days: 1
    weeks: 0
    months: 0
- storage_name: s3
  backup_path: files
  retention:
    days: 30
    weeks: 0
    months: 12
```

Database backup job config at `/etc/nxs-backup/conf.d/mysql.conf`

```yaml
job_name: mysql
type: mysql
tmp_dir: /var/backups/tmp_dump

sources:
- name: prod
  connect:
    db_host: 'db_host'
    db_port: '3306'
    db_user: 'root'
    db_password: 'some$tr0ngP4ss'
  targets:
  - all
  excludes:
  - mysql
  - information_schema
  - performance_schema
  - sys
  gzip: true
  db_extra_keys: '--opt --add-drop-database --routines --comments --create-options --quote-names --order-by-primary --hex-blob --single-transaction'

storages_options:
- storage_name: local
  backup_path: /var/backups/databases
  retention:
    days: 1
    weeks: 0
    months: 0
- storage_name: s3
  backup_path: databases
  retention:
    days: 30
    weeks: 0
    months: 12
```

### Configure

#### On-premise (bare-metal or virtual machine)

After nxs-backup installation to a server/virtual machine need to generate configuration as the config does not appear
automatically.

You can generate a configuration file by running nxs-backup with the ***generate*** command and the options:

* *-T*[*--backup-type*] (required, backup type)
* *-S*[*--storage-types*] (optional, map of storages),
* *-O*[*--out-path*] (optional, path to the generated conf file).
  This will generate a configuration file for the job and output the details. For example:

```bash
$ sudo nxs-backup generate -T mysql -S minio=s3 aws=s3 share=nfs dumps=scp

nxs-backup: Successfully generated '/etc/nxs-backup/conf.d/mysql.conf' configuration file!
```

nxs-backup configuration files are located in the */etc/nxs-backup/* directory by default. If these files do not exist,
you will be prompted to add them at the first startup.

The basic configuration has only the main configuration file *nxs-backup.conf* and an empty subdirectory *conf.d*, where
files with job descriptions should be stored (one file per job). All configuration files are in YAML format.
For more details, see [Settings](#settings).

##### Testing of Conﬁguration

You can verify that the configuration is correct by running nxs-backup with the ***-t*** option and the optional
parameter *-c*/*--config* (the path to the main conf file). The program will process all configurations and display
error messages and then terminate:

```bash
$ sudo nxs-backup -t
The configuration is correct.
```

##### Starting backups

for starting nxs-backup process please do the following:

```bash
$ sudo nxs-backup start all
```

Please note there are several options for nxs-backup running:

+ `all` - simulates the sequential execution of *external*, *databases*, *files* jobs (default value)
+ `files` - random execution of all jobs of types *desc_files*, *inc_files*
+ `databases` - random execution of all jobs of types *mysql*, *mysql_xtrabackup*, *postgresql*, *
  postgresql_basebackup*, *mongodb*, *redis*
+ `external` - random execution of all jobs of type *external*
+ `<some_job_name>` - the name of one of the jobs to be executed

#### Docker-compose

* create config file e.g. `nxs-backup.conf`
* fill in the file with correct [Settings](#settings)
* put at the same path as `docker-compose.yaml`
* pay your attention that nxs-backup in docker better to start on the same host where the backed up system is located
* run docker compose as it described in [quickstart](#docker-compose)
* there is a working `docker-compose.yml` file example below

```yml
services:
  nxs-backup:
    image: nxs-backup:$IMAGE_VERSION
    container_name: nxs-backup
    volumes:
    - /var/www/site:/var/www/site:ro
    command:
    - nxs-backup
    - -c
    - /nxs-backup.conf
    - start
    - all
    configs:
    - nxs-backup.conf
configs:
  nxs-backup.conf:
    file: ./nxs-backup.conf
```

$IMAGE_VERSION - you can discover on [releases page](https://github.com/nixys/go-nxs-backup/releases)

#### Kubernetes

* fill in a `values.yaml` with correct values from [Settings](#settings) see examples [here](.deploy/kubernetes)
* perform actions described in [quickstart](#kubernetes)
* check that application started correct and running:
    * connect to your kubernetes cluster
    * get cronjobs list:

      ```sh
      $ kubectl -n $NAMESPACE get cronjobs
      ```
      $NAMESPACE - namespace where you installed nxs-backup
    * check that nxs-backup exists in the list of cronjobs

### Database restore

As built-in backups restoring tools are under development. You can discover a few tricks
in [our documentation](docs/backup_restore)

### Useful information

#### External nxs-backup module

In this module, an external script is executed passed to the program via the key "dump_cmd".
By default, at the completion of this command, it is expected that:

* A complete backup file with data will be collected
* The stdout will send data in json format, like:

```json
{
  "full_path": "/abs/path/to/backup.file"
}
```

IMPORTANT:

* make sure that there is no unnecessary information in stdout
* the successfully completed program should finish with exit code 0

If the module used with the `skip_backup_rotate` parameter, the standard output is expected as a result of running
the command. For example, when executing the command "rsync -Pavz /local/source /remote/destination" the result is
expected to be a standard output to stdout.

## Roadmap

Following features are already in backlog for our development team and will be released soon:

* Encrypting of backups
* Restore backups by nxs-backup
* API for remote management and metrics monitoring
* Web interface for management
* Proprietary startup scheduler
* New backup types (Clickhouse, Elastic, lvm, etc.)
* Programmatic implementation of backup creation instead of calling external utilities
* Ability to set limits on resource utilization
* Update help info

## Feedback

For support and feedback please contact me:

* telegram: [@r_andreev](https://t.me/r_andreev)
* e-mail: r.andreev@nixys.ru

## License

nxs-backup is released under the [GNU GPL-3.0 license](LICENSE).
