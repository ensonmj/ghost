package tun

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"golang.org/x/net/proxy"
)

func TestHttpChainHttp(t *testing.T) {
	// chained http proxy server
	cproxySrv := httptest.NewServer(GetHttpHandler(nil, true))
	defer cproxySrv.Close()

	// http proxy server with chain
	pc, err := ParseProxyChain(cproxySrv.URL)
	if err != nil {
		t.Fatal(err)
	}
	proxySrv := httptest.NewServer(GetHttpHandler(pc.Dial, true))
	defer proxySrv.Close()

	// http client
	proxyUrl, _ := url.Parse(proxySrv.URL)
	err = setupSrvAndClient(&http.Transport{Proxy: http.ProxyURL(proxyUrl)})
	if err != nil {
		t.Error(err)
	}
}

func TestSocksChainHttp(t *testing.T) {
	// chained http proxy server
	cproxySrv := httptest.NewServer(GetHttpHandler(nil, true))
	defer cproxySrv.Close()

	// socks server with chain
	pc, err := ParseProxyChain(cproxySrv.URL)
	if err != nil {
		t.Fatal(err)
	}
	n, _ := ParseProxyNode("socks://127.0.0.1:8080")
	defer setupSocks5Server(n, pc).Close()

	// http client transport
	dialer, err := proxy.FromURL(&n.URL, proxy.Direct)
	if err != nil {
		t.Error(err)
	}
	err = setupSrvAndClient(&http.Transport{Dial: dialer.Dial})
	if err != nil {
		t.Error(err)
	}
}

func TestHttpChainSocks(t *testing.T) {
	// chained socks server
	cn, _ := ParseProxyNode("socks://127.0.0.1:8080")
	defer setupSocks5Server(cn, nil).Close()

	// http proxy server with chain
	pc, err := NewProxyChain(cn)
	if err != nil {
		t.Fatal(err)
	}
	proxySrv := httptest.NewServer(GetHttpHandler(pc.Dial, true))
	defer proxySrv.Close()

	// http client
	proxyUrl, _ := url.Parse(proxySrv.URL)
	err = setupSrvAndClient(&http.Transport{Proxy: http.ProxyURL(proxyUrl)})
	if err != nil {
		t.Error(err)
	}
}
