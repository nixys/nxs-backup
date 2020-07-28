#! /usr/bin/env python3
# -*- coding: utf-8 -*-

import config
import general_files_func
import general_function
import log_and_mail
import periodic_backup


def desc_files_backup(job_data):
    """ Function, creates a desc backup of directories.
    At the entrance receives a dictionary with the data of the job.

    """

    try:
        job_name = job_data['job']
        backup_type = job_data['type']
        tmp_dir = job_data['tmp_dir']
        sources = job_data['sources']
        storages = job_data['storages']
    except KeyError as e:
        log_and_mail.writelog('ERROR', f"Missing required key:'{e}'!",
                              config.filelog_fd, job_name)
        return 1

    full_path_tmp_dir = general_function.get_tmp_dir(tmp_dir, backup_type)

    for i in range(len(sources)):
        exclude_list = sources[i].get('excludes', '')
        try:
            target_list = sources[i]['target']
            gzip = sources[i]['gzip']
        except KeyError as e:
            log_and_mail.writelog('ERROR', f"Missing required key:'{e}'!",
                                  config.filelog_fd, job_name)
            continue

        # Keeping an exception list in the global variable due to the specificity of
        # the `filter` key of the `add` method of the `tarfile` class
        general_files_func.EXCLUDE_FILES = general_files_func.get_exclude_ofs(target_list,
                                                                              exclude_list)

        # The backup name is selected depending on the particular glob patterns from
        # the list `target_list`
        for regex in target_list:
            target_ofs_list = general_files_func.get_ofs(regex)

            if not target_ofs_list:
                log_and_mail.writelog('ERROR', "No file system objects found that" + \
                                      f"match the regular expression '{regex}'!",
                                      config.filelog_fd, job_name)
                continue

            for i in target_ofs_list:
                # Create a backup only if the directory is not in the exception list
                # so as not to generate empty backups
                if not general_files_func.is_excluded_ofs(i):
                    # A function that by regularity returns the name of
                    # the backup WITHOUT EXTENSION AND DATE
                    backup_file_name = general_files_func.get_name_files_backup(regex, i)
                    # Get the part of the backup storage path for this archive relative to
                    # the backup dir
                    part_of_dir_path = backup_file_name.replace('___', '/')

                    backup_full_tmp_path = general_function.get_full_path(
                        full_path_tmp_dir,
                        backup_file_name,
                        'tar',
                        gzip)

                    periodic_backup.remove_old_local_file(storages, part_of_dir_path, job_name)

                    if general_files_func.create_tar('files', backup_full_tmp_path, i,
                                                     gzip, backup_type, job_name):
                        # If the dump collection in the temporary directory has successfully
                        # transferred the data to the specified storage
                        periodic_backup.general_desc_iteration(backup_full_tmp_path,
                                                               storages, part_of_dir_path,
                                                               job_name)
                else:
                    continue

    # After all the manipulations, delete the created temporary directory and
    # data inside the directory with cache davfs, but not the directory itself!
    general_function.del_file_objects(backup_type,
                                      full_path_tmp_dir, '/var/cache/davfs2/*')
