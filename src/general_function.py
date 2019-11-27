#! /usr/bin/env python3
# -*- coding: utf-8 -*-

import subprocess
import os
import datetime
import shutil
import platform
import psutil
import fcntl
import sys

import config
import log_and_mail
import mount_fuse


class MyError(Exception):
    def __init__(self, message):
        self.message = message
    def __str__(self):
        return self.message


def exec_cmd(cmdline):
    ''' The function accepts a string that it then executes in /bin/bash and
    waits for it to execute. Returns a dictionary with three pairs, like:
        ['stdout'] = $stdout
        ['stderr'] = $stderr
        ['code'] = $code

    '''

    data_dict = {}

    current_process = subprocess.Popen([cmdline], stdout=subprocess.PIPE,
                                       stderr=subprocess.PIPE, shell=True,
                                       executable='/bin/bash')
    data = current_process.communicate()

    data_stdout = data[0][0:-1].decode('utf-8')
    data_stderr = data[1][0:-1].decode('utf-8')

    data_dict['stdout'] = data_stdout
    data_dict['stderr'] = data_stderr
    data_dict['code'] = current_process.returncode

    return data_dict


def print_info(*message):
    ''' Print the message to stdout in special format. 

    '''

    print("{}: {}" .format('nxs-backup',
                           ": ".join(list(str(m) for m in message))),
                           file=sys.stderr)


def get_lock():

    create_files('', config.path_to_lock_file)
    config.lock_file_fd = open(config.path_to_lock_file, 'a')
    fcntl.flock(config.lock_file_fd, fcntl.LOCK_EX | fcntl.LOCK_NB)

    return 1


def get_unlock():

    fcntl.flock(config.lock_file_fd, fcntl.LOCK_UN)

    return 1


def get_time_now(unit):
    ''' The function returns the current time in the required format.

    '''

    now = datetime.datetime.now()

    if unit == "dom":  # Day of the month
        result = now.strftime("%d")
    elif unit == "dow":  # Day of the week
        result = now.strftime("%u")
    elif unit == "moy":  # Month of the year
        result = now.strftime("%m")
    elif unit == "year":
        result = now.strftime("%Y")
    elif unit == "log":  # Full date for logging
        result = now.strftime("%Y-%m-%d %H:%M:%S")
    elif unit == "backup":  # Full date for dump name
        result = now.strftime("%Y-%m-%d_%H-%M")
    return result


def get_dirs_for_log(local_dir, backup_dir, storage=''):
    ''' The function returns a directory for writing to the log.

    '''

    result_dir = backup_dir

    if mount_fuse.mount_point:
        if storage in ('scp', 'nfs'):
            local_part = os.path.relpath(local_dir, mount_fuse.mount_point)
            result_dir = os.path.join(backup_dir, local_part)
        else:
            local_part = os.path.relpath(local_dir, mount_fuse.mount_point)
            result_dir = os.path.join('/', local_part)

    return result_dir


def create_dirs (**kwargs):
    ''' Function for creating directories.

    '''

    job_name = kwargs['job_name']
    dirs_pairs = kwargs['dirs_pairs']  # Dictionary with pairs 'local_dir' = 'remote_dir'

    for i in dirs_pairs:
        if not os.path.exists(i):
            try:
                os.makedirs(i)
            except PermissionError as err:
                if dirs_pairs[i]:  # Means create on the remote storage and the way it is necessary to specify this
                    i = dirs_pairs[i]

                    log_and_mail.writelog('ERROR', f"Can't create directory {i}:{err}!",
                                          config.filelog_fd, job_name)



def create_files(backup_type, *files):
    ''' Function for creating files.

    '''

    for i in list(files):
        create_dirs(job_name=backup_type, dirs_pairs={os.path.dirname(i):''})
        if not (os.path.isfile(i) or os.path.islink(i)):
            try:
                with open(i, 'tw', encoding='utf-8'):
                    pass
            except (PermissionError) as err:
                log_and_mail.writelog('ERROR', f"Can't create file {i}:{err}!",
                                      config.filelog_fd)


def del_file_objects(backup_type, *ofs):
    ''' Removes the object of the FS. 

    '''

    for i in ofs:
        # If you want to delete all objects inside the directory, except for the object itself
        if i.endswith('/*'):
            current_dir = i[:-1]
            if os.path.isdir(current_dir):
                for j in os.listdir(current_dir):
                    full_path = os.path.join(current_dir, j)
                    del_file_objects(backup_type, full_path)
        else:
            try:
                if os.path.isfile(i) or os.path.islink(i):
                    os.unlink(i)
                elif os.path.isdir(i):
                    shutil.rmtree(i)
                #else:  # When ofs does not exist
                    #return 0
            except PermissionError as e:
                raise MyError(e)


def get_dist():
    ''' The function defines the Linux distribution.

    '''

    dist = platform.linux_distribution()[0]
    return dist


def set_prio_process(nice, ionice):
    ''' The function sets the priority of the current script process. 

    '''

    pid = os.getpid()
    p = psutil.Process(pid)
    p.nice(nice)
    if ionice:
        p.ionice(psutil.IOPRIO_CLASS_IDLE)


def get_full_path(path_dir, base_name, base_extension, gzip):
    ''' The function returns the full path to the archive. The input receives the following arguments:
        path_dir - path to the directory with the archive;
        base_name - the key part of the name (for example, the name of the database);
        base_extension - archive extension (.tar, .sql, etc.);
        gzip - True/False.

    '''

    date_now = get_time_now('backup')

    if gzip:
        backup_base_name = f'{base_name}_{date_now}.{base_extension}.gz'
    else:
        backup_base_name = f'{base_name}_{date_now}.{base_extension}'

    full_path = os.path.join(path_dir, backup_base_name)

    return full_path


def get_tmp_dir(tmp_dir, backup_type):
    ''' Returns the full path to the temporary directory to collect the dump.
    The input receives the following arguments:
         tmp_dir - is the main part up to the temporary directory;
         backup_type - type of backup (mysql, postgresql etc).

    '''

    date_now = get_time_now('backup')

    tmp_dir_name = f'{backup_type}_{date_now}'
    full_path_tmp_dir = os.path.join(tmp_dir, tmp_dir_name)
    create_dirs(job_name=backup_type, dirs_pairs={full_path_tmp_dir:''})

    return full_path_tmp_dir


def get_absolute_path(i, root):
    ''' The function returns the absolute path to the object.

    '''

    result = ''

    if i.startswith('/'):
        result = i
    else:
        result = os.path.join(root, i)
    return result


def copy_ofs(src_ofs, dst_ofs):
    ''' The function copies a file system object.

    '''

    try:
        shutil.copy(src_ofs, dst_ofs)
    except (OSError, RuntimeError, IOError, shutil.Error, FileNotFoundError) as err:
        raise MyError(str(err))


def move_ofs(src_ofs, dst_ofs):
    ''' The function move a file system object.

    '''

    try:
        shutil.move(src_ofs, dst_ofs)
    except (OSError, RuntimeError, IOError, shutil.Error, FileNotFoundError) as err:
        raise MyError(str(err))


def create_symlink(src_ofs, dst_ofs):
    ''' The function creates a symlink.

    '''

    try:
        os.symlink(src_ofs, dst_ofs)
    except (OSError, RuntimeError, IOError, shutil.Error, FileNotFoundError) as err:
        raise MyError(str(err))


def get_default_port(type_source):
    ''' The function returns the default port for a specific instance.

    '''

    return config.default_port_dict[type_source]
