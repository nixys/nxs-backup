# Nxs-backup

Nxs-backup is an open source backup software for most popular GNU/Linux distributions. Features of Nxs-backup include amongst others:
* Support of the most popular storages: local, ssh, ftp, cifs(smb), nfs, webdav, s3
* Database backups, such as MySQL(logical/physical), PostgreSQL(logical/physical), MongoDB, Redis
* Possibility to specify extra options for collecting database dumps to fine-tune backup process and minimize load on the server
* Incremental files backups
* Easy to read and maintain configuration files with clear transparent structure
* Built-in generator of the configuration files to expedite inital setup
* Support of user-defined custom scripts to extend functionality
* Possibility to restore backups with standard tools (no extra software including Nxs-backup is required)
* Email notifications about status and errors during backup process

The source code of Nxs-backup is available at https://github.com/nixys/nxs-backup/ under the GPL v3 license. Additionally Nxs-backup offers binary package repositories for the major Linux distributions (Debian, CentOS).

## Getting started

### Understanding Jobs, Type, Sources and Storages
 In order to make nxs-backup as ﬂexible as possible, the directions given to nxs-backup are speciﬁed in several pieces. The main instruction is the job resource, which deﬁnes a job. A backup job generally consists of a Type, a Sources and Storages. 
 The Type deﬁnes what type of backup shall run (e.g. MySQL "physical" backups), the Sources defines the target and exceptions (for each job at least one target must be specified), the Storages define storages where to store backups and at what quantity (for each job at least one storage must be specified). Work with remote storage is performed by local mounting of the FS with special tools.

### Setting Up Nxs-backup Conﬁguration Files
 Nxs-backup conﬁguration ﬁles are usually located in the */etc/nxs-backup/* directory. The default configuration has only one configuration file *nxs-backup.conf* and the *conf.d* subdirectory that stores files with descriptions of jobs (one file per job). Config files are in YAML format. For details, see Settings.
 
### Generate your Configurations Files for job
 You can generate your conﬁguration ﬁle for a job by running the script with the command ***generate*** and *-S*/*--storages* (list of storages), *-T*/*--type* (type of backup), *-P*/*--path* (path to generated file) options. The script will generate conﬁguration ﬁle for the job and print result:

 ```bash
# nxs-backup generate -S local scp -T mysql -P /etc/nxs-backup/conf.d/mysql.conf
nxs-backup: Successfully generated '/etc/nxs-backup/conf.d/mysql.conf' configuration file!
```

### Testing your Conﬁguration Files
 You can test whether your conﬁguration ﬁle is syntactically correct by running the script with the ***-t*** option and optional *-c*/*--config* (path to main conf file). The script will process the conﬁguration ﬁle and print any error messages and then terminate:

```bash
# nxs-backup -t
nxs-backup: The configuration file '/etc/nxs-backup/nxs-backup.conf' syntax is ok!
```

### Start your jobs
 You cat start your jobs by running the script with the command ***start*** and optional *-c*/*--config* (path to main conf file). The script will execute the job passed by the argument. It should be noted that there are several reserved job names:
 + `all` - simulates the sequential execution of *files*, *databases*, *external* job (default value)
 + `files` - random execution of all jobs with the types *desc_files*, *inc_files*
 + `databases` - random execution of all jobs with the types *mysql*, *mysql_xtrabackup*, *postgresql*, *postgresql_basebackup*, *mongodb*, *redis*
 + `external` - random execution of all jobs with the type *external*
 
```bash
# nxs-backup start all
```

## Settings

### `main`

Nxs-backup main settings block description.

* `server_name`: the name of the server on which the nxs-backup is started.
* `admin_mail`: admin email on which notifications about errors during backup process will be sent.
* `client_mail`(optional): emails of additional users that shall also receive nxs-backup notifications.
* `level_message`: level of informing users specified in the directive `client_mail`, two levels are allowed:
 * *error* - send only notifications about errors during backup process;
 * *debug* - send full nxs-backup performance log.
* `mail_from`: mailbox on behalf of which letters are sent.
* `smtp_server`(optional): SMTP host. If not specified send email via /usr/sbin/sendmail.
* `smtp_port`(optional): SMTP port.
* `smtp_ssl`(optional): enable/disable SSL.
* `smtp_tls`(optional): enable/disable TLS.
* `smtp_user`(optional): SMTP user login.
* `smtp_password`(optional): SMTP user password.
* `smtp_timeout`(optional): SMTP connection timeout.
* `block_io_read`(optional): limit reading speed from the block device on which backups are collected (MB/s).
* `block_io_write`(optional): limit writing speed to a block device on which backups are collected (MB/s).
* `blkio_weight`(optional): "weight" of the backup process when working with a block device on which backups are collected (weight must be in range from 100 to 1000).
* `general_path_to_all_tmp_dir`(optional): the general part of the path to directories with temporary copies (based on this, the block device is defined, which IO must be limited).
* `cpu_shares`(optional): the percentage of CPU computing resources that will go to the backup process.
* `log_file`(optional): path to log file. If not specified the default value will be used (/var/log/nxs-backup/nxs-backup.log).
* `loop_timeout`(optional): waiting time for another nxs-backup to be completed. By default disabled.
* `loop_interval`(optional): the interval in seconds, how often to check if another copy of nxs-backup has ended. By default 30 seconds.

### `jobs`

Nxs-backup jobs settings block description. Allows you to connect additional configuration files  by specifying the following (you can use glob patterns):

```yaml
jobs: !include [conf.d/*.conf]
```

* `job`: job name. This value is used to run the required job.
* `type`: type of backup. It can take the following values:
  * *mysql*(MySQL logical backups), *mysql_xtrabackup* (MySQL physical backups), *postgresql*(PostgreSQL logical backups), *postgresql_basebackup*(PostgreSQL physical backups), *mongodb*, *redis*
  * *desc_files*, *inc_files*
  * *external*
* `tmp_dir`: a local path to the temporary directory for backups.
* `dump_cmd`(**only for *external* type**): full command to run an external script.
* `safety_backup`(logical)(optional): Delete outdated backups after creating a new one. By default, "false". **IMPORTANT** Using of this option requires more disk space. Make sure there is enough free space on the end device.
* `deferred_copying_level` (optional)(int): Determines the level of deferred copying. The minimum value is 0 (by default), copying occurs immediately after the temporary backup is created. The maximum value is 3, copying occurs after creation of all temporary backups defined in the task. **IMPORTANT** Using of this option requires more disk space for more level. Make sure there is enough free space on the device where temporary backups stores.
* `inc_months_to_store` (optional)(int, **only for *inc_files* type**): Determines how many months of incremental copies will be stored relative to the current month. Can take values from 0 to 12, the default is 12.
* `sources` (objects array): Specify one target or array of targets for backup:
  * `connect` (object, **Only for *databases* types**). It is necessary to fill a minimum set of keys to allow database connection:
    * `db_host`: DB host.
    * `db_port`: DB port.
    * `socket`:  DB socket.
    * `db_user`: DB user.
    * `db_password`: DB password.
    * `auth_file`: DB auth file. You may use either `auth_file` or `db_host` or `socket` options. Options priority follows: `auth_file` → `db_host` → `socket`.
    * `path_to_conf`(**only for *mysql_xtrabackup* type**): path to the main mysql configuration file with *client* section.
  * `special_keys`(**Only for *databases* types**): special parameters for the collection of database backups 
  * `target`: list of databases or directory/files to be backed up. For *databases types* you can use the keyword **all** (all db). For *files types* you can use glob patterns.
  * `target_dbs`(**Only for *mongodb* type**): list of mongodb databases to be backed up.  
  * `target_collections`(**Only for *mongodb* type**): list of collections of all mongodb databases to be backed up. You can use the keyword **all** (all collections in all db).
  * `excludes`: list of databases or directory/files to be excluded from backup. For *files types* you can use glob patterns.
  * `exclude_dbs`(**Only for *mongodb* type**):
  * `exclude_collections`(**Only for *mongodb* type**): 
  * `gzip`(logicals): compress or not compress the archive 
  * `skip_backup_rotate`(**Only for *external* type**)(optional)(logicals): If creation of a local copy is not required, for example, in case of copying data to a remote server, rotation of local backups may be skipped with this option.
* `storages`(objects array) specify one storage or array of storages to store archive:
 * `storage`: type of storage. It can take the following values:
   * *local*, *scp*, *ftp*, *smb* (via cifs), *nfs*, *webdav*, *s3*
 * `enable`(logicals): enable or disable storage
 * `backup_dir`: directory for storing backups. **IMPORTANT** For the following storages - *scp*, *nfs*, the directory actually acts as a mount resource (used directly in the mount command), so you need to make sure it exists on the remote server or `remote_mount_point` is defined, otherwise there will be an error. For other storages - *local*, *ftp*, *smb*, *webdav*, *s3* this directory is already inside the environment, where we get after mounting the resource, so it can be created by the program itself.
 * `remote_mount_point`(**Only for *scp* and *nfs* storages**)(optional): Remote mounting point directory. This directory will be used as the mount resource, so you need to make sure that it exists on the remote server and is user `user` owned, otherwise an error will occur.  The default is `backup_dir`.
 * `host`: storage host.
 * `port`: storage port.
 * `user`: storage user.
 * `password`: storage password.
 * `extra_keys`(**Only for *nfs* storage**): extra keys for mount command.
 * `bucket_name`(**Only for *s3* storage**): bucket name.
 * `access_key_id`(**Only for *s3* storage**)(optional): S3 compatibility access key.
 * `secret_access_key`(**Only for *s3* storage**)(optional): S3 compatibility secret key.
 * `s3fs_opts`(**Only for *s3* storage**): extra keys for mount s3fs command. For example, for loading on custom s3 compatibility API server you need to add the following options '-o url=https://<custom_endpoint_url> -o use_path_request_style'.
 * `path_to_key`(**Only for *scp* storage**): path to ssh private key.
 * `share`(**Only for *smb* storage**): share.
 * `store`(objects, required for all after exception *inc_files* type backup):
   * `days`: days to store backups.
   * `weeks`: weeks to store backups.
   * `month`: months to store backups.

## Useful information

### Desc files nxs-backup module

Under the hood there is python module `tarfile`.

### Incremental files nxs-backup module

Under the hood there is python module `tarfile`. Incremental copies of files are made according to the following scheme:
![Incremental backup scheme](https://image.ibb.co/dtLn2p/nxs_inc_backup_scheme_last_version.jpg)

At the beginning of the year or on the first start of the script, a full initial backup is created. Then at the beginning of each month - an incremental monthly copy from a yearly copy is created. Inside each month there are incremental ten-day copies. Within each ten-day copy incremental day copies are created.

In this case, since now the tar file is in the PAX format, when you deploy the incremental backup, you do not need to specify the path to inc-files. All the info is stored in the PAX header of the GNU.dumpdir directory inside the archive. Therefore, the commands to restore a backup for a specific date are the following:
* First, unpack a full annual copy with the follow command:
```bash
tar xf PATH_TO_FULL_BACKUP
```
* Then alternately unpack the monthly, ten-day, day incremental backups, specifying a special key -G, for example:
```bash
tar xGf PATH_TO_INCREMENTAL_BACKUP
```

### MySQL(logical) nxs-backup module

Under the hood is the work of the `mysqldump`, so for the correct work of the module you must first install **mysql-client** on the server.

### MySQL(physical) nxs-backup module

Under the hood is the work of the `innobackupex`, so for the correct work of the module you must first install **percona-xtrabackup** on the server. *Supports only backup of local instance*.

### PostgreSQL(logical) nxs-backup module

The work is based on `pg_dump`, so for the correct work of the module you must first install **postgresql-client** on the server.

### PostgreSQL(physical) nxs-backup module

The work is based on `pg_basebackup`, so for the correct work of the module you must first install **postgresql-client** on the server.

### MongoDB nxs-backup module

The work is based on  `mongodump`, so for the correct work of the module you must first install **mongodb-clients** on the server.

### Redis nxs-backup module

The work is based on  `redis-cli with --rdb option`, so for the correct work of the module you must first install **redis-tools** on the server.

### External nxs-backup module

In this module, an external script is executed passed to the program via the key "dump_cmd".  
By default at the completion of this command, it is expected that:
* A complete archive of data will be collected
* The stdout will send data in json format, like:

```json
{
    "full_path": "ABS_PATH_TO_ARCHIVE",
    "basename": "BASENAME_ARCHIVE",
    "extension": "EXTENSION_OF_ARCHIVE",
    "gzip": "true|false"
}
```

In this case, the keys basename, extension, gzip are necessary only for the formation of the final name of the backup. IMPORTANT:
* make sure that there is no unnecessary information in stdout
* *gzip* is a parameter that tells the script whether the file is compressed along the path specified in full_path or not, but does not indicate the need for compression at the nxs-backup
* the successfully completed program must exit with 0

If the module was used with the `skip_backup_rotate` parameter, the standard output is expected as a result of running the command.  
For example, when executing the command "rsync -Pavz /local/source /remote/destination" the result is expected to be a standard output to stdout.  

### SSH storage nxs-backup module

For correct work of the software you must install *openssh-client*, *sshfs*, *sshpass*, *fuse*  packages.

### FTP storage nxs-backup module

For correct work of the software you must install *curlftpfs*, *fuse* packages.

### SMB storage nxs-backup module

For correct work of the software, you must install *cifs-utils*, *fuse* packages.

### NFS storage nxs-backup module

For correct work of the software, you must install *nfs-common*/*nfs-utils*, *fuse* packages.

### WebDAV storage nxs-backup module

For correct work of the software, you must install *davfs2*, *fuse* packages.

### S3 storage nxs-backup module

For correct work of the software, you must install [s3fs](https://github.com/s3fs-fuse/s3fs-fuse)  and *fuse* package.

## Install nxs-backup

### Debian

1.  Add Nixys repository key:

    ```
    apt-key adv --fetch-keys http://packages.nixys.ru/debian/repository.gpg.key
    ```

2.  Add the repository. Currently Debian wheezy, jessie and stretch are available:

    ```
    echo "deb [arch=amd64] http://packages.nixys.ru/debian/ wheezy main" > /etc/apt/sources.list.d/packages.nixys.ru.list
    ```

    ```
    echo "deb [arch=amd64] http://packages.nixys.ru/debian/ jessie main" > /etc/apt/sources.list.d/packages.nixys.ru.list
    ```

    ```
    echo "deb [arch=amd64] http://packages.nixys.ru/debian/ stretch main" > /etc/apt/sources.list.d/packages.nixys.ru.list
    ```

3.  Make an update:

    ```
    apt-get update
    ```

4.  Install nxs-backup:

    ```
    apt-get install nxs-backup
    ```

### CentOS

1.  Add Nixys repository key:

    ```
    rpm --import http://packages.nixys.ru/centos/repository.gpg.key
    ```

2.  Add the repository. Currently CentOS 6 and 7 are available:

    ```
    cat <<EOF > /etc/yum.repos.d/packages.nixys.ru.repo
    [packages.nixys.ru]
    name=Nixys Packages for CentOS \$releasever - \$basearch
    baseurl=http://packages.nixys.ru/centos/\$releasever/\$basearch
    enabled=1
    gpgcheck=1
    gpgkey=http://packages.nixys.ru/centos/repository.gpg.key
    EOF
    ```

3.  Install nxs-backup:

    ```
    yum install nxs-backup
    ```
