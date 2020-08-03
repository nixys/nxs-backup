#! /usr/bin/env python3
# -*- coding: utf-8 -*-

import os
import re
import sys

import config
import general_function

TEMPLATES_DIR = '/usr/share/nxs-backup/templates'


def generate(backup_type, storages, path_to_file):
    """ The function generate a configuration file job.

    """

    backup_type = backup_type[0]
    path_to_file = path_to_file[0]

    template_path = f'{TEMPLATES_DIR}/backup_type/{backup_type}.conf'

    if path_to_file.startswith('/'):
        general_function.create_dirs(job_name=backup_type,
                                     dirs_pairs={os.path.dirname(path_to_file): ''})

    general_function.copy_ofs(template_path, path_to_file)

    try:
        fd = open(path_to_file, 'a')
    except (OSError, PermissionError, FileNotFoundError) as e:
        messange_info = f"Couldn't open file {path_to_file}:{e}!"
        general_function.print_info(messange_info)
        sys.exit(1)

    if backup_type in config.supported_db_backup_type:
        job_type = 'databases'
    elif backup_type in config.supported_file_backup_type:
        job_type = 'files'
    else:
        job_type = 'external'

    for storage in storages:
        storage_template_path = f'{TEMPLATES_DIR}/storages/{storage}.conf'

        with open(storage_template_path, 'r', encoding='utf-8') as f:
            str_storage = f.read()

        str_storage = str_storage.replace('backup_type', backup_type)
        str_storage = str_storage.replace('job_type', job_type)

        if backup_type == 'inc_files':
            str_storage = str_storage.replace('inc_files/dump', 'inc')
            str_storage = re.sub(r"[ ]*store:[\s]*days: ''[\s]*weeks: ''[\s]*month: ''[\s]*",
                                 '', str_storage)

        if backup_type == 'desc_files':
            str_storage = str_storage.replace('desc_files/dump', 'desc/dump')

        if backup_type == 'external':
            str_storage = str_storage.replace('external/dump', 'dump')

        fd.write(str_storage)

    fd.close()

    os.chmod(path_to_file, 0o600)

    general_function.print_info(f"Successfully generated '{path_to_file}' configuration file!")
