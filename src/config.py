#! /usr/bin/env python3
# -*- coding: utf-8 -*-

import sys

import general_function

lock_file_fd = ''
path_to_lock_file = '/tmp/nxs-backup.lock'

supported_db_backup_type = [
    'mysql',
    'mysql_xtrabackup',
    'postgresql',
    'postgresql_basebackup',
    'mongodb',
    'redis'
]

supported_file_backup_type = [
    'desc_files',
    'inc_files'
]

supported_external_backup_type = [
    'external'
]

supported_storages = [
    'local',
    'scp',
    'ftp',
    'webdav',
    'smb',
    'nfs',
    's3'
]

backup_extenstion = [
    '*.sql',
    '*.sql.gz',
    '*.tar',
    '*.tar.gz',
    '*.pgdump',
    '*.pgdump.gz',
    '*.mongodump',
    '*.mongodump.gz',
    '*.rdb',
    '*.rdb.gz'
]

supported_general_job = ['external', 'databases', 'files', 'all']

dow_backup = '4'
dom_backup = '05'

default_port_dict = {
    'mysql': '3306',
    'postgresql': '5432',
    'redis': '6379',
    'mongodb': '27017'
}

filelog_fd = ''
error_log = ''
debug_log = ''

supported_backup_type = supported_db_backup_type + supported_file_backup_type + \
                        supported_external_backup_type

client_mail = []
all_jobs_name = []
general_str = ''
regular_str = ''
regular_str_for_backup_type = ''
general_str_for_backup_type = ''
general_str_for_backup_type_db = ''
general_str_for_backup_type_files = ''
general_str_for_backup_type_external = ''
regular_str_for_storage = ''
general_str_for_storage = ''
log_file = ''
admin_mail = ''
level_message = ''
mail_from = ''
server_name = ''
block_io_write = ''
block_io_read = ''
block_io_weight = ''
general_path_to_all_tmp_dir = ''
cpu_shares = ''
smtp_server = ''
smtp_port = ''
smtp_ssl = ''
smtp_user = ''
smtp_password = ''
smtp_timeout = ''
smtp_tls = ''
loop_timeout = 0
loop_interval = 0


def get_conf_value(parsed_str):
    """ The function assigns a value to the key global program variables.
    At the input, the function takes a parsed line of the configuration file.
    """

    global all_jobs_name
    global general_str
    global regular_str
    global regular_str_for_backup_type
    global general_str_for_backup_type
    global general_str_for_backup_type_db
    global general_str_for_backup_type_files
    global general_str_for_backup_type_external
    global regular_str_for_storage
    global general_str_for_storage
    global log_file
    global admin_mail
    global client_mail
    global level_message
    global mail_from
    global server_name
    global block_io_write
    global block_io_read
    global block_io_weight
    global general_path_to_all_tmp_dir
    global cpu_shares
    global supported_general_job
    global smtp_server
    global smtp_port
    global smtp_ssl
    global smtp_user
    global smtp_password
    global smtp_timeout
    global smtp_tls
    global loop_timeout
    global loop_interval

    general_str_for_backup_type_db = ', '.join(supported_db_backup_type)
    general_str_for_backup_type_files = ', '.join(supported_file_backup_type)
    general_str_for_backup_type_external = ', '.join(supported_external_backup_type)

    regular_str_for_backup_type = ''.join(['^' + item + '$|' for item in supported_backup_type])[0:-1]
    general_str_for_backup_type = ', '.join(supported_backup_type)

    regular_str_for_storage = ''.join(['^' + item + '$|' for item in supported_storages])[0:-1]
    general_str_for_storage = ', '.join(supported_storages)

    count_of_jobs = len(parsed_str['jobs'])
    for i in range(count_of_jobs):
        for j in range(count_of_jobs):
            a = parsed_str['jobs'][i]['job']
            b = parsed_str['jobs'][j]['job']
            if i != j and a == b:
                general_function.print_info(f"Duplicate job name '{a}'. You must use a unique name for the job's name.")
                sys.exit(1)

    db_job_dict = {}
    file_job_dict = {}
    external_job_dict = {}

    for i in range(count_of_jobs):
        job_data = parsed_str['jobs'][i]
        backup_type = job_data['type']
        job_name = job_data['job']
        if backup_type in supported_db_backup_type:
            db_job_dict[job_name] = job_data
        elif backup_type in supported_file_backup_type:
            file_job_dict[job_name] = job_data
        elif backup_type in supported_external_backup_type:
            external_job_dict[job_name] = job_data
        else:
            general_function.print_info(
                f"Backup type '{backup_type}' in job '{job_name}' does not supported, so this job was ignored! "
                f"Only one of this type backup is allowed:{supported_backup_type}!")

    all_jobs_name = (list(db_job_dict.keys()) + list(file_job_dict.keys()) +
                     list(external_job_dict.keys()) + supported_general_job)

    general_str = ', '.join(all_jobs_name)
    regular_str = ''.join(['^' + item + '$|' for item in all_jobs_name])[0:-1]

    log_file = parsed_str['main'].get('log_file', None)
    if not log_file:
        log_file = '/var/log/nxs-backup/nxs-backup.log'

    admin_mail = parsed_str['main']['admin_mail']
    if not admin_mail:
        general_function.print_info("Field 'admin_mail' in 'main' section can't be empty!")
        sys.exit(1)

    client_mail_array = parsed_str['main'].get('client_mail', [])
    for i in client_mail_array:
        client_mail.append(i)

    level_message = parsed_str['main']['level_message']
    mail_from = parsed_str['main']['mail_from']
    server_name = parsed_str['main']['server_name']

    block_io_write = parsed_str['main'].get('block_io_write', None)
    block_io_read = parsed_str['main'].get('block_io_read', None)
    block_io_weight = parsed_str['main'].get('blkio_weight', None)
    general_path_to_all_tmp_dir = parsed_str['main'].get('general_path_to_all_tmp_dir', None)
    cpu_shares = parsed_str['main'].get('cpu_shares', None)

    smtp_port = parsed_str['main'].get('smtp_port', None)
    smtp_ssl = parsed_str['main'].get('smtp_ssl', None)
    smtp_server = parsed_str['main'].get('smtp_server', None)
    smtp_user = parsed_str['main'].get('smtp_user', None)
    smtp_password = parsed_str['main'].get('smtp_password', None)
    smtp_timeout = parsed_str['main'].get('smtp_timeout', None)
    smtp_tls = parsed_str['main'].get('smtp_tls', None)

    loop_timeout = parsed_str['main'].get('loop_timeout', None)
    loop_interval = parsed_str['main'].get('loop_interval', 30)  # 30 seconds - default loop interval

    return db_job_dict, file_job_dict, external_job_dict
