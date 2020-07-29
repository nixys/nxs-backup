#! /usr/bin/env python3
# -*- coding: utf-8 -*-

import glob
import gzip
import os
import re
import shutil
import tarfile

import config
import general_function
import log_and_mail

EXCLUDE_FILES = ''


def filter_function(tarinfo):
    """ The function determines whether the object falls under the exception rules.
    The input receives a TarInfo class object. Returns:
       tarinfo - if the object does not fall into the exclusion rules;
       None - otherwise.
    """

    ofs_name = tarinfo.name

    # Since tar removes all the initial '/' from the archive objects
    #  for correct comparison it falls under the exceptions it is necessary to add '/'
    if not ofs_name.startswith('/'):
        ofs_name = '/%s' % ofs_name

    if not ofs_name.endswith('/') and os.path.isdir(ofs_name):
        ofs_name_alternative = '%s/' % ofs_name
    else:
        ofs_name_alternative = ofs_name

    if (ofs_name in EXCLUDE_FILES) or (ofs_name_alternative in EXCLUDE_FILES):
        return None
    else:
        return tarinfo


def get_exclude_ofs(target_list, exclude_list):
    """ The function returns an array of object paths that fall under the exception.

    """

    exclude_array = []

    if exclude_list:
        if not isinstance(exclude_list, list):
            exclude_list = [exclude_list]

        for i in exclude_list:
            if i:
                if re.match('.*\*\*\/.*', i):
                    recursive_global = True
                else:
                    recursive_global = False

                if not i.startswith('/'):
                    for j in target_list:
                        i = os.path.join(j, i)
                        exclude_array.extend(get_ofs(i, recursive_global))
                else:
                    exclude_array.extend(get_ofs(i, recursive_global))

    return exclude_array


def get_ofs(glob_wildcards, recursive=False):
    """ The function returns an array of object paths that correspond to regulars
    in the glob format. The input receives an array of these regulars.

    """

    array_parsed_files = []

    if not isinstance(glob_wildcards, list):
        glob_wildcards = [glob_wildcards]

    for i in glob_wildcards:
        if recursive:
            for filename in glob.glob(i, recursive=True):
                array_parsed_files.append(filename)
        else:
            for filename in glob.glob(i):
                array_parsed_files.append(filename)

    return array_parsed_files


def get_name_files_backup(regex, target):
    """ A function that returns a basis for the name of the object's backup target
    (actual for backup types desc_files, inc_files). The input receives the following arguments:
        regex - regular in the configuration file (which should be backed up);
        target - the specific path on the server that falls under this regular.
    """

    part_of_regex = regex.split('/')
    part_of_target = target.split('/')

    # Since the lengths of the part_of_regex and part_of_target lists are always equal
    # then the length of which of the arrays calculate
    range_count = len(part_of_regex)

    array_of_part_path = []

    for i in range(range_count):
        if part_of_regex[i] != part_of_target[i]:
            array_of_part_path.append(part_of_target[i])

    name = '___'.join(array_of_part_path)

    if not name:
        # If the path in the regular was set explicitly
        if part_of_target[-1]:
            name = part_of_target[-1]
        else:
            name = part_of_target[-2]

    return name


def create_tar(job_type, backup_full_path, target, gzip, backup_type, job_name,
               remote_dir='', storage='', host='', share=''):
    """ The function creates a tarball. The input receives the following arguments:
      job_type - files / databases (necessary for the correct operation of tar exceptions for files);
      backup_full_path - the path to the archive file;
      target - the object to be archived;
      job_name is the name of the section. Required only for the operation of the logging system;
      gzip - True / False;
      remote_dir, storage, host, share - are needed for logging when creating a full backup for incremental backups.

    """

    try:
        if gzip:
            out_tarfile = tarfile.open(backup_full_path, mode='w:gz')
        else:
            out_tarfile = tarfile.open(backup_full_path, mode='w:')

        if job_type == 'files':
            try:
                out_tarfile.add(target, filter=filter_function)
            except FileNotFoundError:
                pass

        elif job_type == 'databases':
            out_tarfile.add(target)

        out_tarfile.close()
    except tarfile.TarError as err:
        if backup_type == 'inc_files':
            dirs_for_log = general_function.get_dirs_for_log(os.path.dirname(backup_full_path),
                                                             remote_dir, storage)
            file_for_log = os.path.join(dirs_for_log, os.path.basename(backup_full_path))

            if storage == 'local':
                str_message = f"Can't create full-backup '{file_for_log}' on '{storage}' storage: {err}"
            elif storage == 'smb':
                str_message = f"Can't create full-backup '{file_for_log}' in '{share}' share on '{storage}' storage({host}): {err}"
            else:
                str_message = f"Can't create full-backup '{file_for_log}' on '{storage}' storage({host}): {err}"
        else:
            str_message = f"Can't create archive '{backup_full_path}' in tmp directory:{err}"

        log_and_mail.writelog('ERROR', str_message, config.filelog_fd, job_name)
        return False
    else:
        if backup_type == 'inc_files':
            dirs_for_log = general_function.get_dirs_for_log(os.path.dirname(backup_full_path),
                                                             remote_dir, storage)
            file_for_log = os.path.join(dirs_for_log, os.path.basename(backup_full_path))

            if storage == 'local':
                str_message = f"Successfully created full-backup '{file_for_log}' on '{storage}' storage."
            elif storage == 'smb':
                str_message = f"Successfully created full-backup '{file_for_log}' in '{share}' share on '{storage}' storage({host})."
            else:
                str_message = f"Successfully created full-backup '{file_for_log}' on '{storage}' storage({host})."
        else:
            str_message = f"Successfully created '{backup_full_path}' file in tmp directory."

        log_and_mail.writelog('INFO', str_message, config.filelog_fd, job_name)
        return True


def gzip_file(origfile, dstfile):
    """ The function compresses the file (used in redis backup).

    """

    gzip_tarfile = dstfile

    try:
        with open(origfile, 'rb') as f_in:
            with gzip.open(gzip_tarfile, 'wb') as f_out:
                shutil.copyfileobj(f_in, f_out)
    except Exception as e:
        raise general_function.MyError(str(e))


def is_excluded_ofs(ofs):
    """ The function determines whether the FSO (file system object) falls under
    the exception rules or not. It is necessary to exclude the creation of an empty backup.

    """

    alternative_name = None

    if os.path.isdir(ofs):
        if not ofs.endswith('/'):
            alternative_name = f'{ofs}/'
        else:
            alternative_name = ofs[:-1]

    if not ((ofs in EXCLUDE_FILES) or (alternative_name in EXCLUDE_FILES)):
        for i in EXCLUDE_FILES:
            if ofs.find(i) == 0 or alternative_name.find(i) == 0:
                return True
            else:
                continue

        return False
    else:
        return True
