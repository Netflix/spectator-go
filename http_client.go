package spectator

import (
	"bytes"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/klauspost/compress/gzip"

	"github.com/pkg/errors"
)

// HttpClient represents a spectator HTTP client.
type HttpClient struct {
	Client            *http.Client
	registry          *Registry
	compressThreshold int
}

type HttpClientOption func(client *HttpClient)

// NewHttpClient generates a new *HttpClient, allowing us to specify the timeout on requests.
func NewHttpClient(registry *Registry, timeout time.Duration, opts ...HttpClientOption) *HttpClient {
	hc := &HttpClient{
		registry:          registry,
		Client:            newSingleHostClient(timeout),
		compressThreshold: 512,
	}
	for _, opt := range opts {
		opt(hc)
	}
	return hc
}

// WithCompressThreshold is an HttpClientOption that specifies the size
// threshold for compressing payloads. A value of 0 disables compression.
func WithCompressThreshold(threshold int) HttpClientOption {
	return func(hc *HttpClient) {
		hc.compressThreshold = threshold
	}
}

func userFriendlyErr(errStr string) string {
	if strings.Contains(errStr, "connection refused") {
		return "ConnectException"
	}

	return "HttpErr"
}

var payloadPool = &sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(nil)
	},
}

var gzipWriterPool = &sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(nil)
	},
}

const jsonType = "application/json"

func (h *HttpClient) createPayloadRequest(uri string, jsonBytes []byte) (*http.Request, func(), error) {
	log := h.registry.GetLogger()
	compressed := h.compressThreshold != 0 && len(jsonBytes) > h.compressThreshold
	payloadBuffer := payloadPool.Get().(*bytes.Buffer)
	var g *gzip.Writer
	if compressed {
		g = gzipWriterPool.Get().(*gzip.Writer)
		g.Reset(payloadBuffer)
		defer func() {
			if err := g.Close(); err != nil {
				log.Debugf("closing gzip writer, best effort: %v", err)
			}
			gzipWriterPool.Put(g)
		}()

		if _, err := g.Write(jsonBytes); err != nil {
			return nil, func() {}, errors.Wrap(err, "Unable to compress json payload")
		}
		if err := g.Flush(); err != nil {
			return nil, func() {}, errors.Wrap(err, "Unable to flush gzip stream")
		}

		if err := g.Close(); err != nil {
			return nil, func() {}, errors.Wrap(err, "Unable to close gzip stream")
		}
	} else {
		if _, err := payloadBuffer.Write(jsonBytes); err != nil {
			return nil, func() {}, errors.Wrap(err, "write json to buffer")
		}
	}

	req, err := http.NewRequest("POST", uri, payloadBuffer)
	if err != nil {
		return nil, func() {}, err
	}
	req.Header.Set("User-Agent", "spectator-go")
	req.Header.Set("Accept", jsonType)
	req.Header.Set("Content-Type", jsonType)
	if compressed {
		req.Header.Set("Content-Encoding", "gzip")
	}
	return req, func() {
		payloadBuffer.Reset()
		payloadPool.Put(payloadBuffer)
	}, nil
}

const maxAttempts = 3

// HttpResponse represents a read HTTP response.
type HttpResponse struct {
	Status int
	Body   []byte
}

func (h *HttpClient) doHttpPost(uri string, jsonBytes []byte, attemptNumber int) (response HttpResponse, err error) {
	var willRetry bool
	response.Status = -1
	log := h.registry.GetLogger()
	var req *http.Request
	var cleanup func()
	req, cleanup, err = h.createPayloadRequest(uri, jsonBytes)
	if err != nil {
		panic(err)
	}
	entry := NewLogEntry(h.registry, "POST", uri)
	log.Debugf("posting data to %s, payload %d bytes", uri, len(jsonBytes))
	defer func() {
		cleanup()
		h.Client.CloseIdleConnections()
	}()
	resp, err := h.Client.Do(req)
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
		response.Status = resp.StatusCode
		entry.SetStatusCode(resp.StatusCode)
		var body []byte
		response.Body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("Unable to read response body: %v", err)
			return
		}
		log.Debugf("response HTTP %d: %s", resp.StatusCode, body)
		if response.Status >= 200 && response.Status < 300 {
			entry.SetSuccess()
		} else {
			entry.SetError("http-error")
		}
		// only retry 503s for now
		willRetry = response.Status == 503
	}

	final := !(willRetry && (attemptNumber+1) < maxAttempts)
	entry.SetAttempt(attemptNumber, final)
	entry.Log()
	return
}

// PostJson attempts to submit JSON to the uri, with a 100 ms delay between
// failures.
func (h *HttpClient) PostJson(uri string, jsonBytes []byte) (response HttpResponse, err error) {
	for attemptNumber := 0; attemptNumber < maxAttempts; attemptNumber += 1 {
		response, err = h.doHttpPost(uri, jsonBytes, attemptNumber)
		willRetry := err == nil && response.Status == 503
		if !willRetry {
			break
		}
		toSleep := 100 * time.Millisecond
		time.Sleep(time.Duration(attemptNumber+1) * toSleep)
	}
	return
}

func newSingleHostClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &keepAliveTransport{wrapped: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          50,
			MaxIdleConnsPerHost:   50,
			MaxConnsPerHost:       100,
			IdleConnTimeout:       30 * time.Second,
			TLSHandshakeTimeout:   2 * time.Second,
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
