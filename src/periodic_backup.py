#! /usr/bin/env python3
# -*- coding: utf-8 -*-

import glob
import os

import config
import general_function
import log_and_mail
import mount_fuse
import specific_function


def remove_local_file(storages, part_of_dir_path, job_name, files_type='old'):
    active_flag = False
    index = 0

    for i in range(len(storages)):
        if storages[i]['storage'] == 'local' and storages[i]['enable']:
            active_flag = True
            index = i
            break

    if not active_flag:
        return 0
    else:
        backup_dir = storages[index]['backup_dir']

        days = storages[index]['store']['days']
        weeks = storages[index]['store']['weeks']
        month = storages[index]['store']['month']

        list_type_dir = ["monthly", "weekly", "daily"]

        for i in list_type_dir:
            full_path_dir = os.path.join(backup_dir, part_of_dir_path, i)
            if i == 'monthly':
                store_backup_count = month
            elif i == 'weekly':
                store_backup_count = weeks
            else:
                store_backup_count = days

            if os.path.isdir(full_path_dir):
                control_files(full_path_dir, store_backup_count, 'local', job_name, files_type)


def remove_remote_files(store_dict, storage, local_dst_dirname, part_of_dir_path, backup_dir, job_name,
                        host, share, safety_backup, files_type='old'):
    for j in list(store_dict.keys()):
        # For storage: sshfs, nfs backup_dir is the mount point and must already be created before
        # mounting. It'll be created if remote_mount_point is defined.
        # For storage: ftp, smb, webdav, s3 is NOT a mount point, but actually a relative path
        # relative to the mount point.
        if storage in ('scp', 'nfs'):
            full_path = os.path.join(local_dst_dirname, part_of_dir_path, j)
            remote_path_to_backup_dir = os.path.join(backup_dir, part_of_dir_path, j)
        else:
            full_path = os.path.join(local_dst_dirname, backup_dir.lstrip('/'), part_of_dir_path, j)
            remote_path_to_backup_dir = os.path.join(backup_dir.lstrip('/'), part_of_dir_path, j)

        store_backup_count = store_dict[j]

        control_files(full_path, store_backup_count, storage, job_name, files_type,
                      host, remote_path_to_backup_dir, share, safety_backup)


def control_files(full_dir_path, store_backup_count, storage, job_name, files_type,
                  host='', full_path_for_log='', share='', safety_backup=False):
    dow = general_function.get_time_now("dow")
    dom = general_function.get_time_now("dom")

    files_grabbed_list = []

    for extension in config.backup_extenstion:
        full_glob_path = os.path.join(full_dir_path, extension)
        files_grabbed_list.extend(glob.glob(full_glob_path))

    count_file = len(files_grabbed_list)
    time_period = os.path.split(full_dir_path)[1]

    if int(store_backup_count):
        delta_count_file = int(count_file) - int(store_backup_count)

        if ((time_period == 'weekly' and dow != config.dow_backup) or
                (time_period == 'monthly' and dom != config.dom_backup)):
            result_delete_count = delta_count_file
        else:
            result_delete_count = delta_count_file + 1

        if safety_backup:
            result_delete_count -= 1

        if result_delete_count < 1:
            return 1

        try:
            delete_oldest_files(files_grabbed_list, result_delete_count, job_name)
        except general_function.MyError as err:
            if storage == 'local':
                log_and_mail.writelog(
                    'ERROR',
                    f"Can't delete {files_type} '{time_period}' files in directory '{full_dir_path}' on '{storage}' "
                    f"storage: {err}",
                    config.filelog_fd, job_name)
            elif storage == 'smb':
                log_and_mail.writelog(
                    'ERROR',
                    f"Can't delete {files_type} '{time_period}' files in directory '{full_path_for_log}' in '{share}' "
                    f"share on '{storage}' storage({host}): {err}",
                    config.filelog_fd, job_name)
            else:
                log_and_mail.writelog(
                    'ERROR',
                    f"Can't delete {files_type} '{time_period}' files in directory '{full_path_for_log}' on '{storage}' "
                    f"storage({host}): {err}",
                    config.filelog_fd, job_name)
        else:
            if storage == 'local':
                log_and_mail.writelog(
                    'INFO',
                    f"Successfully deleted {files_type} '{time_period}' files  in directory '{full_dir_path}' on '{storage}' "
                    f"storage.",
                    config.filelog_fd, job_name)
            elif storage == 'smb':
                log_and_mail.writelog(
                    'INFO',
                    f"Successfully deleted {files_type} '{time_period}' files in directory '{full_path_for_log}' in '{share}' "
                    f"share on '{storage}' storage({host}).",
                    config.filelog_fd, job_name)
            else:
                log_and_mail.writelog(
                    'INFO',
                    f"Successfully deleted {files_type} '{time_period}' files in directory '{full_path_for_log}' on '{storage}' "
                    f"storage({host}).",
                    config.filelog_fd, job_name)
    else:
        try:
            for i in files_grabbed_list:
                general_function.del_file_objects(job_name, i)
        except general_function.MyError as err:
            if storage == 'local':
                log_and_mail.writelog(
                    'ERROR',
                    f"Can't delete {files_type} '{time_period}' files in directory '{full_dir_path}' on '{storage}' "
                    f"storage:{err}",
                    config.filelog_fd, job_name)
            elif storage == 'smb':
                log_and_mail.writelog(
                    'ERROR',
                    f"Can't delete {files_type} '{time_period}' files in directory '{full_path_for_log}' in '{share}' "
                    f"share on '{storage}' storage({host}):{err}",
                    config.filelog_fd, job_name)
            else:
                log_and_mail.writelog(
                    'ERROR',
                    f"Can't delete {files_type} '{time_period}' files in directory '{full_path_for_log}' on '{storage}' "
                    f"storage({host}):{err}",
                    config.filelog_fd, job_name)
        else:
            if storage == 'local':
                log_and_mail.writelog(
                    'INFO',
                    f"Successfully deleted {files_type} '{time_period}' files in directory '{full_dir_path}' on '{storage}' "
                    f"storage.",
                    config.filelog_fd, job_name)
            elif storage == 'smb':
                log_and_mail.writelog(
                    'INFO',
                    f"Successfully deleted {files_type} '{time_period}' files in directory '{full_path_for_log}' in '{share}' "
                    f"share on '{storage}' storage({host}).",
                    config.filelog_fd, job_name)
            else:
                log_and_mail.writelog(
                    'INFO',
                    f"Successfully deleted {files_type} '{time_period}' files in directory '{full_path_for_log}' on '{storage}' "
                    f"storage({host}).",
                    config.filelog_fd, job_name)


def delete_oldest_files(files_list, count, job_name):
    tmp_list = []

    for i in files_list:
        if os.path.exists(i):
            tmp_list.append(i)
        else:
            general_function.del_file_objects(job_name, i)
            count -= 1

    time_sorted_list = sorted(tmp_list, key=os.path.getmtime)
    length_list = len(time_sorted_list)

    if count <= 0:
        return 0
    elif count > length_list:
        count = length_list

    for i in time_sorted_list[0:count]:
        general_function.del_file_objects(job_name, i)


def general_desc_iteration(full_tmp_path, storages, part_of_dir_path, job_name, safety_backup):
    dow = general_function.get_time_now("dow")
    dom = general_function.get_time_now("dom")

    index_local_storage = -1
    for i in range(len(storages)):
        if storages[i]['storage'] == 'local':
            index_local_storage = i
            break
    if index_local_storage != -1:
        storages += [storages.pop(index_local_storage)]

    for i in range(len(storages)):
        if specific_function.is_save_to_storage(job_name, storages[i]):
            try:
                current_storage_data = mount_fuse.get_storage_data(job_name, storages[i])
            except general_function.MyError as err:
                log_and_mail.writelog('ERROR', f'{err}',
                                      config.filelog_fd, job_name)
                continue
            else:
                storage = current_storage_data['storage']
                backup_dir = current_storage_data['backup_dir']

                try:
                    mount_fuse.mount(current_storage_data)
                except general_function.MyError as err:
                    log_and_mail.writelog('ERROR', f"Can't mount remote '{storage}' storage: {err}",
                                          config.filelog_fd, job_name)
                    continue
                else:
                    remote_dir = ''  # for logging

                    if storage != 'local':
                        remote_dir = backup_dir
                        local_dst_dirname = mount_fuse.mount_point + mount_fuse.mount_point_sub_dir
                    else:
                        local_dst_dirname = backup_dir

                    days_count = storages[i]['store']['days']
                    weeks_count = storages[i]['store']['weeks']
                    month_count = storages[i]['store']['month']

                    store_dict = {'daily': days_count, 'weekly': weeks_count, 'monthly': month_count}

                    if storage != 'local':
                        host, share = general_function.get_host_and_share(storage, current_storage_data)

                        if not safety_backup:
                            remove_remote_files(store_dict, storage, local_dst_dirname, part_of_dir_path,
                                                backup_dir, job_name, host, share, safety_backup)

                    else:
                        host = ''
                        share = ''

                    subdir_name = ''
                    if type(int(month_count)) is int and int(month_count) > 0 and dom == config.dom_backup:
                        subdir_name = 'monthly'
                    elif type(int(weeks_count)) is int and int(weeks_count) > 0 and dow == config.dow_backup:
                        subdir_name = 'weekly'
                    elif type(int(days_count)) is int:
                        subdir_name = 'daily'

                    # For storage: sshfs, nfs backup_dir is the mount point and must already be created before mounting.
                    # It'll be created if remote_mount_point is defined.
                    # For storage: ftp, smb, webdav, s3 is NOT a mount point, but actually a relative path relative
                    # to the mount point
                    if storage in ('local', 'scp', 'nfs'):
                        general_local_dst_path = os.path.join(local_dst_dirname, part_of_dir_path)
                    else:
                        general_local_dst_path = os.path.join(local_dst_dirname, backup_dir.lstrip('/'),
                                                              part_of_dir_path)

                    periodic_backup(full_tmp_path, general_local_dst_path, remote_dir, storage, subdir_name, days_count,
                                    weeks_count, month_count, job_name, host, share)

                    if safety_backup and storage != 'local':
                        remove_remote_files(store_dict, storage, local_dst_dirname, part_of_dir_path,
                                            backup_dir, job_name, host, share, safety_backup)

                    try:
                        mount_fuse.unmount(storage)
                    except general_function.MyError as err:
                        log_and_mail.writelog('ERROR', f"Can't umount remote '{storage}' storage:{err}",
                                              config.filelog_fd, job_name)
                        continue
        else:
            continue


def periodic_backup(full_tmp_path, general_local_dst_path, remote_dir, storage, subdir_name, days_count, weeks_count,
                    month_count, job_name, host, share):
    daily_subdir_name = "daily"
    weekly_subdir_name = "weekly"
    monthly_subdir_name = "monthly"

    link_dict = {}

    dow = general_function.get_time_now("dow")
    backup_file_name = os.path.basename(full_tmp_path)
    full_dst_path = os.path.join(general_local_dst_path, subdir_name)

    dst_dirs = []
    daily_dir = os.path.join(general_local_dst_path, daily_subdir_name)
    weekly_dir = os.path.join(general_local_dst_path, weekly_subdir_name)
    monthly_dir = os.path.join(general_local_dst_path, monthly_subdir_name)
    execute_file_operation = False

    if subdir_name == monthly_subdir_name and int(month_count) > 0:
        execute_file_operation = True
        if storage == 'local':
            dst_dirs.append(monthly_dir)

            if dow == config.dow_backup and int(weeks_count) > 0:
                src_link = os.path.join(general_local_dst_path, monthly_subdir_name, backup_file_name)
                dst_link = os.path.join(general_local_dst_path, weekly_subdir_name, backup_file_name)
                dst_dirs.append(weekly_dir)
                link_dict[dst_link] = src_link

            if int(days_count) > 0:
                src_link = os.path.join(general_local_dst_path, monthly_subdir_name, backup_file_name)
                dst_link = os.path.join(general_local_dst_path, daily_subdir_name, backup_file_name)
                dst_dirs.append(daily_dir)
                link_dict[dst_link] = src_link
        else:
            dst_dirs.append(full_dst_path)
    elif subdir_name == weekly_subdir_name and int(weeks_count) > 0:
        execute_file_operation = True
        if storage == 'local':
            dst_dirs.append(weekly_dir)

            if int(days_count) > 0:
                src_link = os.path.join(general_local_dst_path, weekly_subdir_name, backup_file_name)
                dst_link = os.path.join(general_local_dst_path, daily_subdir_name, backup_file_name)
                dst_dirs.append(daily_dir)
                link_dict[dst_link] = src_link
        else:
            dst_dirs.append(full_dst_path)
    elif int(days_count) > 0:
        execute_file_operation = True
        if storage == 'local':
            dst_dirs.append(daily_dir)
        else:
            dst_dirs.append(full_dst_path)

    for dst_dir in set(dst_dirs):
        dirs_for_log = general_function.get_dirs_for_log(dst_dir, remote_dir, storage)
        general_function.create_dirs(job_name='', dirs_pairs={dst_dir: dirs_for_log})

    if storage == 'local' and execute_file_operation:
        try:
            general_function.move_ofs(full_tmp_path, full_dst_path)
        except general_function.MyError as err:
            log_and_mail.writelog('ERROR',
                                  f"Can't move '{subdir_name}' file '{full_tmp_path}' -> '{full_dst_path}' on "
                                  f"'{storage}' storage: {err}",
                                  config.filelog_fd, job_name)
        else:
            log_and_mail.writelog('INFO',
                                  f"Successfully moved '{subdir_name}' file '{full_tmp_path}' -> '{full_dst_path}' on "
                                  f"'{storage}' storage.",
                                  config.filelog_fd, job_name)

        if link_dict:
            for key in link_dict.keys():
                src = link_dict[key]
                dst = key

                try:
                    general_function.create_symlink(src, dst)
                except general_function.MyError as err:
                    log_and_mail.writelog('ERROR', f"Can't create symlink '{src}' -> '{dst}' on 'local' storage: {err}",
                                          config.filelog_fd, job_name)
    elif execute_file_operation:
        dirs_for_log = general_function.get_dirs_for_log(full_dst_path, remote_dir, storage)
        try:
            general_function.copy_ofs(full_tmp_path, full_dst_path)

        except general_function.MyError as err:
            if storage != 'smb':
                log_and_mail.writelog(
                    'ERROR',
                    f"Can't copy '{subdir_name}' file '{full_tmp_path}' -> '{dirs_for_log}' directory on '{storage}' "
                    f"storage({host}): {err}",
                    config.filelog_fd, job_name)
            else:
                log_and_mail.writelog(
                    'ERROR',
                    f"Can't copy '{subdir_name}' file '{full_tmp_path}' -> '{dirs_for_log}' directory in '{share}' "
                    f"share on '{storage}' storage({host}): {err}",
                    config.filelog_fd, job_name)
        else:
            if storage != 'smb':
                log_and_mail.writelog(
                    'INFO',
                    f"Successfully copied '{subdir_name}' file '{full_tmp_path}' -> '{dirs_for_log}' directory "
                    f"on '{storage}' storage({host}).",
                    config.filelog_fd, job_name)
            else:
                log_and_mail.writelog(
                    'INFO',
                    f"Successfully copied '{subdir_name}' file '{full_tmp_path}' -> '{dirs_for_log}' directory "
                    f"in '{share}' share on '{storage}' storage({host}).",
                    config.filelog_fd, job_name)
