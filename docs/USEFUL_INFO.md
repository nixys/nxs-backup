# Useful information

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

* A complete backup file with data will be collected
* The stdout will send data in json format, like:

```json
{
  "full_path": "/abs/path/to/backup.file"
}
```

**IMPORTANT:**

* make sure that there is no unnecessary information in stdout
* the successfully completed program should finish with exit code 0

If the module used with the `skip_backup_rotate` parameter, the standard output is expected as a result of running
the command. For example, when executing the command "rsync -Pavz /local/source /remote/destination" the result is
expected to be a standard output to stdout.
