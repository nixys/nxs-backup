#! /usr/bin/env python3
# -*- coding: utf-8 -*-

import os.path
import re
import glob
from collections import deque

import log_and_mail
import config
import general_function
import periodic_backup


def mysql_xtrabackup(job_data):
    try:
        job_name = job_data['job']
        backup_type = job_data['type']
        tmp_dir = job_data['tmp_dir']
        sources = job_data['sources']
        storages = job_data['storages']
    except KeyError as e:
        log_and_mail.writelog('ERROR', f"Missing required key:'{e}'!", config.filelog_fd, job_name)
        return 1

    full_path_tmp_dir = general_function.get_tmp_dir(tmp_dir, backup_type)

    for i in range(len(sources)):
        try:
            connect = sources[i]['connect']
            gzip =  sources[i]['gzip']
            extra_keys = sources[i]['extra_keys']
        except KeyError as e:
            log_and_mail.writelog('ERROR', f"Missing required key:'{e}'!", config.filelog_fd, job_name)
            continue

        db_user = connect.get('db_user')
        db_password = connect.get('db_password')
        path_to_conf = connect.get('path_to_conf')

        if  not (path_to_conf and db_user and db_password):
            log_and_mail.writelog('ERROR', "Can't find the authentication data, please fill the required fields",
                                  config.filelog_fd, job_name) 
            continue

        if not os.path.isfile(path_to_conf):
            log_and_mail.writelog('ERROR', f"Configuration file '{path_to_conf}' not found!",
                                  config.filelog_fd, job_name) 
            continue

        str_auth = f'--defaults-file={path_to_conf} --user={db_user} --password={db_password}'

        backup_full_tmp_path = general_function.get_full_path(
                                                            full_path_tmp_dir,
                                                            'xtrabackup', 
                                                            'tar',
                                                            gzip)

        periodic_backup.remove_old_local_file(storages, '', job_name)

        if is_success_mysql_xtrabackup(extra_keys, str_auth, backup_full_tmp_path, gzip, job_name):
            periodic_backup.general_desc_iteration(backup_full_tmp_path, 
                                                    storages, '',
                                                    job_name)

    # After all the manipulations, delete the created temporary directory and
    # data inside the directory with cache davfs, but not the directory itself!
    general_function.del_file_objects(backup_type,
                                      full_path_tmp_dir, '/var/cache/davfs2/*')


def is_success_mysql_xtrabackup(extra_keys, str_auth, backup_full_path, gzip, job_name):

    date_now = general_function.get_time_now('backup')
    tmp_status_file = f'/tmp/xtrabackup_status/{date_now}.log'

    dom = int(general_function.get_time_now('dom'))
    if dom == 1:
        dir_for_status_file = os.path.dirname(tmp_status_file)
        if os.path.isdir(dir_for_status_file):
            listing = glob.glob(dir_for_status_file)
            periodic_backup.delete_oldest_files(listing, 31, job_name)

    general_function.create_files(job_name, tmp_status_file)

    if gzip:
        dump_cmd = f"innobackupex {str_auth} {extra_keys} 2>{tmp_status_file} | gzip > {backup_full_path}"
    else:
        dump_cmd = f"innobackupex {str_auth} {extra_keys} > {backup_full_path} 2>{tmp_status_file} "

    command = general_function.exec_cmd(dump_cmd)
    code = command['code']

    if not is_success_status_xtrabackup(tmp_status_file, job_name):
        log_and_mail.writelog('ERROR', f"Can't create xtrabackup in tmp directory! More information in status file {tmp_status_file}.",
                              config.filelog_fd, job_name)
        return False
    elif code != 0:
        log_and_mail.writelog('ERROR', f"Bad result code external process '{dump_cmd}':'{code}'",
                              config.filelog_fd, job_name)
        return False
    else:
        log_and_mail.writelog('INFO', "Successfully created xtrabackup in tmp directory.",
                              config.filelog_fd, job_name)
        return True


def is_success_status_xtrabackup(status_file, job_name):

    try:
        with open(status_file) as f:
            status = list(deque(f, 1))[0]
    except Exception as e:
        log_and_mail.writelog('ERROR', f"Can't read status file '{status_file}':{e}",
                              config.filelog_fd, job_name)
        return False

    else:
        if re.match("^.*completed OK!\n$", status, re.I):
            return True
        else:
            return False
