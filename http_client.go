package spectator

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
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

func (h *HttpClient) PostJson(uri string, jsonBytes []byte) (statusCode int) {
	const JsonType = "application/json"
	const CompressThreshold = 512
	statusCode = 400
	var buf bytes.Buffer
	payloadBuffer := &buf
	log := h.registry.log
	compressed := len(jsonBytes) > CompressThreshold
	if compressed {
		g := gzip.NewWriter(payloadBuffer)
		if _, err := g.Write(jsonBytes); err != nil {
			log.Errorf("Unable to compress json payload: %v", err)
			return
		}
		if err := g.Close(); err != nil {
			log.Errorf("Unable to close gzip stream: %v", err)
			return
		}
	} else {
		payloadBuffer = bytes.NewBuffer(jsonBytes)
	}
	req, err := http.NewRequest("POST", uri, payloadBuffer)
	req.Header.Set("User-Agent", "spectator-go")
	req.Header.Set("Accept", JsonType)
	req.Header.Set("Content-Type", JsonType)
	if compressed {
		req.Header.Set("Content-Encoding", "gzip")
	}
	if err != nil {
		panic(err)
	}
	client := http.Client{}
	client.Timeout = h.timeout

	tags := map[string]string{
		"client": "spectator-go",
		"method": "POST",
		"mode":   "http-client",
	}

	start := h.registry.clock.MonotonicTime()
	log.Debugf("posting data to %s, payload %s", uri, string(jsonBytes))
	resp, err := client.Do(req)
	if err != nil {
		if urlerr, ok := err.(*url.Error); ok {
			if urlerr.Timeout() {
				tags["status"] = "timeout"
			} else if urlerr.Temporary() {
				tags["status"] = "temporary"
			} else {
				tags["status"] = userFriendlyErr(urlerr.Err.Error())
			}
		} else {
			tags["status"] = err.Error()
		}
		tags["statusCode"] = tags["status"]
		log.Errorf("Unable to POST to %s: %v", uri, err)
	} else {
		defer resp.Body.Close()
		statusCode = resp.StatusCode
		tags["statusCode"] = strconv.Itoa(resp.StatusCode)
		tags["status"] = fmt.Sprintf("%dxx", resp.StatusCode/100)
		body, _ := ioutil.ReadAll(resp.Body)
		log.Debugf("request succeeded (%d): %s", resp.StatusCode, body)

	}
	duration := h.registry.clock.MonotonicTime() - start
	h.registry.Timer("http.req.complete", tags).Record(duration)
	return
}
