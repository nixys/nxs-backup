#! /usr/bin/env python3
# -*- coding: utf-8 -*-

import os.path
import re

import config
import general_function
import log_and_mail


def set_limitations():
    nice = 19
    ionice = True

    general_function.set_prio_process(nice, ionice)

    if config.block_io_write and config.block_io_read:
        set_cgroup('blkio', 'blkio.throttle.write_bps_device', 'blkio.throttle.read_bps_device')

    if config.block_io_weight:
        set_cgroup('blkio', 'blkio.weight_device')

    if config.cpu_shares:
        set_cgroup('cpu', 'cpu.shares')


def set_cgroup(group, *args):
    pid = os.getpid()

    data_1 = general_function.exec_cmd(f"cat /proc/cgroups | grep {group}")
    stdout_1 = data_1['stdout']
    if not stdout_1:
        log_and_mail.writelog('WARNING', f"Your kernel doesn't support cgroup '{group}'.",
                              config.filelog_fd)
        return False

    data_2 = general_function.exec_cmd('mount | grep "/sys/fs/cgroup"')
    stdout_2 = data_2['stdout']

    if not stdout_2:
        general_function.exec_cmd(
            'mount -t tmpfs -o rw,nosuid,nodev,noexec,relatime,size=0k cgroup_root /sys/fs/cgroup/')
    _dir = f'/sys/fs/cgroup/{group}'

    data_3 = general_function.exec_cmd(f'mount | grep "{_dir}"')
    stdout_3 = data_3['stdout']

    if not (os.path.isdir(_dir) or not stdout_3):
        general_function.create_dirs(job_name='', dirs_pairs={_dir: ''})
        general_function.exec_cmd(f'mount -t cgroup -o rw,nosuid,nodev,noexec,relatime,{group} cgroup_{group} {_dir}/')

    general_function.create_dirs(job_name='', dirs_pairs={f'{_dir}/nixys_backup': ''})

    args_list = list(args)
    for index in args_list:
        if not os.path.isfile(os.path.join(_dir, index)):
            log_and_mail.writelog('WARNING', f"Your kernel does not support option '{index}' in subsystem '{group}'.",
                                  config.filelog_fd)
            return False
        _parametr = -100

        if group == 'blkio':
            directory_for_tmp_file = config.general_path_to_all_tmp_dir

            general_function.create_dirs(job_name='', dirs_pairs={directory_for_tmp_file: ''})

            data_4 = general_function.exec_cmd(f"df {directory_for_tmp_file} | tail -1 | awk '{{print $1}}'")
            stdout_4 = data_4['stdout']

            if re.match("/dev/disk/(by-id|by-path|by-uuid)", stdout_4):
                data_5 = general_function.exec_cmd(f"ls -l {stdout_4} | awk '{{print $11}}'")
                stdout_5 = data_5['stdout']
                device = os.path.basename(stdout_5)
            else:
                device = stdout_4

            raid = True
            if not re.match(".*/(md|dm).+", device):
                raid = False
                while re.match("^[0-9]$", str(device[len(device) - 1])):
                    device = device[0:-1]

            data_6 = general_function.exec_cmd(f"ls -l {device} | awk '{{print $5}}'")
            stdout_6 = data_6['stdout']
            major_device = stdout_6[0:-1]

            data_7 = general_function.exec_cmd(f"ls -l {device} | awk '{{print $6}}'")
            stdout_7 = data_7['stdout']
            minor_device = stdout_7

            if index != 'blkio.weight_device':
                if index == 'blkio.throttle.write_bps_device':
                    if not re.match("^([0-9]*)$", config.block_io_write, re.I):
                        log_and_mail.writelog(
                            'WARNING',
                            "Incorrect data in field 'block_io_write'! You must specify the write speed "
                            "in MB/s using only numbers!",
                            config.filelog_fd)
                        return False
                    _parametr = 1024 * 1024 * int(config.block_io_write)
                else:
                    if not re.match("^([0-9]*)$", config.block_io_read, re.I):
                        log_and_mail.writelog(
                            'WARNING',
                            "Incorrect data in field 'block_io_read'! You must specify the read speed "
                            "in MB/s using only numbers!",
                            config.filelog_fd)
                        return False
                    _parametr = 1024 * 1024 * int(config.block_io_read)

            else:
                if not raid:
                    if not (re.match("^([0-9]*)$", config.block_io_weight, re.I) and
                            100 <= int(config.block_io_weight) <= 1000):
                        log_and_mail.writelog(
                            'WARNING',
                            "Incorrect data in field 'blkio_weight'! Process must specify weight "
                            "in the range from 100 to 1000!",
                            config.filelog_fd)
                        return False
                    _parametr = config.block_io_weight
                else:
                    log_and_mail.writelog(
                        'WARNING', "You can not use option 'blkio.weight_device' with the raid!",
                        config.filelog_fd)
                    return False

            general_function.exec_cmd(f'echo {major_device}:{minor_device} {_parametr} > {_dir}/nixys_backup/{index}')

            data_8 = general_function.exec_cmd(f"cat {_dir}/nixys_backup/{index}")
            stdout_8 = data_8['stdout']
            _flag = stdout_8

            if len(_flag) < 3:
                log_and_mail.writelog('WARNING', f"Incorrect data in file '{_dir}/nixys_backup/{index}'!",
                                      config.filelog_fd)
                return False

        if group == 'cpu':
            if index == 'cpu.shares':
                if not re.match("^([0-9]*)$", config.cpu_shares, re.I):
                    log_and_mail.writelog(
                        'WARNING',
                        "Incorrect data in field 'cpu_shares'! You must specify  weight "
                        "in the range from 1 to cpu_count*1000!",
                        config.filelog_fd)
                    return False

                _parametr = int(config.cpu_shares)

                general_function.exec_cmd(f"echo {_parametr} > {_dir}/nixys_backup/{index}")

                data_9 = general_function.exec_cmd(f"cat {_dir}/nixys_backup/{index}")
                stdout_9 = data_9['stdout']
                _flag = stdout_9

                if not _flag:
                    log_and_mail.writelog('WARNING', f"Incorrect data in file '{_dir}/nixys_backup/{index}'!",
                                          config.filelog_fd)
                    return False

    general_function.exec_cmd(f"echo {pid} > {_dir}/nixys_backup/tasks")
    return True
