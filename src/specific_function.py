#! /usr/bin/env python3
# -*- coding: utf-8 -*-

import yaml
import json
import os.path
import sys

import config
import log_and_mail
import general_function
import general_files_func


class Loader(yaml.Loader):
    def __init__(self, stream):
        self._root = os.path.split(stream.name)[0]
        super(Loader, self).__init__(stream)
        Loader.add_constructor('!include', Loader.include)
        Loader.add_constructor('!import',  Loader.include)

    def include(self, node):
        if isinstance(node, yaml.ScalarNode):
            return self.extractFile(self.construct_scalar(node))

        elif isinstance(node, yaml.SequenceNode):
            result = []

            for i in self.construct_sequence(node):
                i = general_function.get_absolute_path(i, self._root)
                for j in general_files_func.get_ofs(i):
                    result += self.extractFile(j)
            return result

        elif isinstance(node, yaml.MappingNode):
            result = {}
            for k,v in self.construct_mapping(node).iteritems():
                result[k] = self.extractFile(v)
            return result

        else:
            print ('Error:: unrecognised node type in !include statement')
            raise yaml.constructor.ConstructorError

    def extractFile(self, filename):
        filepath = os.path.join(self._root, filename)
        with open(filepath, 'r') as f:
            return yaml.load(f, Loader)


def is_save_to_storage(job_name, storage_data):
    ''' Checks the need for collection in a SPECIFIC storage.

    '''

    try:
        storage = storage_data['storage']
        enable_storage = storage_data['enable']
        backup_dir = storage_data['backup_dir']

        if not storage in config.supported_storages:
            log_and_mail.writelog('ERROR', "For '%s' job set incorrect type of storage." %(job_name) +\
                                  "Only one of this type storage is allowed:%s" %(config.supported_storages), 
                                  config.filelog_fd, job_name)
            result = False

        elif not enable_storage:
            result = False
        elif not backup_dir:
            log_and_mail.writelog('ERROR', "Field 'backup_dir' in job '%s' for storage '%s' can't be empty!" %(job_name, storage_data['storage']),
                                      config.filelog_fd, job_name)
            result = False
        else:
            result = True
    except KeyError as err:
            log_and_mail.writelog('ERROR', "Missing required key '%s' in '%s' job storages block." %(err, job_name),
                                  config.filelog_fd, job_name)
            result = False

    return result


def validation_storage_data(job_data):
    ''' The function checks that in the job there is at least one active storage
    according to the schedule of which, it is necessary to collect a backup.

    '''

    result = True
    job_name = job_data['job']

    flag = False
    for storage in range(len(job_data['storages'])):
        if job_data['storages'][storage]['enable']:
            flag = True
            break

    if not flag:
        log_and_mail.writelog('ERROR', 'There are no active storages in the job %s!' %job_name,
                              config.filelog_fd, job_name)
        result = False
    else:
        if not is_time_to_backup(job_data):
            log_and_mail.writelog('INFO', "According to the backup plan today new backups are not created in this job.",
                                  config.filelog_fd, job_name)
            result = False

    return result


def is_time_to_backup(job_data):
    ''''' Фукнция, которая определяет необходимо ли запускать на выполнение сбор копий согласно плану.
    На вход получает словарь с данными для конкретной секции

    '''''

    job_name = job_data['job']
    job_type = job_data['type']
    storages = job_data['storages']

    if job_type == 'inc_files':
        return True

    dow = general_function.get_time_now("dow")
    dom = general_function.get_time_now("dom")

    day_flag = False
    week_flag = False
    month_flag = False

    for i in range(len(storages)):
        if storages[i]['enable']:
            if storages[i]['store']['days'] or storages[i]['store']['weeks'] or storages[i]['store']['month']:
                if int(storages[i]['store']['days']) > 0:
                    day_flag = True
                if int(storages[i]['store']['weeks']) > 0:
                    week_flag = True
                if int(storages[i]['store']['month']) > 0:
                    month_flag = True
            else:
                log_and_mail.writelog('ERROR', 'There are no stores data for storage %s in the job %s!' %(job_type, job_name),
                                      config.filelog_fd, job_name)
                continue
    if not day_flag:
        if not week_flag:
            if not month_flag:
                result = False
            else:
                if dom == config.dom_backup:
                    result = True
                else:
                    result = False
        else:
            if dow == config.dow_backup:
                result = True
            else:
                if not month_flag:
                    result = False
                else:
                    if dom == config.dom_backup:
                        result = True
    else:
        result = True

    return result


def get_parsed_string(path_to_config):

    try:
        with open(path_to_config, 'r') as stream:
            try:
                yaml_str = yaml.load(stream, Loader=Loader)
            except yaml.YAMLError as e:
                raise general_function.MyError(str(e))
            except (RuntimeError) as e:
                if "maximum recursion depth exceeded while calling" in str(e):
                    error_msg = " error in include value - '%s'" %(e)
                else:
                    error_msg = str (e)
                raise general_function.MyError(error_msg)
    except (FileNotFoundError, PermissionError) as e:
        general_function.print_info("No such file '%s' or permission denied!" %(path_to_config))
        sys.exit(1)
    else:
        return yaml_str


def parser_json(json_file):

    try:
        parsed_str = json.load(open(json_file))
    except (PermissionError, OSError) as e:
        raise general_function.MyError(str(e))
    else:
        return parsed_str
