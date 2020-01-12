package spectator

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
)

type HttpClient struct {
	registry *Registry
	timeout  time.Duration
	client   *http.Client
}

func NewHttpClient(registry *Registry, timeout time.Duration) *HttpClient {
	return &HttpClient{registry, timeout, newSingleHostClient()}
}

func userFriendlyErr(errStr string) string {
	if strings.Contains(errStr, "connection refused") {
		return "ConnectException"
	}

	return "HttpErr"
}

func (h *HttpClient) createPayloadRequest(uri string, jsonBytes []byte) (*http.Request, error) {
	const JsonType = "application/json"
	const CompressThreshold = 512
	compressed := len(jsonBytes) > CompressThreshold
	var payloadBuffer *bytes.Buffer
	if compressed {
		payloadBuffer = &bytes.Buffer{}
		g := gzip.NewWriter(payloadBuffer)
		if _, err := g.Write(jsonBytes); err != nil {
			return nil, errors.Wrap(err, "Unable to compress json payload")
		}
		if err := g.Close(); err != nil {
			return nil, errors.Wrap(err, "Unable to close gzip stream")
		}
	} else {
		payloadBuffer = bytes.NewBuffer(jsonBytes)
	}

	req, err := http.NewRequest("POST", uri, payloadBuffer)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "spectator-go")
	req.Header.Set("Accept", JsonType)
	req.Header.Set("Content-Type", JsonType)
	if compressed {
		req.Header.Set("Content-Encoding", "gzip")
	}
	return req, nil
}

const maxAttempts = 3

type HttpResponse struct {
	status int
	body   []byte
}

func (h *HttpClient) doHttpPost(uri string, jsonBytes []byte, attemptNumber int) (response HttpResponse, err error) {
	var willRetry bool
	response.status = -1
	log := h.registry.GetLogger()
	var req *http.Request
	req, err = h.createPayloadRequest(uri, jsonBytes)
	if err != nil {
		panic(err)
	}
	entry := NewLogEntry(h.registry, "POST", uri)
	log.Debugf("posting data to %s, payload %d bytes", uri, len(jsonBytes))
	defer h.client.CloseIdleConnections()
	resp, err := h.client.Do(req)
	if err != nil {
		var status string
		if urlErr, ok := err.(*url.Error); ok {
			if urlErr.Timeout() {
				status = "timeout"
			} else if urlErr.Temporary() {
				status = "temporary"
			} else {
				status = userFriendlyErr(urlErr.Err.Error())
			}
		} else {
			status = err.Error()
		}
		entry.SetError(status)
		willRetry = false
		if timeout, ok := err.(*url.Error); ok && timeout.Timeout() {
			log.Infof("Timed out attempting to POST to %s", uri)
		} else {
			log.Infof("Unable to POST to %s: %v", uri, err)
		}
	} else {
		defer func() {
			if err = resp.Body.Close(); err != nil {
				log.Errorf("Unable to close body: %v", err)
			}
		}()
		response.status = resp.StatusCode
		entry.SetStatusCode(resp.StatusCode)
		var body []byte
		response.body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("Unable to read response body: %v", err)
			return
		}
		log.Debugf("response HTTP %d: %s", resp.StatusCode, body)
		if response.status >= 200 && response.status < 300 {
			entry.SetSuccess()
		} else {
			entry.SetError("http-error")
		}
		// only retry 503s for now
		willRetry = response.status == 503
	}

	final := !(willRetry && (attemptNumber+1) < maxAttempts)
	entry.SetAttempt(attemptNumber, final)
	entry.Log()
	return
}

func (h *HttpClient) PostJson(uri string, jsonBytes []byte) (response HttpResponse, err error) {
	for attemptNumber := 0; attemptNumber < maxAttempts; attemptNumber += 1 {
		response, err = h.doHttpPost(uri, jsonBytes, attemptNumber)
		willRetry := err == nil && response.status == 503
		if !willRetry {
			break
		}
		toSleep := 100 * time.Millisecond
		time.Sleep(time.Duration(attemptNumber+1) * toSleep)
	}
	return
}
func newSingleHostClient() *http.Client {
	return &http.Client{
		Transport: &keepAliveTransport{wrapped: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 90 * time.Second,
				DualStack: true,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          10,
			MaxIdleConnsPerHost:   10,
			MaxConnsPerHost:       30,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}},
	}
}

type keepAliveTransport struct {
	wrapped http.RoundTripper
}

func (k *keepAliveTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := k.wrapped.RoundTrip(r)
	if err != nil {
		return resp, err
	}
	resp.Body = &drainingReadCloser{rdr: resp.Body}
	return resp, nil
}

type drainingReadCloser struct {
	rdr     io.ReadCloser
	seenEOF uint32
}

func (d *drainingReadCloser) Read(p []byte) (n int, err error) {
	n, err = d.rdr.Read(p)
	if err == io.EOF || n == 0 {
		atomic.StoreUint32(&d.seenEOF, 1)
	}
	return
}

func (d *drainingReadCloser) Close() error {
	// drain buffer
	if atomic.LoadUint32(&d.seenEOF) != 1 {
		//#nosec
		_, _ = io.Copy(ioutil.Discard, d.rdr)
	}
	return d.rdr.Close()
}
