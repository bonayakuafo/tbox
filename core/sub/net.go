package sub

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

const maxRetries = 3

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	if err == io.EOF {
		return true
	}
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout() || netErr.Temporary()
	}
	return false
}

func doRequestWithRetry(doReq func() (*http.Response, error)) (*http.Response, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		resp, err := doReq()
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !isRetryableError(err) {
			return nil, err
		}
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}
	return nil, lastErr
}

// GetByHTTPProxy fetches a URL through an HTTP proxy.
func GetByHTTPProxy(objUrl, proxyAddress string, proxyPort int, timeOut time.Duration, userAgent string) (*http.Response, error) {
	return doRequestWithRetry(func() (*http.Response, error) {
		proxy := func(_ *http.Request) (*url.URL, error) {
			return url.Parse(fmt.Sprintf("http://%s:%d", proxyAddress, proxyPort))
		}
		transport := &http.Transport{Proxy: proxy}
		client := &http.Client{
			Transport: transport,
			Timeout:   timeOut,
		}
		req, _ := http.NewRequest("GET", objUrl, nil)
		req.Header.Set("User-Agent", userAgent)
		return client.Do(req)
	})
}

// GetBySocks5Proxy fetches a URL through a SOCKS5 proxy.
func GetBySocks5Proxy(objUrl, proxyAddress string, proxyPort int, timeOut time.Duration, userAgent string) (*http.Response, error) {
	return doRequestWithRetry(func() (*http.Response, error) {
		proxy := func(_ *http.Request) (*url.URL, error) {
			return url.Parse(fmt.Sprintf("socks5://%s:%d", proxyAddress, proxyPort))
		}
		transport := &http.Transport{Proxy: proxy}
		client := &http.Client{
			Transport: transport,
			Timeout:   timeOut,
		}
		req, _ := http.NewRequest("GET", objUrl, nil)
		req.Header.Set("User-Agent", userAgent)
		return client.Do(req)
	})
}

// GetNoProxy fetches a URL directly without a proxy.
func GetNoProxy(objUrl string, timeOut time.Duration, userAgent string) (*http.Response, error) {
	return doRequestWithRetry(func() (*http.Response, error) {
		client := &http.Client{
			Timeout: timeOut,
		}
		req, _ := http.NewRequest("GET", objUrl, nil)
		req.Header.Set("User-Agent", userAgent)
		return client.Do(req)
	})
}

// ReadDate reads the http response body as a string.
func ReadDate(resp *http.Response) string {
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}
