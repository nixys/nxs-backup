package ctx

import (
	"fmt"
	"net/mail"

	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"

	"nxs-backup/interfaces"
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

func notifiersInit(conf confOpts) ([]interfaces.Notifier, error) {
	var errs *multierror.Error
	var ns []interfaces.Notifier

	if conf.Notifications.Mail.Enabled {
		var mailErrs *multierror.Error
		mailList := conf.Notifications.Mail.Recipients
		mailList = append(mailList, conf.Notifications.Mail.From)
		for _, a := range mailList {
			_, err := mail.ParseAddress(a)
			if err != nil {
				mailErrs = multierror.Append(mailErrs, fmt.Errorf("Email init fail. Failed to parse email \"%s\". %s ", a, err))
			}
		}

		ml, ok := messageLevels[conf.Notifications.Mail.MessageLevel]
		if ok {
			if mailErrs != nil {
				errs = multierror.Append(errs, mailErrs.Errors...)
			} else {
				m, err := notifier.MailerInit(notifier.MailOpts{
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
				if err != nil {
					errs = multierror.Append(errs, err)
				} else {
					ns = append(ns, m)
				}
			}
		} else {
			errs = multierror.Append(fmt.Errorf("Email init fail. Unknown message level. Available levels: 'INFO', 'WARN', 'ERR' "))
		}
	}

	for _, wh := range conf.Notifications.Webhooks {
		if wh.Enabled {
			ml, ok := messageLevels[wh.MessageLevel]
			if ok {
				a, err := notifier.WebhookInit(notifier.WebhookOpts{
					WebhookURL:        wh.WebhookURL,
					InsecureTLS:       wh.InsecureTLS,
					ExtraHeaders:      wh.ExtraHeaders,
					PayloadMessageKey: wh.PayloadMessageKey,
					ExtraPayload:      wh.ExtraPayload,
					MessageLevel:      ml,
					ProjectName:       conf.ProjectName,
					ServerName:        conf.ServerName,
				})
				if err != nil {
					errs = multierror.Append(errs, err)
				} else {
					ns = append(ns, a)
				}
			} else {
				errs = multierror.Append(errs, fmt.Errorf("Webhook init fail. Unknown message level. Available levels: 'INFO', 'WARN', 'ERR' "))
			}
		}
	}

	return ns, errs.ErrorOrNil()
}
