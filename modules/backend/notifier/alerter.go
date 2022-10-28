package notifier

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	appctx "github.com/nixys/nxs-go-appctx/v2"
	"github.com/sirupsen/logrus"
	"nxs-backup/modules/logger"
)

// AlertServerOpts contains nxs-alert options
type AlertServerOpts struct {
	Enabled      bool
	NxsAlertURL  string
	AuthKey      string
	InsecureTLS  bool
	ProjectName  string
	ServerName   string
	MessageLevel logrus.Level
}

// AlertServer contains nxs-alert context data
type AlertServer struct {
	enabled      bool
	addr         string
	authKey      string
	client       *http.Client
	projectName  string
	serverName   string
	messageLevel logrus.Level
}

// messageRx contains message with result of request received from nxs-alert.
// General struct
type messageRx struct {
	Message string `json:"message"`
}

// AlertServerInit initiates new nxs-alert context
func AlertServerInit(opts AlertServerOpts) (AlertServer, error) {
	p := AlertServer{
		projectName:  opts.ProjectName,
		serverName:   opts.ServerName,
		messageLevel: opts.MessageLevel,
		enabled:      opts.Enabled,
	}

	if !opts.Enabled {
		return p, nil
	}

	_, err := url.Parse(opts.NxsAlertURL)
	if err != nil {
		return p, err
	}

	p.addr = opts.NxsAlertURL
	p.authKey = opts.AuthKey
	p.client = &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).DialContext,
			//ResponseHeaderTimeout: 60 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: opts.InsecureTLS,
			},
		},
	}

	return p, nil
}

// alertTx contains alert data requests for nxs-alert server
type alertTx struct {
	TriggerMessage    string `json:"triggerMessage"`
	IsEmergencyAlert  bool   `json:"isEmergencyAlert"`
	RAWTriggerMessage bool   `json:"rawTriggerMessage"`
	MonitoringURL     string `json:"monitoringURL"`
}

func (a *AlertServer) Send(appCtx *appctx.AppContext, n logger.LogRecord, wg *sync.WaitGroup) {
	if !a.enabled || n.Level > a.messageLevel {
		return
	}

	wg.Add(1)
	defer wg.Done()

	var m messageRx

	t, err := json.Marshal(alertTx{
		TriggerMessage:    a.getMessage(n),
		IsEmergencyAlert:  false,
		RAWTriggerMessage: false,
		MonitoringURL:     "-",
	})
	if err != nil {
		appCtx.Log().Errorf("Can't marshal request struct: %v", err)
		return
	}

	req, err := http.NewRequest("POST", a.addr, strings.NewReader(string(t)))
	if err != nil {
		appCtx.Log().Errorf("Can't create new request: %v", err)
		return
	}

	req.Header.Add("X-Auth-Key", a.authKey)

	res, err := a.client.Do(req)
	if err != nil {
		appCtx.Log().Errorf("Request error: %v", err)
		return
	}
	defer func() { _ = res.Body.Close() }()

	jd := json.NewDecoder(res.Body)
	if err = jd.Decode(&m); err != nil {
		appCtx.Log().Errorf("Can't decode response body: %v, response code: %d", err, res.StatusCode)
		return
	}

	if res.StatusCode != 200 {
		appCtx.Log().Errorf("Unexpected HTTP response code: %d, message: %v", res.StatusCode, m.Message)
	}
}

func (a *AlertServer) getMessage(n logger.LogRecord) (m string) {

	switch n.Level {
	case logrus.DebugLevel:
		m += "[DEBUG]\n\n"
	case logrus.InfoLevel:
		m += "[INFO]\n\n"
	case logrus.WarnLevel:
		m += "⚠️[WARNING]\n\n"
	case logrus.ErrorLevel:
		m += "‼️[ERROR]\n\n"
	}

	if a.projectName != "" {
		m += fmt.Sprintf("Project: %s\n", a.projectName)
	}
	if a.serverName != "" {
		m += fmt.Sprintf("Server: %s\n\n", a.serverName)
	}

	if n.JobName != "" {
		m += fmt.Sprintf("Job: %s\n", n.JobName)
	}
	if n.StorageName != "" {
		m += fmt.Sprintf("Storage: %s\n", n.StorageName)
	}
	m += fmt.Sprintf("\nMessage: %s\n", n.Message)

	return
}
