package notifier

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"sync"

	appctx "github.com/nixys/nxs-go-appctx/v2"
	"github.com/sirupsen/logrus"
	"gopkg.in/gomail.v2"

	"nxs-backup/modules/logger"
)

type MailOpts struct {
	Enabled      bool
	From         string
	SmtpServer   string
	SmtpPort     int
	SmtpUser     string
	SmtpPassword string
	SmtpTimeout  string
	Recipients   []string
	MessageLevel logrus.Level
	ProjectName  string
	ServerName   string
}

type Mailer struct {
	opts MailOpts
}

func MailerInit(mailCfg MailOpts) (Mailer, error) {
	m := Mailer{opts: mailCfg}

	if !mailCfg.Enabled {
		return m, nil
	}

	if mailCfg.SmtpServer != "" {
		d := gomail.NewDialer(mailCfg.SmtpServer, mailCfg.SmtpPort, mailCfg.SmtpUser, mailCfg.SmtpPassword)
		sc, err := d.Dial()
		if err != nil {
			return Mailer{}, fmt.Errorf("Failed to dial SMTP server. Error: %v ", err)
		}
		defer func() { _ = sc.Close() }()
	}

	return m, nil
}

// Send sends notification via Email
func (m *Mailer) Send(appCtx *appctx.AppContext, n logger.LogRecord, wg *sync.WaitGroup) {
	if !m.opts.Enabled || n.Level > m.opts.MessageLevel {
		return
	}

	wg.Add(1)
	defer wg.Done()

	var (
		sc  gomail.SendCloser
		err error
	)
	defer func() { _ = sc.Close() }()

	msg := gomail.NewMessage()
	msg.SetHeader("From", m.opts.From)
	msg.SetHeader("To", m.opts.Recipients...)

	subjStr := fmt.Sprintf("[%s] Nxs-backup notification: server %q", n.Level, m.opts.ServerName)
	if m.opts.ProjectName != "" {
		subjStr += fmt.Sprintf(" of project %q", m.opts.ProjectName)
	}
	msg.SetHeader("Subject", subjStr)

	msg.SetBody("text/html", m.getMailBody(n))

	if m.opts.SmtpServer != "" {
		d := gomail.NewDialer(m.opts.SmtpServer, m.opts.SmtpPort, m.opts.SmtpUser, m.opts.SmtpPassword)
		sc, err = d.Dial()
		if err != nil {
			appCtx.Log().Errorf("Failed to dial SMTP server. Error: %v", err)
			return
		}
	} else {
		sc = localMail{}
	}

	if err = gomail.Send(sc, msg); err != nil {
		appCtx.Log().Errorf("Could not send email: %v", err)
	}
}

func (m *Mailer) getMailBody(n logger.LogRecord) (b string) {
	switch n.Level {
	case logrus.DebugLevel:
		b += "[DEBUG]:\n\n"
	case logrus.InfoLevel:
		b += "[INFO]:\n\n"
	case logrus.WarnLevel:
		b += "[WARNING]:\n\n"
	case logrus.ErrorLevel:
		b += "[ERROR]:\n\n"
	}

	if m.opts.ProjectName != "" {
		b += fmt.Sprintf("Project: %s\n", m.opts.ProjectName)
	}
	if m.opts.ServerName != "" {
		b += fmt.Sprintf("Server: %s\n\n", m.opts.ServerName)
	}

	if n.JobName != "" {
		b += fmt.Sprintf("Job: %s\n", n.JobName)
	}
	if n.StorageName != "" {
		b += fmt.Sprintf("Storage: %s\n", n.StorageName)
	}
	b += fmt.Sprintf("Message: %s\n", n.Message)

	return
}

type localMail struct {
}

func (l localMail) Send(_ string, _ []string, msg io.WriterTo) error {
	buf := bytes.Buffer{}
	_, _ = msg.WriteTo(&buf)
	cmd := exec.Command("/usr/sbin/sendmail", "-t", "-oi", buf.String())
	return cmd.Run()
}

func (l localMail) Close() error {
	return nil
}
