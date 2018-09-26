#! /usr/bin/env python3
# -*- coding: utf-8 -*-

import redis

import config
import log_and_mail
import general_function
import periodic_backup
import general_files_func


def redis_backup(job_data):
    try:
        job_name = job_data['job']
        backup_type = job_data['type']
        tmp_dir = job_data['tmp_dir']
        sources = job_data['sources']
        storages = job_data['storages']
    except KeyError as e:
        log_and_mail.writelog('ERROR', "Missing required key:'%s'!" %(e), config.filelog_fd, job_name)
        return 1

    full_path_tmp_dir = general_function.get_tmp_dir(tmp_dir, backup_type)

    for i in range(len(sources)):
        try:
            connect = sources[i]['connect']
            gzip =  sources[i]['gzip']
        except KeyError as e:
            log_and_mail.writelog('ERROR', "Missing required key:'%s'!" %(e), config.filelog_fd, job_name)
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
                    str_auth = " -h %s -p %s -a '%s' " %(db_host, db_port, db_password)
                else:
                    redis.StrictRedis(host=db_host, port=db_port)
                    str_auth = " -h %s -p %s " %(db_host, db_port)
            else:
                if db_password:
                    redis.StrictRedis(unix_socket_path=socket, password=db_password)
                    str_auth = " -s %s -a '%s' " %(socket, db_password)
                else:
                    redis.StrictRedis(unix_socket_path=socket)
                    str_auth = " -s %s " %(socket)
        except (redis.exceptions.ConnectionError, ConnectionRefusedError) as err:
            log_and_mail.writelog('ERROR', "Can't connect to Redis instances with with following data host='%s', port='%s', passwd='%s', socket='%s' :%s" %(db_host, db_port, db_password, socket, err),
                                  config.filelog_fd, job_name) 
            continue
        else:
            backup_full_tmp_path = general_function.get_full_path(
                                                                full_path_tmp_dir,
                                                                'redis', 
                                                                'rdb',
                                                                gzip)

            periodic_backup.remove_old_local_file(storages, '', job_name)

            if is_success_bgsave(str_auth, backup_full_tmp_path, gzip, job_name):
                periodic_backup.general_desc_iteration(backup_full_tmp_path, 
                                                        storages, '',
                                                        job_name)

    # After all the manipulations, delete the created temporary directory and
    # data inside the directory with cache davfs, but not the directory itself!
    general_function.del_file_objects(backup_type,
                                      full_path_tmp_dir, '/var/cache/davfs2/*')


def is_success_bgsave(str_auth, backup_full_tmp_path, gzip, job_name):

    backup_full_tmp_path_tmp = backup_full_tmp_path.split('.gz')[0]

    dump_cmd = "redis-cli %s --rdb %s" %(str_auth, backup_full_tmp_path_tmp)

    command = general_function.exec_cmd(dump_cmd)
    stderr = command['stderr']

    check_success_cmd =  "echo $?"
    check_command = general_function.exec_cmd(check_success_cmd)
    stdout = check_command.get('stdout')

    if stdout == 1:
        log_and_mail.writelog('ERROR', "Can't create redis database dump '%s' in tmp directory:%s" %(backup_full_tmp_path_tmp, stderr),
                              config.filelog_fd, job_name)
        return False
    else:
        if gzip:
            try:
                general_files_func.gzip_file(backup_full_tmp_path_tmp, backup_full_tmp_path)
            except general_function.MyError as stderr:
                log_and_mail.writelog('ERROR', "Can't gzip redis database dump '%s' in tmp directory:%s." %(backup_full_tmp_path_tmp, stderr),
                                      config.filelog_fd, job_name)
                return False
            else:
                log_and_mail.writelog('INFO', "Successfully created redis database dump '%s' in tmp directory." %(backup_full_tmp_path),
                                      config.filelog_fd, job_name)
                return True
            finally:
                general_function.del_file_objects(job_name, backup_full_tmp_path_tmp)
        else:
            log_and_mail.writelog('INFO', "Successfully created redis database dump '%s' in tmp directory." %(backup_full_tmp_path_tmp),
                              config.filelog_fd, job_name)
            return True
