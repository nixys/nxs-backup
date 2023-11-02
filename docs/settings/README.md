# Settings

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
