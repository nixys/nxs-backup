# On-premise configuration files example

Here is example of configs for create backups of files and databases for projects located on-premise in
directory `/var/www` with exclude of `bitrix` files and upload backups to remote s3 and ssh storages.

Main config file located at [/etc/nxs-backup/nxs-backup.conf](nxs-backup.conf)

Files discrete backup job config at [/etc/nxs-backup/conf.d/files_desc.conf](conf.d/files_desc.conf)

Files incremental backup job config at [/etc/nxs-backup/conf.d/files_inc.conf](conf.d/files_inc.conf)

Mysql database backup job config at [/etc/nxs-backup/conf.d/mysql.conf](conf.d/mysql.conf)

PSQL database backup job config at [/etc/nxs-backup/conf.d/psql.conf](conf.d/psql.conf)
