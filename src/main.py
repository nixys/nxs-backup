#! /usr/bin/env python3
# -*- coding: utf-8 -*-

import argparse
import sys
import traceback

import config
import desc_files_backup
import external_backup
import general_function
import generate_config
import inc_files_backup
import log_and_mail
import mongodb_backup
import mysql_backup
import mysql_xtrabackup
import postgresql_backup
import postgresql_basebackup
import redis_backup
import resource_constraint
import specific_function

try:
    import version
except ImportError as err:
    general_function.print_info(f"Can't get version from file version.py: {err}")
    VERSION = 'unknown'
else:
    VERSION = ''


def do_backup(path_to_config, jobs_name):
    try:
        parsed_string = specific_function.get_parsed_string(path_to_config)
    except general_function.MyError as e:
        general_function.print_info(f"An error in the parse of the configuration file {path_to_config}:{e}!")
        sys.exit(1)

    db_jobs_dict, file_jobs_dict, external_jobs_dict = config.get_conf_value(parsed_string)

    resource_constraint.set_limitations()

    general_function.create_files('', config.log_file)

    if jobs_name not in config.all_jobs_name:
        general_function.print_info(f"Only one of this job's name is allowed: {config.general_str}")
        sys.exit(1)

    try:
        config.filelog_fd = open(config.log_file, 'a')
    except OSError:  # e.g. /dev/stdout
        try:
            config.filelog_fd = open(config.log_file, 'w')
        except (OSError, PermissionError, FileNotFoundError) as e:
            messange_info = f"Couldn't open file {config.log_file}:{e}!"
            general_function.print_info(messange_info)
            log_and_mail.send_report(messange_info)
            sys.exit(1)
    except (PermissionError, FileNotFoundError) as e:
        messange_info = f"Couldn't open file {config.log_file}:{e}!"
        general_function.print_info(messange_info)
        log_and_mail.send_report(messange_info)
        sys.exit(1)

    try:
        general_function.get_lock()
    except general_function.MyError as ex:
        msg = ex.message
        log_and_mail.writelog('ERROR', f"{msg}", config.filelog_fd, '')
        general_function.print_info(f"{msg}")
        sys.exit(1)

    log_and_mail.writelog('INFO', "Starting script.\n", config.filelog_fd)

    if jobs_name == 'all':
        log_and_mail.writelog('INFO', "Starting files block backup.", config.filelog_fd)
        for i in list(file_jobs_dict.keys()):
            current_jobs_name = file_jobs_dict[i]['job']
            execute_job(current_jobs_name, file_jobs_dict[i])
        log_and_mail.writelog('INFO', "Finishing files block backup.", config.filelog_fd)

        log_and_mail.writelog('INFO', "Starting databases block backup.", config.filelog_fd)
        for i in list(db_jobs_dict.keys()):
            current_jobs_name = db_jobs_dict[i]['job']
            execute_job(current_jobs_name, db_jobs_dict[i])
        log_and_mail.writelog('INFO', "Finishing databases block backup.\n", config.filelog_fd)

        log_and_mail.writelog('INFO', "Starting external block backup.", config.filelog_fd)
        for i in list(external_jobs_dict.keys()):
            current_jobs_name = external_jobs_dict[i]['job']
            execute_job(current_jobs_name, external_jobs_dict[i])
        log_and_mail.writelog('INFO', "Finishing external block backup.\n", config.filelog_fd)
    elif jobs_name == 'databases':
        log_and_mail.writelog('INFO', "Starting databases block backup.", config.filelog_fd)
        for i in list(db_jobs_dict.keys()):
            current_jobs_name = db_jobs_dict[i]['job']
            execute_job(current_jobs_name, db_jobs_dict[i])
        log_and_mail.writelog('INFO', "Finishing databases block backup.\n", config.filelog_fd)
    elif jobs_name == 'files':
        log_and_mail.writelog('INFO', "Starting files block backup.", config.filelog_fd)
        for i in list(file_jobs_dict.keys()):
            current_jobs_name = file_jobs_dict[i]['job']
            execute_job(current_jobs_name, file_jobs_dict[i])
        log_and_mail.writelog('INFO', "Finishing files block backup.\n", config.filelog_fd)
    elif jobs_name == 'external':
        log_and_mail.writelog('INFO', "Starting external block backup.", config.filelog_fd)
        for i in list(external_jobs_dict.keys()):
            current_jobs_name = external_jobs_dict[i]['job']
            execute_job(current_jobs_name, external_jobs_dict[i])
        log_and_mail.writelog('INFO', "Finishing external block backup.\n", config.filelog_fd)
    else:
        if jobs_name in list(db_jobs_dict.keys()):
            log_and_mail.writelog('INFO', "Starting databases block backup.", config.filelog_fd)
            execute_job(jobs_name, db_jobs_dict[jobs_name])
            log_and_mail.writelog('INFO', "Finishing databases block backup.\n", config.filelog_fd)
        elif jobs_name in list(file_jobs_dict.keys()):
            log_and_mail.writelog('INFO', "Starting files block backup.", config.filelog_fd)
            execute_job(jobs_name, file_jobs_dict[jobs_name])
            log_and_mail.writelog('INFO', "Finishing files block backup.\n", config.filelog_fd)
        else:
            log_and_mail.writelog('INFO', "Starting external block backup.", config.filelog_fd)
            execute_job(jobs_name, external_jobs_dict[jobs_name])
            log_and_mail.writelog('INFO', "Finishing external block backup.\n", config.filelog_fd)

    log_and_mail.writelog('INFO', "Stopping script.", config.filelog_fd)


def execute_job(job_name, job_data):
    """ The function makes a backup of a particular job.
    The input receives a dictionary with data of this job.

    """

    log_and_mail.writelog('INFO', f"Starting backup for job '{job_name}'.", config.filelog_fd, job_name)

    if not specific_function.validation_storage_data(job_data):
        return 1

    backup_type = job_data['type']

    if backup_type == 'mysql':
        mysql_backup.mysql_backup(job_data)

    elif backup_type == 'mysql_xtrabackup':
        mysql_xtrabackup.mysql_xtrabackup(job_data)

    elif backup_type == 'postgresql':
        postgresql_backup.postgresql_backup(job_data)

    elif backup_type == 'postgresql_basebackup':
        postgresql_basebackup.postgresql_basebackup(job_data)

    elif backup_type == 'mongodb':
        mongodb_backup.mongodb_backup(job_data)

    elif backup_type == 'redis':
        redis_backup.redis_backup(job_data)

    elif backup_type == 'desc_files':
        desc_files_backup.desc_files_backup(job_data)

    elif backup_type == 'inc_files':
        inc_files_backup.inc_files_backup(job_data)

    else:
        external_backup.external_backup(job_data)

    log_and_mail.writelog('INFO', f"Finishing backup for job '{job_name}'.", config.filelog_fd, job_name)

    return 0


def test_config(path_to_config):
    try:
        specific_function.get_parsed_string(path_to_config)
    except general_function.MyError as e:
        general_function.print_info(f"The configuration file '{path_to_config}' syntax is bad: {e}!")
    else:
        general_function.print_info(f"The configuration file '{path_to_config}' syntax is ok!")
    finally:
        sys.exit()


def get_parser():
    global VERSION
    if not VERSION:
        try:
            VERSION = version.VERSION
        except AttributeError as err:
            general_function.print_info(f'Can\'t get version from file version.py: {err}')
            VERSION = 'unknown'

    # Parent parsers
    version_parser = argparse.ArgumentParser(add_help=False)
    version_parser.add_argument('-v', '--version', action='version', version=VERSION)

    config_parser = argparse.ArgumentParser(add_help=False)
    config_parser.add_argument('-c', '--config', dest='path_to_config', type=str,
                               action='store', help='path to config',
                               default=r'/etc/nxs-backup/nxs-backup.conf')

    # Main parser
    command_parser = argparse.ArgumentParser(parents=[config_parser, version_parser],
                                             description='Make to backups with %(prog)s',
                                             usage='%(prog)s [arguments]')
    # Optional argument
    command_parser.add_argument('-t', '--test', dest='test_conf', action='store_true',
                                help="Check the syntax of the configuration file.",
                                )

    # Positional argument
    subparsers = command_parser.add_subparsers(dest='cmd', help='List of commands')

    # Start command
    start_parser = subparsers.add_parser('start', parents=[config_parser],
                                         help='Start backup script for one of the job in config file.')
    start_parser.add_argument('jobs_name', type=str, help='One of the active job\'s name.', nargs='?', default='all')

    # Generate command
    generate_parser = subparsers.add_parser('generate', help='Generate backup\'s config file.')
    generate_parser.add_argument('-T', '--type', dest='backup_type', type=str, help='One of the type backup.',
                                 nargs=1, choices=config.supported_backup_type, required=True)
    generate_parser.add_argument('-S', '--storages', dest='storages', type=str, help='One or more storages.',
                                 nargs='+', choices=config.supported_storages, required=True)
    generate_parser.add_argument('-P', '--path', dest='path_to_generate_file', type=str,
                                 help='Path to generate config file.', nargs=1, required=True)

    return command_parser


def main():
    parser = get_parser()
    args = parser.parse_args()

    if args.test_conf:
        test_config(args.path_to_config)
    elif args.cmd == 'start':
        try:
            do_backup(args.path_to_config, args.jobs_name)
        except Exception:
            full_traceback = traceback.format_exc()
            log_and_mail.writelog('ERROR', f"An unexpected error occurred: {full_traceback}", config.filelog_fd)
        finally:
            if config.filelog_fd:
                log_and_mail.send_report()
                config.filelog_fd.close()
            if config.lock_file_fd:
                general_function.get_unlock()
    elif args.cmd == 'generate':
        generate_config.generate(args.backup_type, args.storages, args.path_to_generate_file)
    else:
        parser.print_help()


if __name__ == '__main__':
    main()
