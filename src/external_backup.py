#! /usr/bin/env python3
# -*- coding: utf-8 -*-

import json
import os.path

import config
import general_function
import log_and_mail
import periodic_backup


def external_backup(job_data):
    """ Function, creates a external backup.
    At the entrance receives a dictionary with the data of the job.

    """

    job_name = 'undefined'
    try:
        job_name = job_data['job']
        backup_type = job_data['type']
        dump_cmd = job_data['dump_cmd']
        storages = job_data['storages']
    except KeyError as e:
        log_and_mail.writelog('ERROR', f"Missing required key:'{e}'!",
                              config.filelog_fd, job_name)
        return

    safety_backup = job_data.get('safety_backup', False)
    skip_backup_rotate = job_data.get('skip_backup_rotate', False)

    periodic_backup.remove_local_file(storages, '', job_name)

    command = general_function.exec_cmd(dump_cmd)
    stderr = command['stderr']
    stdout = command['stdout']
    code = command['code']

    if code != 0:
        log_and_mail.writelog('ERROR',
                              f"Bad result code external process '{dump_cmd}': '{code}' with next STDERR:\n"
                              f"'{stderr}'",
                              config.filelog_fd, job_name)
        return

    if skip_backup_rotate:
        log_and_mail.writelog('INFO', f"Command '{dump_cmd}' finished success with the next result:\n{stdout}",
                              config.filelog_fd, job_name)
        return

    source_dict = get_value_from_stdout(stderr, stdout, job_name)

    if source_dict is None:
        return

    full_tmp_path = source_dict['full_path']
    basename = source_dict['basename']
    extension = source_dict['extension']
    gzip = source_dict['gzip']

    new_name = os.path.basename(general_function.get_full_path('', basename, extension, gzip))
    new_full_tmp_path = os.path.join(os.path.dirname(full_tmp_path), new_name)

    general_function.move_ofs(full_tmp_path, new_full_tmp_path)

    periodic_backup.general_desc_iteration(new_full_tmp_path, storages, '',
                                           job_name, safety_backup)

    # After all the manipulations, delete the created temporary directory and
    # data inside the directory with cache davfs, but not the directory itself!
    general_function.del_file_objects(backup_type, '/var/cache/davfs2/*')


def get_value_from_stdout(stderr, stdout, job_name):
    """ On the input receives the data that the script sent to the stdout, stderr.
    Analyzes them and if everything is OK, then it returns the dictionary from the stdout.

    """

    if stderr:
        log_and_mail.writelog('ERROR', f"Can't create external backup in tmp directory:{stderr}",
                              config.filelog_fd, job_name)
        return None
    else:
        try:
            source_dict = json.loads(stdout)
        except ValueError as err:
            log_and_mail.writelog('ERROR', f"Can't parse output str: {err}",
                                  config.filelog_fd, job_name)
            return None
        else:
            try:
                full_path = source_dict['full_path']
                source_dict['basename']
                source_dict['extension']
                source_dict['gzip']
            except KeyError as err:
                log_and_mail.writelog('ERROR', f"Can't find required key: {err}",
                                      config.filelog_fd, job_name)
                return None
            else:
                if not os.path.isfile(full_path):
                    log_and_mail.writelog('ERROR', f"File '{full_path}' not found!",
                                          config.filelog_fd, job_name)
                    return None
                else:
                    log_and_mail.writelog('INFO', "Successfully created external backup in tmp directory.",
                                          config.filelog_fd, job_name)
                    return source_dict
