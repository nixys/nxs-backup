#! /usr/bin/env python3
# -*- coding: utf-8 -*-

import psycopg2

import config
import general_function
import log_and_mail
import periodic_backup


def postgresql_basebackup(job_data):
    is_prams_read, job_name, options = general_function.get_job_parameters(job_data)
    if not is_prams_read:
        return

    full_path_tmp_dir = general_function.get_tmp_dir(options['tmp_dir'], options['backup_type'])

    for i in range(len(options['sources'])):
        try:
            connect = options['sources'][i]['connect']
            gzip = options['sources'][i]['gzip']
            extra_keys = options['sources'][i]['extra_keys']
        except KeyError as e:
            log_and_mail.writelog('ERROR', f"Missing required key:'{e}'!", config.filelog_fd, job_name)
            continue

        db_host = connect.get('db_host')
        db_port = connect.get('db_port')
        db_user = connect.get('db_user')
        db_password = connect.get('db_password')

        if not (db_user or db_host or db_password):
            log_and_mail.writelog('ERROR', "Can't find the authentication data, please fill in the required fields",
                                  config.filelog_fd, job_name)
            continue

        if not db_port:
            db_port = general_function.get_default_port('postgresql')

        try:
            connection = psycopg2.connect(dbname="postgres", user=db_user, password=db_password, host=db_host,
                                          port=db_port)
        except psycopg2.Error as err:
            log_and_mail.writelog(
                'ERROR',
                f"Can't connect to PostgreSQL instances with with following data host='{db_host}', "
                f"port='{db_port}', user='{db_user}', passwd='{db_password}':{err}",
                config.filelog_fd, job_name)
            continue
        else:
            connection.close()

        backup_full_tmp_path = general_function.get_full_path(
            full_path_tmp_dir,
            'postgresq_hot',
            'tar',
            gzip, i)

        periodic_backup.remove_local_file(options['storages'], '', job_name)

        str_auth = f' --dbname=postgresql://{db_user}:{db_password}@{db_host}:{db_port}/ '

        if is_success_pgbasebackup(extra_keys, str_auth, backup_full_tmp_path, gzip, job_name):
            periodic_backup.general_desc_iteration(backup_full_tmp_path, options['storages'], '', job_name,
                                                   options['safety_backup'])

    # After all the manipulations, delete the created temporary directory and
    # data inside the directory with cache davfs, but not the directory itself!
    general_function.del_file_objects(options['backup_type'], full_path_tmp_dir, '/var/cache/davfs2/*')


def is_success_pgbasebackup(extra_keys, str_auth, backup_full_path, gzip, job_name):
    if gzip:
        dump_cmd = f"pg_basebackup {str_auth} {extra_keys} | gzip > {backup_full_path}"
    else:
        dump_cmd = f"pg_basebackup {str_auth} {extra_keys} > {backup_full_path}"

    command = general_function.exec_cmd(dump_cmd)
    stderr = command['stderr']
    code = command['code']

    if stderr:
        log_and_mail.writelog('ERROR', f"Can't create postgresql_basebackup in tmp directory:{stderr}",
                              config.filelog_fd, job_name)
        return False
    elif code != 0:
        log_and_mail.writelog('ERROR', f"Bad result code external process '{dump_cmd}':'{code}'",
                              config.filelog_fd, job_name)
        return False
    else:
        log_and_mail.writelog('INFO', f"Successfully created postgresql_basebackup in tmp directory.",
                              config.filelog_fd, job_name)
        return True
