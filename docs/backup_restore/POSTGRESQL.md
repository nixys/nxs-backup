# PostgreSQL

## Logical backup restore

### Introduction

For restore logical PostgreSQL dump use standart `psql tool`. Detailed information you can find in [official documentation.](https://www.postgresql.org/docs/current/app-psql.html)

#### Examples of use

Basic example:

##### Syntax

```shell
$ psql <database_name> < <full_dmp_file_path>
```

##### Example

```shell
$ psql Names < /home/user/Documents/names.dmp
```

With auth:

```shell
$ psql -U nxs-user -W Names < /tmp/users.dmp
```
* `-U nxs-user` or `--username=nxs-user` :

    Connect to the database as an user, for example `nxs-user`;

* `-W` or `--password` :

    `psql` will ask you to prompt for a password before connecting to a database;

| From `gzip` archive

##### Extract the dump then perform restoration:

    $ gunzip names.dmp.gz
    $ psql Names < names.dmp


## Physical backup restore

### Introduction

For restore physical PostgreSQL dump use standart `pg_restore tool`. Detailed information you can find in [official documentation.](https://www.postgresql.org/docs/current/app-pgrestore.html)

#### Examples of use

##### Restoration of full dump (if you a performed backups of all databases)

```shell
# If database already exists:
$ pg_restore -U postgres -d full_db < full_db.tar

# If database doesn't exists:
$ pg_restore -U postgres -C -d full_db < full_db.tar
```

* `-U` or `--username=username` :

    username to connect to database;

* `-d` or `--dbname=dbname` :

    database defining and restore directly into it;

* `-C` or `--create` :

    firstly create database if it needed;

