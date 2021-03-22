#! /usr/bin/env python3
# -*- coding: utf-8 -*-

import redis

import config
import general_files_func
import general_function
import log_and_mail
import periodic_backup


def redis_backup(job_data):
    is_prams_read, job_name, options = general_function.get_job_parameters(job_data)
    if not is_prams_read:
        return

    full_path_tmp_dir = general_function.get_tmp_dir(options['tmp_dir'], options['backup_type'])

    for i in range(len(options['sources'])):
        try:
            connect = options['sources'][i]['connect']
            gzip = options['sources'][i]['gzip']
        except KeyError as e:
            log_and_mail.writelog('ERROR', f"Missing required key:'{e}'!", config.filelog_fd, job_name)
            continue

        db_host = connect.get('db_host')
        db_port = connect.get('db_port')
        db_password = connect.get('db_password')
        socket = connect.get('socket')

        if not (db_host or socket):
            log_and_mail.writelog('ERROR', "Can't find the authentication data, please fill in the required fields",
                                  config.filelog_fd, job_name)
            continue

        if not db_port:
            db_port = general_function.get_default_port('redis')

        try:
            if db_host:
                if db_password:
                    redis.StrictRedis(host=db_host, port=db_port, password=db_password)
                    str_auth = f" -h {db_host} -p {db_port} -a '{db_password}' "
                else:
                    redis.StrictRedis(host=db_host, port=db_port)
                    str_auth = f" -h {db_host} -p {db_port} "
            else:
                if db_password:
                    redis.StrictRedis(unix_socket_path=socket, password=db_password)
                    str_auth = f" -s {socket} -a '{db_password}' "
                else:
                    redis.StrictRedis(unix_socket_path=socket)
                    str_auth = f" -s {socket} "
        except (redis.exceptions.ConnectionError, ConnectionRefusedError) as err:
            log_and_mail.writelog('ERROR',
                                  f"Can't connect to Redis instances with with following data host='{db_host}', "
                                  f"port='{db_port}', passwd='{db_password}', socket='{socket}': {err}",
                                  config.filelog_fd, job_name)
            continue
        else:
            backup_full_tmp_path = general_function.get_full_path(
                full_path_tmp_dir,
                'redis',
                'rdb',
                gzip)
            periodic_backup.remove_local_file(options['storages'], '', job_name)

            if is_success_bgsave(str_auth, backup_full_tmp_path, gzip, job_name):
                periodic_backup.general_desc_iteration(backup_full_tmp_path, options['storages'], '', job_name,
                                                       options['safety_backup'])

    # After all the manipulations, delete the created temporary directory and
    # data inside the directory with cache davfs, but not the directory itself!
    general_function.del_file_objects(options['backup_type'], full_path_tmp_dir, '/var/cache/davfs2/*')


def is_success_bgsave(str_auth, backup_full_tmp_path, gzip, job_name):
    backup_full_tmp_path_tmp = backup_full_tmp_path.split('.gz')[0]

    dump_cmd = f"redis-cli {str_auth} --rdb {backup_full_tmp_path_tmp}"

    command = general_function.exec_cmd(dump_cmd)
    stderr = command['stderr']

    check_success_cmd = "echo $?"
    check_command = general_function.exec_cmd(check_success_cmd)
    stdout = check_command.get('stdout')

    if stdout == 1:
        log_and_mail.writelog(
            'ERROR',
            f"Can't create redis database dump '{backup_full_tmp_path_tmp}' in tmp directory:{stderr}",
            config.filelog_fd, job_name)
        return False
    else:
        if gzip:
            try:
                general_files_func.gzip_file(backup_full_tmp_path_tmp, backup_full_tmp_path)
            except general_function.MyError as stderr:
                log_and_mail.writelog(
                    'ERROR',
                    f"Can't gzip redis database dump '{backup_full_tmp_path_tmp}' in tmp directory:{stderr}.",
                    config.filelog_fd, job_name)
                return False
            else:
                log_and_mail.writelog(
                    'INFO',
                    f"Successfully created redis database dump '{backup_full_tmp_path}' in tmp directory.",
                    config.filelog_fd, job_name)
                return True
            finally:
                general_function.del_file_objects(job_name, backup_full_tmp_path_tmp)
        else:
            log_and_mail.writelog(
                'INFO',
                f"Successfully created redis database dump '{backup_full_tmp_path_tmp}' in tmp directory.",
                config.filelog_fd, job_name)
            return True
