package tun

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestHttpChainHttp(t *testing.T) {
	// http chained proxy server
	cn := NewHttpNode(&ProxyNode{})
	cproxySrv := httptest.NewServer(cn.GetHttpProxyHandler(true))
	defer cproxySrv.Close()

	// http proxy server with chain
	pc, err := NewProxyChain(cproxySrv.URL)
	t.Logf("chain addr: %s\n", cproxySrv.URL)
	if err != nil {
		t.Fatal(err)
	}
	n := NewHttpNode(&ProxyNode{})
	n.pc = pc
	proxySrv := httptest.NewServer(n.GetHttpProxyHandler(true))
	defer proxySrv.Close()
	t.Logf("proxy addr: %s\n", proxySrv.URL)

	// http client
	proxyUrl, _ := url.Parse(proxySrv.URL)
	err = setupSrvAndClient(&http.Transport{Proxy: http.ProxyURL(proxyUrl)})
	if err != nil {
		t.Error(err)
	}
}
