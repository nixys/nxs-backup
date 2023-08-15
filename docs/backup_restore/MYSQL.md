# MySQL

## Introduction

You can use standart tools for dump restoration. More information you can find on the [official documentation page](https://dev.mysql.com/doc/).

### Step by step instruction

1) Log into mysql server:

    ```shell
    $ mysql -u root -p
    ```

    * `-u` - user for loging to datase;
    * `-p` - shell will ask you to prompt for a password before connecting to a database;

2) Create database if it doesn't exists

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
