package webhooker

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/nixys/nxs-backup/misc"
	"github.com/nixys/nxs-backup/modules/logger"
	"github.com/sirupsen/logrus"
)

// Opts contains webhook options
type Opts struct {
	WebhookURL        string
	InsecureTLS       bool
	PayloadMessageKey string
	ExtraPayload      map[string]interface{}
	ExtraHeaders      map[string]string
	MessageLevel      logrus.Level
	ProjectName       string
	ServerName        string
}

type webhook struct {
	opts Opts
	hc   *http.Client
}

func Init(opts Opts) (*webhook, error) {

	wh := &webhook{
		opts: opts,
	}

	_, err := url.Parse(opts.WebhookURL)
	if err != nil {
		return wh, err
	}

	d := &net.Dialer{
		Timeout: 5 * time.Second,
	}
	wh.hc = &http.Client{
		Transport: &http.Transport{
			DialContext: d.DialContext,
			//ResponseHeaderTimeout: 60 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: opts.InsecureTLS,
			},
		},
	}

	return wh, nil
}

func (wh *webhook) Send(log *logrus.Logger, n logger.LogRecord) {
	if n.Level > wh.opts.MessageLevel {
		return
	}

	req, err := http.NewRequest(http.MethodPost, wh.opts.WebhookURL, bytes.NewBuffer(wh.getJsonData(log, n)))
	if err != nil {
		log.Errorf("Can't create webhook request: %v", err)
		return
	}

	for k, v := range wh.opts.ExtraHeaders {
		req.Header.Add(k, v)
	}

	resp, err := wh.hc.Do(req)
	if err != nil {
		log.Errorf("Request error: %v", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	log.Tracef("HTTP response code: %d, body: %v", resp.StatusCode, string(body))

	if resp.StatusCode != 200 {
		log.Errorf("Unexpected HTTP response code: %d, body: %v", resp.StatusCode, string(body))
	}
}

func (wh *webhook) getJsonData(log *logrus.Logger, n logger.LogRecord) []byte {
	data := make(map[string]interface{})

	data[wh.opts.PayloadMessageKey] = misc.GetMessage(n, wh.opts.ProjectName, wh.opts.ServerName)
	for k, v := range wh.opts.ExtraPayload {
		data[k] = v
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Errorf("Can't marshal json for webhook request: %v", err)
		return nil
	}

	return jsonData
}
