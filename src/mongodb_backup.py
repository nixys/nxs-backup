#! /usr/bin/env python3
# -*- coding: utf-8 -*-

import os
import re

import pymongo

import config
import general_function
import log_and_mail
import periodic_backup


def is_real_mongo_err(str_err):
    """ The function that determines the criticality of the information provided
    in the stderr of the mongodump command.

    """

    if str_err:
        if re.search('Failed', str_err, re.I):
            return True
        else:
            return False
    else:
        return False


def mongodb_backup(job_data):
    """ Function, creates a mongodb backup.
    At the entrance receives a dictionary with the data of the job.

    """

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
        exclude_dbs_list = sources[i].get('exclude_dbs', [])
        exclude_collections_list = sources[i].get('exclude_collections', [])
        try:
            connect = sources[i]['connect']
            target_db_list = sources[i]['target_dbs']
            target_collection_list = sources[i]['target_collections']
            gzip = sources[i]['gzip']
            extra_keys = sources[i]['extra_keys']
        except KeyError as e:
            log_and_mail.writelog('ERROR', f"Missing required key:'{e}'!", config.filelog_fd, job_name)
            continue

        db_host = connect.get('db_host')
        db_port = connect.get('db_port')
        db_user = connect.get('db_user')
        db_password = connect.get('db_password')

        if not (db_host and not (bool(db_user) ^ bool(db_password))):
            log_and_mail.writelog('ERROR', "Can't find the authentication data, please fill in the required fields",
                                  config.filelog_fd, job_name)
            continue

        if not db_port:
            db_port = general_function.get_default_port('mongodb')

        is_all_flag_db = is_all_flag_collection = False

        if 'all' in target_db_list:
            is_all_flag_db = True

        if 'all' in target_collection_list:
            is_all_flag_collection = True

        if db_user:
            uri = f"mongodb://{db_user}:{db_password}@{db_host}:{db_port}/"  # for pymongo
            str_auth = f" --host {db_host} --port {db_port} --username {db_user} --password {db_password} "  # for mongodump
        else:
            uri = f"mongodb://{db_host}:{db_port}/"
            str_auth = f" --host {db_host} --port {db_port} "

        if is_all_flag_db:
            try:
                client = pymongo.MongoClient(uri)
                target_db_list = client.database_names()
            except pymongo.errors.PyMongoError as err:
                log_and_mail.writelog('ERROR',
                                      f"Can't connect to MongoDB instances with the following data host='{db_host}', port='{db_port}', user='{db_user}', passwd='{db_password}':{err}",
                                      config.filelog_fd, job_name)
                continue
            finally:
                client.close()

        for db in target_db_list:
            if not db in exclude_dbs_list:
                try:
                    client = pymongo.MongoClient(uri)
                    current_db = client[db]
                    collection_list = current_db.collection_names()
                except pymongo.errors.PyMongoError as err:
                    log_and_mail.writelog('ERROR',
                                          f"Can't connect to MongoDB instances with the following data host='{db_host}', port='{db_port}', user='{db_user}', passwd='{db_password}':{err}",
                                          config.filelog_fd, job_name)
                    continue
                finally:
                    client.close()

                if is_all_flag_collection:
                    target_collection_list = collection_list

                for collection in target_collection_list:
                    if not collection in exclude_collections_list and collection in collection_list:
                        str_auth_finally = f"{str_auth} --collection {collection} "

                        backup_full_tmp_path = general_function.get_full_path(
                            full_path_tmp_dir,
                            collection,
                            'mongodump',
                            gzip)

                        part_of_dir_path = os.path.join(db, collection)
                        periodic_backup.remove_old_local_file(storages, part_of_dir_path, job_name)

                        if is_success_mongodump(collection, db, extra_keys, str_auth_finally, backup_full_tmp_path,
                                                gzip, job_name):
                            periodic_backup.general_desc_iteration(backup_full_tmp_path,
                                                                   storages, part_of_dir_path,
                                                                   job_name)

    # After all the manipulations, delete the created temporary directory and
    # data inside the directory with cache davfs, but not the directory itself!
    general_function.del_file_objects(backup_type,
                                      full_path_tmp_dir, '/var/cache/davfs2/*')


def is_success_mongodump(collection, db, extra_keys, str_auth, backup_full_path, gzip, job_name):
    if gzip:
        dump_cmd = f"mongodump --db {db} {extra_keys} {str_auth}  --out -| gzip > {backup_full_path}"
    else:
        dump_cmd = f"mongodump --db {db} {extra_keys} {str_auth} --out - > {backup_full_path}"

    command = general_function.exec_cmd(dump_cmd)

    stderr = command['stderr']
    code = command['code']

    if stderr and is_real_mongo_err(stderr):
        log_and_mail.writelog('ERROR',
                              f"Can't create collection '{collection}' in '{db}' database dump in tmp directory:{stderr}",
                              config.filelog_fd, job_name)
        return False
    elif code != 0:
        log_and_mail.writelog('ERROR', f"Bad result code external process '{dump_cmd}':'{code}'",
                              config.filelog_fd, job_name)
        return False
    else:
        log_and_mail.writelog('INFO',
                              f"Successfully created collection '{collection}' in '{db}' database dump in tmp directory.",
                              config.filelog_fd, job_name)
        return True
