package webdav

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Client struct {
	http.Client
	Params
}

type Params struct {
	URL               string
	Username          string
	Password          string
	OAuthToken        string
	ConnectionTimeout time.Duration
}

type quotaResp struct {
	Responses []struct {
		Href  string `xml:"href"`
		Props []struct {
			Available int64 `xml:"prop>quota-available-bytes,omitempty"`
			Used      int64 `xml:"prop>quota-used-bytes,omitempty"`
		} `xml:"propstat"`
	} `xml:"response"`
}

type listResp struct {
	Responses []struct {
		Href  string `xml:"href"`
		Props []struct {
			Name     string   `xml:"prop>displayname,omitempty"`
			Type     xml.Name `xml:"prop>resourcetype>collection,omitempty"`
			Size     int64    `xml:"prop>getcontentlength,omitempty"`
			Modified string   `xml:"prop>getlastmodified,omitempty"`
		} `xml:"propstat"`
	} `xml:"response"`
}

func Init(p Params) (*Client, error) {

	wd := new(Client)

	wd.Params = p

	if wd.OAuthToken == "" && wd.Username == "" && wd.Password == "" {
		return nil, fmt.Errorf("auth not defined. OAuthToken or User/Pass should be provided")
	}

	wdURL := regexp.MustCompile(`\/$`).ReplaceAllString(wd.URL, "")
	if !strings.HasPrefix(wdURL, "http://") && !strings.HasPrefix(wdURL, "https://") {
		wdURL = "https://" + wdURL
	}
	wd.URL = wdURL

	wd.Client = http.Client{
		Timeout: wd.ConnectionTimeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 10 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			IdleConnTimeout:       60 * time.Second,
			ResponseHeaderTimeout: 60 * time.Second,
		},
	}

	_, err := wd.getQuotaAvailableBytes()
	if err != nil {
		return nil, err
	}

	return wd, nil
}

func (w *Client) getQuotaAvailableBytes() (int, error) {

	query := `<d:propfind xmlns:d='DAV:'>
			<d:prop>
				<d:quota-available-bytes/>
				<d:quota-used-bytes/>
			</d:prop>
		</d:propfind>`

	res, err := w.request("PROPFIND", "/", strings.NewReader(query), nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode >= 400 {
		return 0, fmt.Errorf("%s(%d): can't get strage quota", httpFriendlyStatus(res.StatusCode), res.StatusCode)
	}

	var r quotaResp
	decoder := xml.NewDecoder(res.Body)
	err = decoder.Decode(&r)
	if err != nil {
		return 0, fmt.Errorf("can't decode server response: %s", err)
	}
	if len(r.Responses) == 0 {
		return 0, fmt.Errorf("server not found(404)")
	}

	return int(r.Responses[0].Props[0].Available), nil
}

func (w *Client) Ls(path string) ([]os.FileInfo, error) {
	files := make([]os.FileInfo, 0)
	query := `<d:propfind xmlns:d='DAV:'>
			<d:prop>
				<d:displayname/>
				<d:resourcetype/>
				<d:getlastmodified/>
				<d:getcontentlength/>
			</d:prop>
		</d:propfind>`

	res, err := w.request("PROPFIND", path, strings.NewReader(query), func(req *http.Request) {
		req.Header.Add("Depth", "1")
	})
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode >= 400 {
		if res.StatusCode == 404 {
			return nil, err
		}
		return nil, fmt.Errorf("%s(%d): can't get things in %s", httpFriendlyStatus(res.StatusCode), res.StatusCode, filepath.Base(path))
	}

	var r listResp
	decoder := xml.NewDecoder(res.Body)
	_ = decoder.Decode(&r)
	if len(r.Responses) == 0 {
		return nil, fmt.Errorf("server not found(404)")
	}

	LongURLDav := w.URL + path
	ShortURLDav := regexp.MustCompile(`^http[s]?://[^/]*`).ReplaceAllString(LongURLDav, "")
	for _, tag := range r.Responses {
		decodedHref := decodeURL(tag.Href)
		if decodedHref == ShortURLDav || decodedHref == LongURLDav {
			continue
		}

		for i, prop := range tag.Props {
			if i > 0 {
				break
			}
			files = append(files, &webDavFile{
				name: filepath.Base(decodedHref),
				mode: func(p string) os.FileMode {
					if p == "collection" {
						return fs.ModeDir
					}
					return fs.ModeType
				}(prop.Type.Local),
				mtime: func() time.Time {
					t, err := time.Parse(time.RFC1123, prop.Modified)
					if err != nil {
						return time.Time{}
					}
					return t
				}(),
				size: prop.Size,
			})
		}
	}

	return files, nil
}

func (w *Client) Mkdir(path string) error {
	res, err := w.request("MKCOL", path, nil, func(req *http.Request) {
		req.Header.Add("Overwrite", "F")
	})
	if err != nil {
		return err
	}
	_ = res.Body.Close()
	if res.StatusCode >= 400 {
		return fmt.Errorf("%s(%d): can't create %s", httpFriendlyStatus(res.StatusCode), res.StatusCode, filepath.Base(path))
	}
	return nil
}

func (w *Client) Upload(path string, file io.Reader) error {
	res, err := w.request("PUT", path, file, nil)
	if err != nil {
		return err
	}
	_ = res.Body.Close()
	if res.StatusCode >= 400 {
		return fmt.Errorf("%s(%d): can't upload file %s", httpFriendlyStatus(res.StatusCode), res.StatusCode, filepath.Base(path))
	}
	return nil
}

func (w *Client) Copy(src, dst string) error {
	res, err := w.request("COPY", src, nil, func(req *http.Request) {
		req.Header.Add("Destination", w.URL+encodeURL(dst))
		req.Header.Add("Overwrite", "F")
	})
	if err != nil {
		return err
	}
	_ = res.Body.Close()
	if res.StatusCode >= 400 {
		return fmt.Errorf("%s(%d): can't copy %s to %s", httpFriendlyStatus(res.StatusCode), res.StatusCode, src, dst)
	}
	return nil
}

func (w *Client) Read(path string) (io.ReadCloser, error) {
	res, err := w.request("GET", path, nil, nil)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("%s(%d): can't read %s", httpFriendlyStatus(res.StatusCode), res.StatusCode, path)
	}
	return res.Body, nil
}

func (w *Client) Rm(path string) error {
	res, err := w.request("DELETE", path, nil, nil)
	if err != nil {
		return err
	}
	_ = res.Body.Close()
	if res.StatusCode >= 400 {
		return fmt.Errorf("%s(%d): can't remove %s", httpFriendlyStatus(res.StatusCode), res.StatusCode, filepath.Base(path))
	}
	return nil
}

func (w *Client) request(method, path string, body io.Reader, fn func(req *http.Request)) (*http.Response, error) {
	req, err := http.NewRequest(method, w.URL+encodeURL(path), body)
	if err != nil {
		return nil, err
	}

	if w.Username != "" {
		req.SetBasicAuth(w.Username, w.Password)
	} else {
		req.Header.Set("Authorization", "OAuth "+w.OAuthToken)
	}
	req.Header.Add("Content-Type", "text/xml;charset=UTF-8")
	req.Header.Add("Accept", "application/xml,text/xml")
	req.Header.Add("Accept-Charset", "utf-8")

	if req.Body != nil {
		defer func() { _ = req.Body.Close() }()
	}
	if fn != nil {
		fn(req)
	}

	return w.Do(req)
}

func encodeURL(path string) string {
	p := url.PathEscape(path)
	return strings.Replace(p, "%2F", "/", -1)
}

func decodeURL(path string) string {
	str, err := url.PathUnescape(path)
	if err != nil {
		return path
	}
	return str
}
