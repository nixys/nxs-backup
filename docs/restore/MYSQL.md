# MySQL

## Logical backup restore

### Introduction

You can use standard tools for logical dump restoration. More information you can find on the [official documentation page](https://dev.mysql.com/doc/).

#### Step-by-step instruction

1) Log into mysql server:

    ```shell
    $ mysql -u root -p
    ```

    - `-u` - user for loging to datase;
    - `-p` - shell will ask you to prompt for a password before connecting to a database;

2) Create database if it doesn't exist

    ```shell
    mysql> CREATE DATABASE Users;
    ```

3) Exit to OS shell

4) Restore DB dump from OS shell:

    ```shell
    # syntax
    # mysql -u <username> -p <database_name> < /path/to/dump.sql
    # example
    $ mysql -u nxs-user -p Names < /home/user/Documents/names-dump.sql
    ```

## Physical backup restore

### Introduction

Feel free to use [Percona XtraBackup](https://docs.percona.com/percona-xtrabackup/8.0/restore-a-backup.html) to restore of the physical mysql backups. Install and configure it according to [official instruction](https://docs.percona.com/percona-xtrabackup/8.0/installation.html)

#### Example

There are two ways to restore of full backup:

```shell
## 1) backup data stay on your storrage. Use --copy-back option.
$ xtrabackup --copy-back --target-dir=/data/backups/
```

```shell
## 2) backup data will remove. Use --move-back option.
$ xtrabackup --move-back --target-dir=/data/backups/
```

