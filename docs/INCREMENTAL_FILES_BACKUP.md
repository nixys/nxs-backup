# Incremental files

## Introduction

To make backups of your files, you have to ensure that you have **GNU tar** of whatever version is available on your OS.

## Incremental files backup

Works identical like creating a backup using `tar`.

Incremental copies of files are made according to the following scheme:

![Incremental backup scheme](https://image.ibb.co/dtLn2p/nxs_inc_backup_scheme_last_version.jpg)

At the beginning of the year or on the first start of nxs-backup, a full initial backup is created. Then at the
beginning of each month - an incremental monthly copy from a yearly copy is created. Inside each month there are
incremental ten-day copies. Within each ten-day copy incremental day copies are created.

In this case, since now the tar file is in the PAX format, when you deploy the incremental backup, you do not need to
specify the path to inc-files. All the info is stored in the PAX header of the `GNU.dumpdir` directory inside the
archive.
