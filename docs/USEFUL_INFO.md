# Useful information

## Generate Configuration Files

After nxs-backup installation to a server/virtual machine need to generate configuration as the config does not appear
automatically.

You can generate a configuration file by running nxs-backup with the ***generate*** command and the options:

* *-T* [*--backup-type*] (required, backup type)
* *-S* [*--storage-types*] (optional, map of storages),
* *-O* [*--out-path*] (optional, path to the generated conf file).

This will generate a configuration file for the job and output the details. For example:

```bash
$ sudo nxs-backup generate -T mysql -S minio=s3 aws=s3 share=nfs dumps=scp

nxs-backup: Successfully generated '/etc/nxs-backup/conf.d/mysql.conf' configuration file!
```

nxs-backup configuration files are located in the */etc/nxs-backup/* directory by default. If these files do not exist,
you will be prompted to add them at the first startup.

The basic configuration has only the main configuration file *nxs-backup.conf* and an empty subdirectory *conf.d*, where
files with job descriptions should be stored (one file per job). All configuration files are in YAML format.
For more details, see [Settings](/docs/settings/README.md).

You can find the example of on-premise config files [here](example/on-premise/README.md).

### Testing of Conﬁguration

You can verify that the configuration is correct by running nxs-backup with the ***-t*** option and the optional
parameter *-c*/*--config* (the path to the main conf file). The program will process all configurations and display
error messages and then terminate:

```sh
$ sudo nxs-backup -t
The configuration is correct.
```

## Run specific jobs

There are several options for nxs-backup running:

- `all` - simulates the sequential execution of *external*, *databases*, *files* jobs (default value)
- `files` - random execution of all jobs of types *desc_files*, *inc_files*
- `databases` - random execution of all jobs of types *mysql*, *mysql_xtrabackup*, *postgresql*, *
  postgresql_basebackup*, *mongodb*, *redis*
- `external` - random execution of all jobs of type *external*
- `<some_job_name>` - the name of one of the jobs to be executed


## External nxs-backup module

In this module, an external script is executed passed to the program via the key "dump_cmd".
By default, at the completion of this command, it is expected that:

- A complete backup file with data will be collected
- The stdout will send data in json format, like:

```json
{
  "full_path": "/abs/path/to/backup.file"
}
```

**IMPORTANT:**

- Make sure that there is no unnecessary information in stdout
- The successfully completed program should finish with exit code 0

If the module used with the `skip_backup_rotate` parameter, the standard output is expected as a result of running
the command. For example, when executing the command "rsync -Pavz /local/source /remote/destination" the result is
expected to be a standard output to stdout.
