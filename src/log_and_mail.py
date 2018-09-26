#! /usr/bin/env python3
# -*- coding: utf-8 -*-

import subprocess
from email.mime.text import MIMEText

import config
import general_function


def send_report(*message):
    ''' A function that sends a report about the operation of the backup tools to
    administrators and clients.

    '''

    local_message = ''.join(message)

    if local_message:
        send_mail(config.mail_from, config.admin_mail, [], config.server_name, local_message)
        return 1

    if config.level_message == 'debug':
        if config.error_log:
            send_mail(config.mail_from, config.admin_mail, [], config.server_name, config.error_log)

        send_mail(config.mail_from, '', config.client_mail, config.server_name, config.debug_log)
    else:
        if config.error_log:
            send_mail(config.mail_from, config.admin_mail, config.client_mail, config.server_name, config.error_log)


def send_mail(sender, recipient_admin, recipient_client, server_name, body):
    ''' The function sends a letter. The input receives the following arguments:
     sender - the sender's mailing address;
     recipient_admin - postal technical address of project administrators;
     recipient_client - postal addresses of recipients (clients);
     server_name - the name of the server in our system of tasks;
     body is the body of the letter.

    '''

    recipient_client.append(recipient_admin)
    itog_mail_addr = []

    for i in range(len(recipient_client)):
        q = recipient_client[i]
        if q:
            itog_mail_addr.append(q)

    msg = MIMEText(body, "", "utf-8")
    msg['Subject'] = '%s notification dump.' %(server_name)
    msg['From'] = sender
    msg['To'] = ','.join(itog_mail_addr)

    try:
        p = subprocess.Popen(["/usr/sbin/sendmail -t -oi"], stdin=subprocess.PIPE, shell=True)
        p.communicate(msg.as_bytes())
    except Exception as e:
        writelog('ERROR', "Some problem when sending a message via /usr/bin/sendmail: %s" %e,
                 config.filelog_fd)


def get_log(log_level, log_message, type_message=''):
    ''' The function of forming a string for writing to a log file.
    The input is given the following values:
     log_level - event level (error, info, warning);
     log_message - message;
     type_message is the section in the configuration file to which the event belongs.

    '''

    time_now = general_function.get_time_now('log')

    if type_message:
        result_str = "%s [%s] [%s]: %s\n" %(log_level, type_message, time_now, log_message)
    else:
        result_str = "%s [%s]: %s\n" %(log_level, time_now, log_message)

    return result_str


def writelog(log_level, log_message, fd, type_message=''):
    ''' The function of recording events in the log file. The input is given the following values:
     log_level - event level (error, info, warning);
     log_message - message;
     fd - file descriptor number of the log file;
     type_message is the section in the configuration file to which the event belongs.

    '''

    log_str = get_log(log_level, log_message, type_message)

    try:
        fd.write(log_str)
    except (OSError, PermissionError, FileNotFoundError) as err:
        messange_info = "Couldn't write to log file:%s" %(err)
        general_function.print_info(messange_info)

    if log_level == 'ERROR':
        config.error_log += log_str
        config.debug_log += log_str
    else:
        config.debug_log += log_str
