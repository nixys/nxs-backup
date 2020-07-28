#! /usr/bin/env python3
# -*- coding: utf-8 -*-

import smtplib
import subprocess
from email.mime.multipart import MIMEMultipart
from email.mime.text import MIMEText

import config
import general_function


def send_report(*message):
    """ A function that sends a report about the operation of the backup tools to
    administrators and clients.

    """

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
    """ The function sends a letter. The input receives the following arguments:
     sender - the sender's mailing address;
     recipient_admin - postal technical address of project administrators;
     recipient_client - postal addresses of recipients (clients);
     server_name - the name of the server in our system of tasks;
     body is the body of the letter.

    """

    recipient_client.append(recipient_admin)
    itog_mail_addr = []

    for i in range(len(recipient_client)):
        q = recipient_client[i]
        if q:
            itog_mail_addr.append(q)

    if config.smtp_server:
        msg = MIMEMultipart()
    else:
        msg = MIMEText(body, "", "utf-8")

    msg['To'] = ','.join(itog_mail_addr)
    msg['Subject'] = f'{server_name} notification dump.'

    if config.smtp_server:
        msg['From'] = config.smtp_user if config.smtp_user and '@' in config.smtp_user else sender
        msg.attach(MIMEText(body))

        try:
            if config.smtp_ssl:
                smtp = smtplib.SMTP_SSL(config.smtp_server, port=config.smtp_port if config.smtp_port else 465,
                                        timeout=config.smtp_timeout)
            else:
                smtp = smtplib.SMTP(config.smtp_server, port=config.smtp_port if config.smtp_port else 25,
                                    timeout=config.smtp_timeout)
            if config.smtp_tls:
                smtp.starttls()
            if config.smtp_user and config.smtp_password:
                smtp.login(config.smtp_user, config.smtp_password)

            smtp.sendmail(msg['From'], itog_mail_addr, msg.as_string())
            smtp.close()
        except Exception as e:
            writelog('ERROR', f"Some problem when sending a message via {config.smtp_server}: {e}",
                     config.filelog_fd)
    else:
        msg['From'] = sender

        try:
            p = subprocess.Popen(["/usr/sbin/sendmail -t -oi"], stdin=subprocess.PIPE, shell=True)
            p.communicate(msg.as_bytes())
        except Exception as e:
            writelog('ERROR', f"Some problem when sending a message via /usr/bin/sendmail: {e}",
                     config.filelog_fd)


def get_log(log_level, log_message, type_message=''):
    """ The function of forming a string for writing to a log file.
    The input is given the following values:
     log_level - event level (error, info, warning);
     log_message - message;
     type_message is the section in the configuration file to which the event belongs.

    """

    time_now = general_function.get_time_now('log')

    if type_message:
        result_str = f"{log_level} [{type_message}] [{time_now}]: {log_message}\n"
    else:
        result_str = f"{log_level} [{time_now}]: {log_message}\n"

    return result_str


def writelog(log_level, log_message, fd, type_message=''):
    """ The function of recording events in the log file. The input is given the following values:
     log_level - event level (error, info, warning);
     log_message - message;
     fd - file descriptor number of the log file;
     type_message is the section in the configuration file to which the event belongs.

    """

    log_str = get_log(log_level, log_message, type_message)

    try:
        fd.write(log_str)
        fd.flush()
    except (OSError, PermissionError, FileNotFoundError) as err:
        messange_info = f"Couldn't write to log file:{err}"
        general_function.print_info(messange_info)

    if log_level == 'ERROR':
        config.error_log += log_str
        config.debug_log += log_str
    else:
        config.debug_log += log_str
