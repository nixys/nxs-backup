#! /usr/bin/env python3
# -*- coding: utf-8 -*-

import re

import MySQLdb

import config
import general_function
import log_and_mail
import periodic_backup


def is_real_mysql_err(str_err):
    if str_err:
        if re.search('Using a password on the command line interface can be insecure', str_err):
            # Not error for 5.6, 5.7 MySQL
            return False
        else:
            return True
    else:
        return False


def get_connection(db_host, db_port, db_user, db_password, auth_file, socket, job_name):
    if auth_file:
        try:
            connection = MySQLdb.connect(read_default_file=auth_file)
        except MySQLdb.Error as err:
            log_and_mail.writelog('ERROR', f"Can't connect to MySQL instances with '{auth_file}' auth file:{err}",
                                  config.filelog_fd, job_name)
            return 1
        str_auth = f' --defaults-extra-file={auth_file} '
    else:
        if db_host:
            try:
                connection = MySQLdb.connect(host=db_host, port=int(db_port), user=db_user, passwd=db_password)
            except MySQLdb.Error as err:
                log_and_mail.writelog('ERROR',
                                      f"Can't connect to MySQL instances with following data host='{db_host}', port='{db_port}', user='{db_user}', passwd='{db_password}':{err}",
                                      config.filelog_fd, job_name)
                return 1
            str_auth = f' --host={db_host} --port={db_port} --user={db_user} --password={db_password} '
        else:
            try:
                connection = MySQLdb.connect(unix_socket=socket, user=db_user, passwd=db_password)
            except MySQLdb.Error as err:
                log_and_mail.writelog('ERROR',
                                      f"Can't connect to MySQL instances with following data: socket='{socket}', user='{db_user}', passwd='{db_password}':{err}",
                                      config.filelog_fd, job_name)
                return 1
            str_auth = f' --socket={socket} --user={db_user} --password={db_password} '

    return (connection, str_auth)


def mysql_backup(job_data):
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
        exclude_list = sources[i].get('excludes', [])
        try:
            connect = sources[i]['connect']
            target_list = sources[i]['target']
            gzip = sources[i]['gzip']
            is_slave = sources[i]['is_slave']
            extra_keys = sources[i]['extra_keys']
        except KeyError as e:
            log_and_mail.writelog('ERROR', f"Missing required key:'{e}'!", config.filelog_fd, job_name)
            continue

        db_host = connect.get('db_host')
        db_port = connect.get('db_port')
        socket = connect.get('socket')
        db_user = connect.get('db_user')
        db_password = connect.get('db_password')
        auth_file = connect.get('auth_file')

        if not (auth_file or ((db_host or socket) and db_user and db_password)):
            log_and_mail.writelog('ERROR', "Can't find the authentication data, please fill in the required fields",
                                  config.filelog_fd, job_name)
            continue

        if not db_port:
            db_port = general_function.get_default_port('mysql')

        is_all_flag = False

        if 'all' in target_list:
            is_all_flag = True

        try:
            (connection_1, str_auth) = get_connection(db_host, db_port, db_user, db_password, auth_file, socket,
                                                      job_name)
        except:
            continue

        cur_1 = connection_1.cursor()

        if is_all_flag:
            cur_1.execute("SHOW DATABASES")
            target_list = [i[0] for i in cur_1.fetchall()]

        if is_slave:
            try:
                cur_1.execute("STOP SLAVE")
            except MySQLdb.Error as err:
                log_and_mail.writelog('ERROR', f"Can't stop slave: {err}",
                                      config.filelog_fd, job_name)

        connection_1.close()

        for db in target_list:
            if not db in exclude_list:
                backup_full_tmp_path = general_function.get_full_path(
                    full_path_tmp_dir,
                    db,
                    'sql',
                    gzip)

                periodic_backup.remove_old_local_file(storages, db, job_name)

                if is_success_mysqldump(db, extra_keys, str_auth, backup_full_tmp_path, gzip, job_name):
                    periodic_backup.general_desc_iteration(backup_full_tmp_path,
                                                           storages, db,
                                                           job_name)

        if is_slave:
            try:
                (connection_2, str_auth) = get_connection(db_host, db_port, db_user, db_password, auth_file, socket,
                                                          job_name)
                cur_2 = connection_2.cursor()
                cur_2.execute("START SLAVE")
            except MySQLdb.Error as err:
                log_and_mail.writelog('ERROR', f"Can't start slave: {err} ",
                                      config.filelog_fd, job_name)
            finally:
                connection_2.close()

    # After all the manipulations, delete the created temporary directory and
    # data inside the directory with cache davfs, but not the directory itself!
    general_function.del_file_objects(backup_type,
                                      full_path_tmp_dir, '/var/cache/davfs2/*')


def is_success_mysqldump(db, extra_keys, str_auth, backup_full_path, gzip, job_name):
    if gzip:
        dump_cmd = f"mysqldump {str_auth} {extra_keys} {db} | gzip > {backup_full_path}"
    else:
        dump_cmd = f"mysqldump {str_auth} {extra_keys} {db} > {backup_full_path}"

    command = general_function.exec_cmd(dump_cmd)
    stderr = command['stderr']
    code = command['code']

    if stderr and is_real_mysql_err(stderr):
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
