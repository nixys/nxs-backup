#! /usr/bin/env python3
# -*- coding: utf-8 -*-

import psycopg2

import config
import general_function
import log_and_mail
import periodic_backup


def postgresql_backup(job_data):
    job_name = 'undefined'
    try:
        job_name = job_data['job']
        backup_type = job_data['type']
        tmp_dir = job_data['tmp_dir']
        sources = job_data['sources']
        storages = job_data['storages']
    except KeyError as e:
        log_and_mail.writelog('ERROR', f"Missing required key:'{e}'!", config.filelog_fd, job_name)
        return 1

    safety_backup = job_data.get('safety_backup', False)
    full_path_tmp_dir = general_function.get_tmp_dir(tmp_dir, backup_type)

    for i in range(len(sources)):
        exclude_list = sources[i].get('excludes', [])
        try:
            connect = sources[i]['connect']
            target_list = sources[i]['target']
            gzip = sources[i]['gzip']
            extra_keys = sources[i]['extra_keys']
        except KeyError as e:
            log_and_mail.writelog('ERROR', f"Missing required key:'{e}'!", config.filelog_fd, job_name)
            continue

        db_host = connect.get('db_host')
        db_port = connect.get('db_port')
        db_user = connect.get('db_user')
        db_password = connect.get('db_password')

        if not (db_user and db_host and db_password):
            log_and_mail.writelog('ERROR', "Can't find the authentication data, please fill in the required fields",
                                  config.filelog_fd, job_name)
            continue

        if not db_port:
            db_port = general_function.get_default_port('postgresql')

        is_all_flag = False

        if 'all' in target_list:
            is_all_flag = True

        if is_all_flag:
            try:
                connection = psycopg2.connect(dbname="postgres", user=db_user, password=db_password, host=db_host,
                                              port=db_port)
            except psycopg2.Error as err:
                log_and_mail.writelog('ERROR',
                                      f"Can't connect to PostgreSQL instances with with following data host='{db_host}', port='{db_port}', user='{db_user}', passwd='{db_password}':{err}",
                                      config.filelog_fd, job_name)
                continue

            cur = connection.cursor()
            cur.execute("select datname from pg_database;")
            target_list = [i[0] for i in cur.fetchall()]
            connection.close()

        for db in target_list:
            if db not in exclude_list:
                backup_full_tmp_path = general_function.get_full_path(
                    full_path_tmp_dir,
                    db,
                    'pgdump',
                    gzip)

                if not safety_backup:
                    periodic_backup.remove_old_local_file(storages, db, job_name)

                str_auth = f' --dbname=postgresql://{db_user}:{db_password}@{db_host}:{db_port}/{db} '

                if is_success_pgdump(db, extra_keys, str_auth, backup_full_tmp_path, gzip, job_name):
                    periodic_backup.general_desc_iteration(backup_full_tmp_path,
                                                           storages, db,
                                                           job_name)
                    if safety_backup:
                        periodic_backup.remove_old_local_file(storages, db, job_name)

    # After all the manipulations, delete the created temporary directory and
    # data inside the directory with cache davfs, but not the directory itself!
    general_function.del_file_objects(backup_type,
                                      full_path_tmp_dir, '/var/cache/davfs2/*')


def is_success_pgdump(db, extra_keys, str_auth, backup_full_path, gzip, job_name):
    if gzip:
        dump_cmd = f"pg_dump {extra_keys} {str_auth} | gzip > {backup_full_path}"
    else:
        dump_cmd = f"pg_dump {extra_keys} {str_auth} > {backup_full_path}"

    command = general_function.exec_cmd(dump_cmd)
    stderr = command['stderr']
    code = command['code']

    if stderr:
        log_and_mail.writelog('ERROR', f"Can't create '{db}' database dump in tmp directory:{stderr}",
                              config.filelog_fd, job_name)
        return False
    elif code != 0:
        log_and_mail.writelog('ERROR', f"Bad result code external process '{dump_cmd}':'{code}'",
                              config.filelog_fd, job_name)
        return False
    else:
        log_and_mail.writelog('INFO', f"Successfully created '{db}' database dump in tmp directory.",
                              config.filelog_fd, job_name)
        return True
