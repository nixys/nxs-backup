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
    is_prams_read, job_name, options = general_function.get_job_parameters(job_data)
    if not is_prams_read:
        return

    full_path_tmp_dir = general_function.get_tmp_dir(options['tmp_dir'], options['backup_type'])

    dumped_ofs = {}
    for i in range(len(options['sources'])):
        exclude_list = options['sources'][i].get('excludes', '')
        try:
            target_list = options['sources'][i]['target']
            gzip = options['sources'][i]['gzip']
        except KeyError as e:
            log_and_mail.writelog('ERROR', f"Missing required key:'{e}'!",
                                  config.filelog_fd, job_name)
            continue

        # Keeping an exception list in the global variable due to the specificity of
        # the `filter` key of the `add` method of the `tarfile` class
        general_files_func.EXCLUDE_FILES = general_files_func.get_exclude_ofs(target_list, exclude_list)

        # The backup name is selected depending on the particular glob patterns from
        # the list `target_list`
        for regex in target_list:
            target_ofs_list = general_files_func.get_ofs(regex)

            if not target_ofs_list:
                log_and_mail.writelog('ERROR', "No file system objects found that" +
                                      f"match the regular expression '{regex}'!",
                                      config.filelog_fd, job_name)
                continue

            for ofs in target_ofs_list:
                # Create a backup only if the directory is not in the exception list
                # so as not to generate empty backups
                if not general_files_func.is_excluded_ofs(ofs):
                    # A function that by regularity returns the name of
                    # the backup WITHOUT EXTENSION AND DATE
                    backup_file_name = general_files_func.get_name_files_backup(regex, ofs)
                    # Get the part of the backup storage path for this archive relative to
                    # the backup dir
                    part_of_dir_path = backup_file_name.replace('___', '/')

                    backup_full_tmp_path = general_function.get_full_path(
                        full_path_tmp_dir,
                        backup_file_name,
                        'tar',
                        gzip)

                    periodic_backup.remove_local_file(options['storages'], part_of_dir_path, job_name)

                    if general_files_func.create_tar('files', backup_full_tmp_path, ofs,
                                                     gzip, options['backup_type'], job_name):
                        dumped_ofs[ofs] = {'success': True,
                                           'tmp_path': backup_full_tmp_path,
                                           'part_of_dir_path': part_of_dir_path}
                    else:
                        dumped_ofs[ofs] = {'success': False}

                    if options['deferred_copying_level'] <= 0 and dumped_ofs[ofs]['success']:
                        periodic_backup.general_desc_iteration(backup_full_tmp_path, options['storages'],
                                                               part_of_dir_path, job_name, options['safety_backup'])
                else:
                    continue

            for ofs, result in dumped_ofs.items():
                if options['deferred_copying_level'] == 1 and result['success']:
                    periodic_backup.general_desc_iteration(result['tmp_path'], options['storages'],
                                                           result['part_of_dir_path'], job_name,
                                                           options['safety_backup'])

        for ofs, result in dumped_ofs.items():
            if options['deferred_copying_level'] == 2 and result['success']:
                periodic_backup.general_desc_iteration(result['tmp_path'], options['storages'],
                                                       result['part_of_dir_path'], job_name, options['safety_backup'])

    for ofs, result in dumped_ofs.items():
        if options['deferred_copying_level'] >= 3 and result['success']:
            periodic_backup.general_desc_iteration(result['tmp_path'], options['storages'],
                                                   result['part_of_dir_path'], job_name, options['safety_backup'])

    # After all the manipulations, delete the created temporary directory and
    # data inside the directory with cache davfs, but not the directory itself!
    general_function.del_file_objects(options['backup_type'], full_path_tmp_dir, '/var/cache/davfs2/*')
