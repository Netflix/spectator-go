package spectator

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type HttpClient struct {
	registry *Registry
	timeout  time.Duration
}

func NewHttpClient(registry *Registry, timeout time.Duration) *HttpClient {
	return &HttpClient{registry, timeout}
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

type httpResponse struct {
	status int
	body   []byte
}

func (h *HttpClient) doHttpPost(uri string, jsonBytes []byte, attemptNumber int) (response httpResponse, err error) {
	var willRetry bool
	response.status = -1
	log := h.registry.GetLogger()
	var req *http.Request
	req, err = h.createPayloadRequest(uri, jsonBytes)
	if err != nil {
		panic(err)
	}
	client := http.Client{}
	client.Timeout = h.timeout
	entry := NewLogEntry(h.registry, "POST", uri)
	log.Debugf("posting data to %s, payload %d bytes", uri, len(jsonBytes))
	resp, err := client.Do(req)
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

func (h *HttpClient) postJson(uri string, jsonBytes []byte) (response httpResponse, err error) {
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
