#! /usr/bin/env python3
# -*- coding: utf-8 -*-

import fnmatch
import json
import os
import re
import tarfile

import config
import general_files_func
import general_function
import log_and_mail
import mount_fuse
import specific_function


def inc_files_backup(job_data):
    """ The function collects an incremental backup for the specified partition.

    """

    is_prams_read, job_name, job_options = general_function.get_job_parameters(job_data)
    if not is_prams_read:
        return

    for i in range(len(job_options['sources'])):
        target_list = job_options['sources'][i]['target']
        exclude_list = job_options['sources'][i].get('excludes', '')
        gzip = job_options['sources'][i]['gzip']

        # Keeping an exception list in the global variable due to the specificity of
        # the `filter` key of the `add` method of the `tarfile` class
        general_files_func.EXCLUDE_FILES = general_files_func.get_exclude_ofs(target_list, exclude_list)

        # The backup name is selected depending on the particular glob patterns from
        # the list `target_list`
        for regex in target_list:
            target_ofs_list = general_files_func.get_ofs(regex)

            for ofs in target_ofs_list:
                if not general_files_func.is_excluded_ofs(ofs):
                    # Create a backup only if the directory is not in the exception list
                    # so as not to generate empty backups

                    # A function that by regularity returns the name of
                    # the backup WITHOUT EXTENSION AND DATE
                    backup_file_name = general_files_func.get_name_files_backup(regex, ofs)

                    # Get the part of the backup storage path for this archive relative to
                    # the backup dir
                    part_of_dir_path = backup_file_name.replace('___', '/')

                    for j in range(len(job_options['storages'])):
                        if specific_function.is_save_to_storage(job_name, job_options['storages'][j]):
                            try:
                                current_storage_data = mount_fuse.get_storage_data(job_name, job_options['storages'][j])
                            except general_function.MyError as err:
                                log_and_mail.writelog('ERROR', f'{err}', config.filelog_fd, job_name)
                                continue
                            else:
                                storage = current_storage_data['storage']
                                backup_dir = current_storage_data['backup_dir']
                                # Если storage активный - монтируем его
                                try:
                                    mount_fuse.mount(current_storage_data)
                                except general_function.MyError as err:
                                    log_and_mail.writelog('ERROR', f"Can't mount remote '{storage}' storage: {err}",
                                                          config.filelog_fd, job_name)
                                    continue
                                else:
                                    remote_dir = ''  # Only for logging
                                    if storage != 'local':
                                        local_dst_dirname = mount_fuse.mount_point + mount_fuse.mount_point_sub_dir
                                        remote_dir = backup_dir
                                        if storage != 's3':
                                            host = current_storage_data['host']
                                        else:
                                            host = ''
                                        share = current_storage_data.get('share')
                                    else:
                                        host = ''
                                        share = ''
                                        local_dst_dirname = backup_dir

                                    if storage not in ('local', 'scp', 'nfs'):
                                        local_dst_dirname = os.path.join(local_dst_dirname, backup_dir.lstrip('/'))

                                    create_inc_backup(local_dst_dirname, remote_dir, part_of_dir_path, backup_file_name,
                                                      ofs, exclude_list, gzip, job_name, storage, host, share,
                                                      job_options['months_to_store'])

                                    try:
                                        mount_fuse.unmount(storage)
                                    except general_function.MyError as err:
                                        log_and_mail.writelog('ERROR',
                                                              f"Can't umount remote '{storage}' storage :{err}",
                                                              config.filelog_fd, job_name)
                                        continue
                        else:
                            continue
                else:
                    continue


def get_dated_paths(local_dst_dirname, part_of_dir_path, date_year, date_month, date_day):
    """

    :rtype: dict
    :param str local_dst_dirname:
    :param str part_of_dir_path:
    :param date_year:
    :param date_month:
    :param date_day:
    :return: Dict with next keys: initial_dir, daily_dir, month_dir, year_dir, old_year_dir, daily_inc_file,
        month_inc_file, year_inc_file
    """
    dated_paths = {}

    if int(date_day) < 11:
        daily_prefix = 'day_01'
    elif int(date_day) < 21:
        daily_prefix = 'day_11'
    else:
        daily_prefix = 'day_21'

    old_year = int(date_year) - 1

    dated_paths['year_dir'] = year_dir = os.path.join(local_dst_dirname, part_of_dir_path, date_year)
    dated_paths['initial_dir'] = initial_dir = os.path.join(year_dir, 'year')  # Path to full backup
    dated_paths['month_dir'] = month_dir = os.path.join(year_dir, f'month_{date_month}', 'monthly')
    dated_paths['daily_dir'] = daily_dir = os.path.join(year_dir, f'month_{date_month}', 'daily', daily_prefix)
    dated_paths['year_inc_file'] = os.path.join(initial_dir, 'year.inc')
    dated_paths['month_inc_file'] = os.path.join(month_dir, 'month.inc')
    dated_paths['daily_inc_file'] = os.path.join(daily_dir, 'daily.inc')
    dated_paths['old_year_dir'] = os.path.join(local_dst_dirname, part_of_dir_path, str(old_year))

    return dated_paths


def write_meta_info(inc_file_path, inc_file_data):
    with open(inc_file_path, "w") as index_file:
        json.dump(inc_file_data, index_file)


def get_dict_directory(target, diff_json):
    # Form GNU.dumpdir headers
    dict_directory = {}  # Dict to store pairs like dir:GNU.dumpdir

    excludes = r'|'.join([fnmatch.translate(x)[:-2] for x in general_files_func.EXCLUDE_FILES]) or r'$.'

    for dir_name, dirs, files in os.walk(target):
        first_level_files = []
        if re.match(excludes, dir_name):
            continue
        for file in files:
            if re.match(excludes, os.path.join(dir_name, file)):
                continue
            first_level_files.append(file)
        first_level_subdirs = dirs
        dict_directory[dir_name] = get_gnu_dumpdir_format(diff_json, dir_name, target, excludes,
                                                          first_level_subdirs, first_level_files)
    return dict_directory


def create_links_and_copies(link_dict, copy_dict, job_name):
    if link_dict:
        for dst, src in link_dict.items():
            try:
                general_function.create_symlink(src, dst)
            except general_function.MyError as err:
                log_and_mail.writelog('ERROR', f"Can't create symlink {src} -> {dst}: {err}",
                                      config.filelog_fd, job_name)

    if copy_dict:
        for dst, src in copy_dict.items():
            try:
                general_function.copy_ofs(src, dst)
            except general_function.MyError as err:
                log_and_mail.writelog('ERROR', f"Can't copy {src} -> {dst}: {err}",
                                      config.filelog_fd, job_name)


def create_inc_backup(local_dst_dirname, remote_dir, part_of_dir_path, backup_file_name,
                      target, exclude_list, gzip, job_name, storage, host, share, months_to_store):
    """ The function determines whether to collect a full backup or incremental,
    prepares all the necessary information.

    """
    date_year = general_function.get_time_now('year')
    date_month = general_function.get_time_now('moy')
    date_day = general_function.get_time_now('dom')

    dated_paths = get_dated_paths(local_dst_dirname, part_of_dir_path, date_year, date_month, date_day)

    # Before we proceed to collect a copy, we need to delete the copies for the same month last year
    # if they are to not save extra archives
    old_month_dirs = []
    if os.path.isdir(dated_paths['old_year_dir']) or months_to_store < 12:
        if months_to_store < 12:
            int_date_month = int(date_month)
            last_month = int_date_month - months_to_store
            if last_month <= 0:
                m_range = list(range(last_month+12, 13))
                m_range.extend(list(range(1, int_date_month)))
            else:
                m_range = list(range(last_month, int_date_month))
            for i in range(1, 13):
                if i not in m_range:
                    date = str(i).zfill(2)
                    if i < int(date_month):
                        year_to_cleanup = dated_paths['year_dir']
                    else:
                        year_to_cleanup = dated_paths['old_year_dir']
                    old_month_dirs.append(os.path.join(year_to_cleanup, f'month_{date}'))
        else:
            old_month_dirs.append(os.path.join(dated_paths['old_year_dir'], f'month_{date_month}'))
        del_old_inc_file(dated_paths['old_year_dir'], old_month_dirs)

    link_dict = {}  # dict for symlink with pairs like dst: src
    copy_dict = {}  # dict for copy with pairs like dst: src

    # Get the current list of files
    new_meta_info = get_index(target, exclude_list)

    if not os.path.isfile(dated_paths['year_inc_file']):
        # There is no original index file, so we need to check the existence of an year directory
        if os.path.isdir(dated_paths['year_dir']):
            # There is a directory, but there is no file itself, then something went wrong, so
            # we delete this directory with all the data inside, because even if they are there
            # continue to collect incremental copies it will not be able to
            general_function.del_file_objects(job_name, dated_paths['year_dir'])
            dirs_for_log = general_function.get_dirs_for_log(dated_paths['year_dir'], remote_dir, storage)
            file_for_log = os.path.join(dirs_for_log, os.path.basename(dated_paths['year_inc_file']))
            log_and_mail.writelog('ERROR',
                                  f"The file {file_for_log} not found, so the directory {dirs_for_log} is cleared. "
                                  f"Incremental backup will be reinitialized ",
                                  config.filelog_fd, job_name)

        # Initialize the incremental backup, i.e. collect a full copy
        remote_dir_for_logs = general_function.get_dirs_for_log(dated_paths['initial_dir'], remote_dir, storage)
        general_function.create_dirs(job_name=job_name, dirs_pairs={dated_paths['initial_dir']: remote_dir_for_logs})

        write_meta_info(dated_paths['year_inc_file'], new_meta_info)

        full_backup_path = general_function.get_full_path(dated_paths['initial_dir'],
                                                          backup_file_name,
                                                          'tar',
                                                          gzip)

        general_files_func.create_tar('files', full_backup_path, target,
                                      gzip, 'inc_files', job_name,
                                      remote_dir, storage, host, share)

        daily_dirs_remote = general_function.get_dirs_for_log(dated_paths['daily_dir'], remote_dir, storage)
        month_dirs_remote = general_function.get_dirs_for_log(dated_paths['month_dir'], remote_dir, storage)
        general_function.create_dirs(job_name=job_name, dirs_pairs={dated_paths['daily_dir']: daily_dirs_remote,
                                                                    dated_paths['month_dir']: month_dirs_remote})

        if storage in 'local':
            link_dict[dated_paths['month_inc_file']] = dated_paths['year_inc_file']
            link_dict[os.path.join(dated_paths['month_dir'], os.path.basename(full_backup_path))] = full_backup_path
            link_dict[dated_paths['daily_inc_file']] = dated_paths['year_inc_file']
            link_dict[os.path.join(dated_paths['daily_dir'], os.path.basename(full_backup_path))] = full_backup_path
        elif storage in 'scp, nfs':
            copy_dict[dated_paths['month_inc_file']] = dated_paths['year_inc_file']
            link_dict[os.path.join(dated_paths['month_dir'], os.path.basename(full_backup_path))] = \
                full_backup_path.replace(local_dst_dirname, remote_dir)
            copy_dict[dated_paths['daily_inc_file']] = dated_paths['year_inc_file']
            link_dict[os.path.join(dated_paths['daily_dir'], os.path.basename(full_backup_path))] = \
                full_backup_path.replace(local_dst_dirname, remote_dir)
        else:
            copy_dict[dated_paths['month_inc_file']] = dated_paths['year_inc_file']
            copy_dict[os.path.join(dated_paths['month_dir'], os.path.basename(full_backup_path))] = full_backup_path
            copy_dict[dated_paths['daily_inc_file']] = dated_paths['year_inc_file']
            copy_dict[os.path.join(dated_paths['daily_dir'], os.path.basename(full_backup_path))] = full_backup_path

    else:
        symlink_dir = ''
        meta_path = ''
        if int(date_day) == 1:
            meta_path = dated_paths['month_inc_file']
            old_meta_path = dated_paths['year_inc_file']
            general_inc_backup_dir = dated_paths['month_dir']
            symlink_dir = dated_paths['daily_dir']
        elif int(date_day) == 11 or int(date_day) == 21:
            meta_path = dated_paths['daily_inc_file']
            old_meta_path = dated_paths['month_inc_file']
            general_inc_backup_dir = dated_paths['daily_dir']
        else:
            old_meta_path = dated_paths['daily_inc_file']
            general_inc_backup_dir = dated_paths['daily_dir']

        try:
            old_meta_info = specific_function.parser_json(old_meta_path)
        except general_function.MyError as e:
            log_and_mail.writelog('ERROR',
                                  f"Couldn't open old meta info file '{old_meta_path}': {e}!",
                                  config.filelog_fd, job_name)
            return 2

        general_dirs_for_log = general_function.get_dirs_for_log(general_inc_backup_dir, remote_dir, storage)
        general_function.create_dirs(job_name=job_name, dirs_pairs={general_inc_backup_dir: general_dirs_for_log})
        if meta_path:
            write_meta_info(meta_path, new_meta_info)

        # Calculate the difference between the old and new file states
        diff_json = compute_diff(new_meta_info, old_meta_info)

        # Define the list of files that need to be included in the archive
        target_change_list = diff_json['modify']

        dict_directory = get_dict_directory(target, diff_json)

        inc_backup_path = general_function.get_full_path(general_inc_backup_dir, backup_file_name, 'tar', gzip)
        create_inc_tar(
            inc_backup_path, remote_dir, dict_directory, target_change_list, gzip, job_name, storage, host, share
        )

        if symlink_dir:
            symlink_dirs_for_log = general_function.get_dirs_for_log(symlink_dir, remote_dir, storage)
            general_function.create_dirs(job_name=job_name, dirs_pairs={symlink_dir: symlink_dirs_for_log})
            if storage in 'local':
                link_dict[dated_paths['daily_inc_file']] = dated_paths['month_inc_file']
            elif storage in 'scp, nfs':
                copy_dict[dated_paths['daily_inc_file'].replace(local_dst_dirname, remote_dir)] = \
                    dated_paths['month_inc_file'].replace(local_dst_dirname, remote_dir)
            else:
                copy_dict[dated_paths['daily_inc_file']] = dated_paths['month_inc_file']

    create_links_and_copies(link_dict, copy_dict, job_name)


def del_old_inc_file(old_year_dir, old_month_dirs):
    """

    :param str old_year_dir:
    :param list old_month_dirs:
    """
    for old_month_dir in old_month_dirs:
        general_function.del_file_objects('inc_files', old_month_dir)

    if os.path.isdir(old_year_dir):
        list_subdir_in_old_dir = os.listdir(old_year_dir)

        if len(list_subdir_in_old_dir) == 1 and \
                list_subdir_in_old_dir[0] == 'year' and \
                old_year_dir != general_function.get_time_now('year'):
            general_function.del_file_objects('inc_files', old_year_dir)


def get_gnu_dumpdir_format(diff_json, dir_name, backup_dir, excludes, first_level_subdirs, first_level_files):
    """ The function on the input receives a dictionary with modified files.

    """

    delimiter = '\0'
    not_modify_special_symbol = 'N'
    modify_special_symbol = 'Y'
    directory_special_symbol = 'D'

    general_dict = {}

    if first_level_subdirs:
        for i in first_level_subdirs:
            general_dict[i] = directory_special_symbol

    if first_level_files:
        for file in first_level_files:
            if os.path.join(dir_name, file) in diff_json['modify']:
                general_dict[file] = modify_special_symbol
            else:
                general_dict[file] = not_modify_special_symbol

    keys = list(general_dict.keys())
    keys.sort()

    result = ''
    for i in range(len(keys)):
        result += general_dict.get(keys[i]) + keys[i] + delimiter

    result += delimiter

    return result


def get_index(backup_dir, exclude_list):
    """ Return a tuple containing:
    - a dict: filepath => ctime
    """

    file_index = {}

    excludes = r'|'.join([fnmatch.translate(x)[:-2] for x in general_files_func.EXCLUDE_FILES]) or r'$.'

    for root, dirs, filenames in os.walk(backup_dir):

        filenames = [os.path.join(root, f) for f in filenames]
        filenames = [f for f in filenames if not re.match(excludes, f)]

        for f in filenames:
            if os.path.isfile(f):
                file_index[f] = os.path.getmtime(f)

    return file_index


def compute_diff(new_meta_info, old_meta_info):
    data = {}

    created_files = list(set(new_meta_info.keys()) - set(old_meta_info.keys()))
    updated_files = []

    data['modify'] = []
    data['not_modify'] = []

    for f in set(old_meta_info.keys()).intersection(set(new_meta_info.keys())):
        try:
            if new_meta_info[f] != old_meta_info[f]:
                updated_files.append(f)
            else:
                data['not_modify'].append(f)
        except KeyError:
            # Occurs when in one of the states (old or new) one and the same path
            # are located both the broken and normal file
            updated_files.append(f)

    data['modify'] = created_files + updated_files

    return data


def create_inc_tar(path_to_tarfile, remote_dir, dict_directory, target_change_list, gzip, job_name, storage, host,
                   share):
    """ The function creates an incremental backup based on the GNU.dumpdir header in the PAX format.

    """

    dirs_for_log = general_function.get_dirs_for_log(os.path.dirname(path_to_tarfile), remote_dir, storage)
    file_for_log = os.path.join(dirs_for_log, os.path.basename(path_to_tarfile))

    try:
        if gzip:
            out_tarfile = tarfile.open(path_to_tarfile, mode='w:gz', format=tarfile.PAX_FORMAT)
        else:
            out_tarfile = tarfile.open(path_to_tarfile, mode='w:', format=tarfile.PAX_FORMAT)

        for i in dict_directory.keys():
            try:
                meta_file = out_tarfile.gettarinfo(name=i)
                pax_headers = {
                    'GNU.dumpdir': dict_directory.get(i)
                }
                meta_file.pax_headers = pax_headers

                out_tarfile.addfile(meta_file)
            except FileNotFoundError:
                continue

        for i in target_change_list:
            try:
                out_tarfile.add(i)
            except FileNotFoundError:
                continue

        out_tarfile.close()
    except tarfile.TarError as err:
        if storage == 'local':
            log_and_mail.writelog(
                'ERROR',
                f"Can't create incremental '{file_for_log}' archive on '{storage}' storage: {err}",
                config.filelog_fd, job_name)
        elif storage == 'smb':
            log_and_mail.writelog(
                'ERROR',
                f"Can't create incremental '{file_for_log}' archive in '{share}' share on '{storage}' "
                f"storage({host}): {err}",
                config.filelog_fd, job_name)
        else:
            log_and_mail.writelog(
                'ERROR',
                f"Can't create incremental '{file_for_log}' archive on '{storage}' storage({host}): {err}",
                config.filelog_fd, job_name)
        return False
    else:
        if storage == 'local':
            log_and_mail.writelog(
                'INFO',
                f"Successfully created incremental '{file_for_log}' archive on '{storage}' storage.",
                config.filelog_fd, job_name)
        elif storage == 'smb':
            log_and_mail.writelog(
                'INFO',
                f"Successfully created incremental '{file_for_log}' archive in '{share}' share on '{storage}' "
                f"storage({host}).",
                config.filelog_fd, job_name)
        else:
            log_and_mail.writelog(
                'INFO',
                f"Successfully created incremental '{file_for_log}' archive on '{storage}' storage({host}).",
                config.filelog_fd, job_name)
        return True
