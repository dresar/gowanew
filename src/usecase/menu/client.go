package menu

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

var (
	sharedTransport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   3 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	fastHTTPClient = &http.Client{
		Transport: sharedTransport,
		Timeout:   4 * time.Second,
	}

	standardHTTPClient = &http.Client{
		Transport: sharedTransport,
		Timeout:   6 * time.Second,
	}
)

func GetFastClient() *http.Client {
	return fastHTTPClient
}

func GetStandardClient() *http.Client {
	return standardHTTPClient
}
