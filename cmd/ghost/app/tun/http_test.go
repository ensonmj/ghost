package tun

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestHttpServer(t *testing.T) {
	// http proxy server
	n := NewHttpServer(&ProxyNode{}, nil)
	proxySrv := httptest.NewServer(n.GetHttpProxyHandler(false))
	defer proxySrv.Close()

	// http client
	proxyUrl, _ := url.Parse(proxySrv.URL)
	err := setupSrvAndClient(&http.Transport{Proxy: http.ProxyURL(proxyUrl)})
	if err != nil {
		t.Error(err)
	}
}
