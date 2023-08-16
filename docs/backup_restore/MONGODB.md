# MongoDB

## Introduction

Feel free to use standart [mongorestore tool](https://www.mongodb.com/docs/database-tools/mongorestore/#installation) which is part of strandart MongoDB toolkit. There are the most popular variants of restorations MongoDB backup.
More about `mongorestore` options you can find in [official documentation](https://www.mongodb.com/docs/database-tools/mongorestore/#options)

### Examples:

```sh
# archived backup restore
$ mongorestore --gzip --archive=/path/contains/backup_archive.gz
```
* `--gzip`

    Restores from compressed files or data stream

* `--archive`

    Specified an archive file for restoring

```sh
# basic backup restore
$ mongorestore --drop --dir /tmp/backup
```

* `--drop`

    Drop collection before restore if it exists

* `--dir`

    Points to a directory with a backup

```sh
# Restore only provided namespaces
$ mongorestore --drop --dir /home/user/backup --nsInclude 'nxs.collection'
```
* `--nsInclude`

    Comma separated list of namespaces to restore

```sh
# Restore database without list of namespaces
$ mongorestore --drop --dir /home/user/backup --nsExclude 'nxs.collection'
```

* `--nsExclude`

    Comma separated list of namespaces to exclude from the restore