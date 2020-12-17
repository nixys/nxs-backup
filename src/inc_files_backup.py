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

    is_prams_read, job_name, backup_type, tmp_dir, sources, storages, safety_backup, deferred_copying_level = \
        general_function.get_job_parameters(job_data)
    if not is_prams_read:
        return

    for i in range(len(sources)):
        target_list = sources[i]['target']
        exclude_list = sources[i].get('excludes', '')
        gzip = sources[i]['gzip']

        # Keeping an exception list in the global variable due to the specificity of
        # the `filter` key of the `add` method of the `tarfile` class
        general_files_func.EXCLUDE_FILES = general_files_func.get_exclude_ofs(target_list,
                                                                              exclude_list)

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

                    for j in range(len(storages)):
                        if specific_function.is_save_to_storage(job_name, storages[j]):
                            try:
                                current_storage_data = mount_fuse.get_storage_data(job_name,
                                                                                   storages[j])
                            except general_function.MyError as err:
                                log_and_mail.writelog('ERROR', f'{err}',
                                                      config.filelog_fd, job_name)
                                continue
                            else:
                                storage = current_storage_data['storage']
                                backup_dir = current_storage_data['backup_dir']
                                # Если storage активный - монтируем его
                                try:
                                    mount_fuse.mount(current_storage_data)
                                except general_function.MyError as err:
                                    log_and_mail.writelog('ERROR', f"Can't mount remote '{storage}' storage :{err}",
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

                                    create_inc_file(local_dst_dirname, remote_dir, part_of_dir_path, backup_file_name,
                                                    ofs, exclude_list, gzip, job_name, storage, host, share)

                                    try:
                                        mount_fuse.unmount()
                                    except general_function.MyError as err:
                                        log_and_mail.writelog('ERROR',
                                                              f"Can't umount remote '{storage}' storage :{err}",
                                                              config.filelog_fd, job_name)
                                        continue
                        else:
                            continue
                else:
                    continue


def create_inc_file(local_dst_dirname, remote_dir, part_of_dir_path, backup_file_name,
                    target, exclude_list, gzip, job_name, storage, host, share):
    """ The function determines whether to collect a full backup or incremental,
    prepares all the necessary information.

    """

    date_year = general_function.get_time_now('year')
    date_month = general_function.get_time_now('moy')
    date_day = general_function.get_time_now('dom')

    if int(date_day) < 11:
        daily_prefix = 'day_01'
    elif int(date_day) < 21:
        daily_prefix = 'day_11'
    else:
        daily_prefix = 'day_21'

    year_dir = os.path.join(local_dst_dirname, part_of_dir_path, date_year)
    initial_dir = os.path.join(year_dir, 'year')  # Path to full backup
    month_dir = os.path.join(year_dir, f'month_{date_month}', 'monthly')
    daily_dir = os.path.join(year_dir, f'month_{date_month}', 'daily', daily_prefix)

    year_inc_file = os.path.join(initial_dir, 'year.inc')
    month_inc_file = os.path.join(month_dir, 'month.inc')
    daily_inc_file = os.path.join(daily_dir, 'daily.inc')

    link_dict = {}  # dict for symlink with pairs like dst: src
    copy_dict = {}  # dict for copy with pairs like dst: src

    # Before we proceed to collect a copy, we need to delete the copies for the same month last year
    # if they are to not save extra archives

    old_year = int(date_year) - 1
    old_year_dir = os.path.join(local_dst_dirname, part_of_dir_path, str(old_year))
    if os.path.isdir(old_year_dir):
        old_month_dir = os.path.join(old_year_dir, f'month_{date_month}')
        del_old_inc_file(old_year_dir, old_month_dir)

    if not os.path.isfile(year_inc_file):
        # There is no original index file, so we need to check the existence of an year directory
        if os.path.isdir(year_dir):
            # There is a directory, but there is no file itself, then something went wrong, so
            # we delete this directory with all the data inside, because even if they are there
            # continue to collect incremental copies it will not be able to
            general_function.del_file_objects(job_name, year_dir)
            dirs_for_log = general_function.get_dirs_for_log(year_dir, remote_dir, storage)
            file_for_log = os.path.join(dirs_for_log, os.path.basename(year_inc_file))
            log_and_mail.writelog('ERROR',
                                  f"The file {file_for_log} not found, so the directory {dirs_for_log} is cleared. "
                                  f"Incremental backup will be reinitialized ",
                                  config.filelog_fd, job_name)

        # Initialize the incremental backup, i.e. collect a full copy
        dirs_for_log = general_function.get_dirs_for_log(initial_dir, remote_dir, storage)
        general_function.create_dirs(job_name=job_name, dirs_pairs={initial_dir: dirs_for_log})

        # Get the current list of files and write to the year inc file
        meta_info = get_index(target, exclude_list)
        with open(year_inc_file, "w") as index_file:
            json.dump(meta_info, index_file)

        full_backup_path = general_function.get_full_path(initial_dir,
                                                          backup_file_name,
                                                          'tar',
                                                          gzip)

        general_files_func.create_tar('files', full_backup_path, target,
                                      gzip, 'inc_files', job_name,
                                      remote_dir, storage, host, share)

        # After creating the full copy, you need to make the symlinks for the inc.file and
        # the most collected copy in the month directory of the current month
        # as well as in the decade directory if it's local, scp the repository and
        # copy inc.file for other types of repositories that do not support symlynk.

        month_dirs_for_log = general_function.get_dirs_for_log(month_dir, remote_dir, storage)
        daily_dirs_for_log = general_function.get_dirs_for_log(daily_dir, remote_dir, storage)
        general_function.create_dirs(job_name=job_name, dirs_pairs={month_dir: month_dirs_for_log,
                                                                    daily_dir: daily_dirs_for_log})

        if storage in 'local, scp':
            link_dict[month_inc_file] = year_inc_file
            link_dict[os.path.join(month_dir, os.path.basename(full_backup_path))] = full_backup_path
            link_dict[daily_inc_file] = year_inc_file
            link_dict[os.path.join(daily_dir, os.path.basename(full_backup_path))] = full_backup_path
        else:
            copy_dict[month_inc_file] = year_inc_file
            copy_dict[daily_inc_file] = year_inc_file
    else:
        symlink_dir = ''
        if int(date_day) == 1:
            # It is necessary to collect monthly incremental backup relative to the year copy
            old_meta_info = specific_function.parser_json(year_inc_file)
            new_meta_info = get_index(target, exclude_list)

            general_inc_backup_dir = month_dir

            # It is also necessary to make a symlink for inc files and backups to the directory with the first decade
            symlink_dir = daily_dir

            general_dirs_for_log = general_function.get_dirs_for_log(general_inc_backup_dir, remote_dir, storage)
            symlink_dirs_for_log = general_function.get_dirs_for_log(symlink_dir, remote_dir, storage)
            general_function.create_dirs(job_name=job_name, dirs_pairs={general_inc_backup_dir: general_dirs_for_log,
                                                                        symlink_dir: symlink_dirs_for_log})

            with open(month_inc_file, "w") as index_file:
                json.dump(new_meta_info, index_file)

        elif int(date_day) == 11 or int(date_day) == 21:
            # It is necessary to collect a ten-day incremental backup relative to a monthly copy
            try:
                old_meta_info = specific_function.parser_json(month_inc_file)
            except general_function.MyError as e:
                log_and_mail.writelog('ERROR', f"Couldn't open old month meta info file '{month_inc_file}': {e}!",
                                      config.filelog_fd, job_name)
                return 2

            new_meta_info = get_index(target, exclude_list)

            general_inc_backup_dir = daily_dir
            general_dirs_for_log = general_function.get_dirs_for_log(general_inc_backup_dir, remote_dir, storage)
            general_function.create_dirs(job_name=job_name, dirs_pairs={general_inc_backup_dir: general_dirs_for_log})

            with open(daily_inc_file, "w") as index_file:
                json.dump(new_meta_info, index_file)
        else:
            # It is necessary to collect a normal daily incremental backup relative to a ten-day copy
            try:
                old_meta_info = specific_function.parser_json(daily_inc_file)
            except general_function.MyError as e:
                log_and_mail.writelog('ERROR', f"Couldn't open old decade meta info file '{daily_inc_file}': {e}!",
                                      config.filelog_fd, job_name)
                return 2

            new_meta_info = get_index(target, exclude_list)

            general_inc_backup_dir = daily_dir
            general_dirs_for_log = general_function.get_dirs_for_log(general_inc_backup_dir, remote_dir, storage)
            general_function.create_dirs(job_name=job_name, dirs_pairs={general_inc_backup_dir: general_dirs_for_log})

        # Calculate the difference between the old and new file states
        diff_json = compute_diff(new_meta_info, old_meta_info)

        inc_backup_path = general_function.get_full_path(general_inc_backup_dir,
                                                         backup_file_name,
                                                         'tar',
                                                         gzip)

        # Define the list of files that need to be included in the archive
        target_change_list = diff_json['modify']

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

        create_inc_tar(inc_backup_path, remote_dir, dict_directory, target_change_list, gzip, job_name, storage, host,
                       share)

        if symlink_dir:
            if storage in 'local, scp':
                link_dict[daily_inc_file] = month_inc_file
            else:
                copy_dict[daily_inc_file] = month_inc_file

    if link_dict:
        for key in link_dict.keys():
            src = link_dict[key]
            dst = key

            try:
                general_function.create_symlink(src, dst)
            except general_function.MyError as err:
                log_and_mail.writelog('ERROR', f"Can't create symlink {src} -> {dst}: {err}",
                                      config.filelog_fd, job_name)

    if copy_dict:
        for key in copy_dict.keys():
            src = copy_dict[key]
            dst = key

            try:
                general_function.copy_ofs(src, dst)
            except general_function.MyError as err:
                log_and_mail.writelog('ERROR', f"Can't copy {src} -> {dst}: {err}",
                                      config.filelog_fd, job_name)


def del_old_inc_file(old_year_dir, old_month_dir):
    general_function.del_file_objects('inc_files', old_month_dir)

    list_subdir_in_old_dir = os.listdir(old_year_dir)

    if len(list_subdir_in_old_dir) == 1 and list_subdir_in_old_dir[0] == 'year':
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
