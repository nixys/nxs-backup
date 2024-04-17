package ctx

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/nixys/nxs-backup/modules/notifier/mailer"
	"github.com/nixys/nxs-backup/modules/notifier/webhooker"
	"github.com/sirupsen/logrus"
	"net/mail"
	"strings"

	"github.com/nixys/nxs-backup/interfaces"
)

var messageLevels = map[string]logrus.Level{
	"ERR":     logrus.ErrorLevel,
	"ERROR":   logrus.ErrorLevel,
	"WARN":    logrus.WarnLevel,
	"WARNING": logrus.WarnLevel,
	"INF":     logrus.InfoLevel,
	"INFO":    logrus.InfoLevel,
}

func notifiersInit(c *Ctx, conf ConfOpts) error {
	var errs *multierror.Error
	var ns []interfaces.Notifier

	if conf.Notifications.Mail.Enabled {
		var mailErrs *multierror.Error
		mailList := conf.Notifications.Mail.Recipients
		for _, a := range mailList {
			_, err := mail.ParseAddress(a)
			if err != nil {
				mailErrs = multierror.Append(mailErrs, fmt.Errorf("Email init fail. Failed to parse email \"%s\". %v ", a, err))
			}
		}
		if _, err := mail.ParseAddress(conf.Notifications.Mail.From); err != nil {
			mailErrs = multierror.Append(mailErrs, fmt.Errorf("Email init fail. Failed to parse `mail_from` \"%s\". %v ", conf.Notifications.Mail.From, err))
		}

		ml, ok := messageLevels[strings.ToUpper(conf.Notifications.Mail.MessageLevel)]
		if ok {
			if mailErrs != nil {
				errs = multierror.Append(errs, mailErrs.Errors...)
			} else {
				m, err := mailer.Init(mailer.Opts{
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
			ml, ok := messageLevels[strings.ToUpper(wh.MessageLevel)]
			if ok {
				a, err := webhooker.Init(webhooker.Opts{
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

	c.Notifiers = ns

	return errs.ErrorOrNil()
}
