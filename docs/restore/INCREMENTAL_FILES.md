# Incremental files

## Introduction

To restore files backups, you have to ensure that you have **GNU tar** of whatever version is available on your OS.

## Description

To restore files backup to the specific date, you have to untar files in next sequence:

`full year backup` -> `monthly backup` -> `decade backup` -> `day backup`

Therefore, the commands to restore a backup to the specific date are the following:

- First, unpack the `full year` copy with the follow command:

    ```bash
    tar xGf /path/to/full/year/backup
    ```

- Then alternately unpack the `monthly`, `decade` and `day` incremental backups, specifying a special key -G:

    ```bash
    tar xGf /path/to/monthly/backup
    tar xGf /path/to/decade/backup
    tar xGf /path/to/day/backup
    ```

## Example

```sh
# Tree of backups files
/var/nxs-backups
├── files
│   ├── desc [...]
│   └── inc
│       └── www
│           ├── project0
│           │   └── 2023
│           │       ├── inc_meta_info [...]
│           │       ├── month_01 [...]
│           │       ├── month_02 [...]
│           │       ├── month_03 [...]
│           │       ├── month_04 [...]
│           │       ├── month_05 [...]
│           │       ├── month_06 [...]
│           │       ├── month_07 [...]
│           │       │   ├── day_01 [...]
│           │       │   ├── day_11 [...]
│           │       │   ├── day_21
│           │       │   │   ├── project0_2023-07-21_01-45.tar.gz
│           │       │   │   ├── project0_2023-07-22_01-43.tar.gz
│           │       │   │   ├── project0_2023-07-23_01-44.tar.gz
│           │       │   │   ├── project0_2023-07-24_01-47.tar.gz
│           │       │   │   ├── project0_2023-07-25_01-44.tar.gz
│           │       │   │   └── project0_2023-07-26_01-48.tar.gz
│           │       │   └── montly
│           │       │       └── project0_2023-07-01_01-45.tar.gz
│           │       └── year
│           │           └── project0_2023-01-01_01-44.tar.gz
│           └── project1 [...]
└── databases [...]

```

```sh
# Restore files to July 26 of 2023
tar xGf /var/nxs-backups/files/inc/www/project0/2023/year/project0_2023-01-01_01-44.tar.gz
tar xGf /var/nxs-backups/files/inc/www/project0/2023/month_07/montly/project0_2023-07-01_01-45.tar.gz
tar xGf /var/nxs-backups/files/inc/www/project0/2023/month_07/day_21/project0_2023-07-21_01-45.tar.gz
tar xGf /var/nxs-backups/files/inc/www/project0/2023/month_07/day_21project0_2023-07-26_01-48.tar.gz
```