package notifier

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	appctx "github.com/nixys/nxs-go-appctx/v2"
	"github.com/sirupsen/logrus"
	"nxs-backup/misc"
	"nxs-backup/modules/logger"
)

// WebhookOpts contains webhook options
type WebhookOpts struct {
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
	opts WebhookOpts
	hc   *http.Client
}

func WebhookInit(opts WebhookOpts) (*webhook, error) {

	wh := &webhook{
		opts: opts,
	}

	_, err := url.Parse(opts.WebhookURL)
	if err != nil {
		return wh, err
	}

	wh.hc = &http.Client{
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

	return wh, nil
}

func (wh *webhook) Send(appCtx *appctx.AppContext, n logger.LogRecord, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	if n.Level > wh.opts.MessageLevel {
		return
	}

	req, err := http.NewRequest(http.MethodPost, wh.opts.WebhookURL, bytes.NewBuffer(wh.getJsonData(appCtx, n)))
	if err != nil {
		appCtx.Log().Errorf("Can't create webhook request: %v", err)
		return
	}

	for k, v := range wh.opts.ExtraHeaders {
		req.Header.Add(k, v)
	}

	resp, err := wh.hc.Do(req)
	if err != nil {
		appCtx.Log().Errorf("Request error: %v", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	appCtx.Log().Debugf("HTTP response code: %d, body: %v", resp.StatusCode, string(body))

	if resp.StatusCode != 200 {
		appCtx.Log().Errorf("Unexpected HTTP response code: %d, body: %v", resp.StatusCode, string(body))
	}
}

func (wh *webhook) getJsonData(appCtx *appctx.AppContext, n logger.LogRecord) []byte {
	data := make(map[string]interface{})

	data[wh.opts.PayloadMessageKey] = misc.GetMessage(n, wh.opts.ProjectName, wh.opts.ServerName)
	for k, v := range wh.opts.ExtraPayload {
		data[k] = v
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		appCtx.Log().Errorf("Can't marshal json for webhook request: %v", err)
		return nil
	}

	return jsonData
}
