package ctx

import (
	"fmt"
	"net/mail"

	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"

	"nxs-backup/modules/backend/notifier"
)

var messageLevels = map[string]logrus.Level{
	"err":     logrus.ErrorLevel,
	"Err":     logrus.ErrorLevel,
	"ERR":     logrus.ErrorLevel,
	"error":   logrus.ErrorLevel,
	"Error":   logrus.ErrorLevel,
	"ERROR":   logrus.ErrorLevel,
	"warn":    logrus.WarnLevel,
	"Warn":    logrus.WarnLevel,
	"WARN":    logrus.WarnLevel,
	"warning": logrus.WarnLevel,
	"Warning": logrus.WarnLevel,
	"WARNING": logrus.WarnLevel,
	"inf":     logrus.InfoLevel,
	"Inf":     logrus.InfoLevel,
	"INF":     logrus.InfoLevel,
	"info":    logrus.InfoLevel,
	"Info":    logrus.InfoLevel,
	"INFO":    logrus.InfoLevel,
}

func mailerInit(conf confOpts) (m notifier.Mailer, err error) {
	var errs *multierror.Error

	if !conf.Notifications.Mail.Enabled {
		return
	}

	mailList := conf.Notifications.Mail.Recipients
	mailList = append(mailList, conf.Notifications.Mail.From)
	for _, a := range mailList {
		_, err = mail.ParseAddress(a)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("  failed to parse email \"%s\". %s", a, err))
		}
	}

	ml, ok := messageLevels[conf.Notifications.Mail.MessageLevel]
	if !ok {
		errs = multierror.Append(fmt.Errorf("Unknown Mail message level. Available levels: 'INFO', 'WARN', 'ERR' "))
	}
	if errs != nil {
		err = errs
		return
	}

	m, err = notifier.MailerInit(notifier.MailOpts{
		Enabled:      conf.Notifications.Mail.Enabled,
		From:         conf.Notifications.Mail.From,
		SmtpServer:   conf.Notifications.Mail.SmtpServer,
		SmtpPort:     conf.Notifications.Mail.SmtpPort,
		SmtpUser:     conf.Notifications.Mail.SmtpUser,
		SmtpPassword: conf.Notifications.Mail.SmtpPassword,
		Recipients:   conf.Notifications.Mail.Recipients,
		MessageLevel: ml,
		ProjectName:  conf.ProjectName,
		ServerName:   conf.ServerName,
	})

	return
}

func alerterInit(conf confOpts) (a notifier.AlertServer, err error) {

	if !conf.Notifications.NxsAlert.Enabled {
		return
	}

	ml, ok := messageLevels[conf.Notifications.NxsAlert.MessageLevel]
	if !ok {
		err = fmt.Errorf("Unknown Mail message level. Available levels: 'INFO', 'WARN', 'ERR' ")
		return
	}

	a, err = notifier.AlertServerInit(notifier.AlertServerOpts{
		Enabled:      conf.Notifications.NxsAlert.Enabled,
		NxsAlertURL:  conf.Notifications.NxsAlert.NxsAlertURL,
		AuthKey:      conf.Notifications.NxsAlert.AuthKey,
		InsecureTLS:  conf.Notifications.NxsAlert.InsecureTLS,
		MessageLevel: ml,
		ProjectName:  conf.ProjectName,
		ServerName:   conf.ServerName,
	})

	return
}
