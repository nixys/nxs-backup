# PostgreSQL

## Logical backup restore

### Introduction

For restore logical PostgreSQL dump use standard `psql tool`. Detailed information you can find in [official documentation.](https://www.postgresql.org/docs/current/app-psql.html)

#### Examples

```shell
# Basic example
$ psql Names < /home/user/Documents/names.dmp
```

```shell
# With auth
$ psql -U nxs-user -W Names < /tmp/users.dmp
```
- `-U nxs-user` or `--username=nxs-user` :

    Connect to the database as an user, for example `nxs-user`;

- `-W` or `--password` :

    `psql` will ask you to prompt for a password before connecting to a database;

```shell
# From `gzip` archive
# 1) extract the dump
$ gunzip names.dmp.gz
# 2) perform restoration
$ psql Names < names.dmp
```

## Physical backup restore

### Introduction

For restore physical PostgreSQL dump use standart `pg_restore tool`. Detailed information you can find in [official documentation.](https://www.postgresql.org/docs/current/app-pgrestore.html)

#### Examples

```shell
# Restoration of full dump (if you a performed backups of all databases)
# If database already exists:
$ pg_restore -U postgres -d full_db < full_db.tar

# If database doesn't exists:
$ pg_restore -U postgres -C -d full_db < full_db.tar
```

- `-U` or `--username=username` :

    username to connect to database;

- `-d` or `--dbname=dbname` :

    database defining and restore directly into it;

- `-C` or `--create` :

    firstly create database if it needed;

