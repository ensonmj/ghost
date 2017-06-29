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
	cn := NewHttpServer(&ProxyNode{}, nil)
	cproxySrv := httptest.NewServer(cn.GetHttpProxyHandler(true))
	defer cproxySrv.Close()

	// http proxy server with chain
	pc, err := ParseProxyChain(cproxySrv.URL)
	if err != nil {
		t.Fatal(err)
	}
	n := NewHttpServer(&ProxyNode{}, pc)
	proxySrv := httptest.NewServer(n.GetHttpProxyHandler(true))
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
	cn := NewHttpServer(&ProxyNode{}, nil)
	cproxySrv := httptest.NewServer(cn.GetHttpProxyHandler(true))
	defer cproxySrv.Close()

	// socks server with chain
	pc, err := ParseProxyChain(cproxySrv.URL)
	if err != nil {
		t.Fatal(err)
	}
	n, _ := ParseProxyNode("socks://127.0.0.1:8080")
	proxySrv := NewSocks5Server(n, pc)
	ln := proxySrv.listen()
	defer ln.Close()
	go proxySrv.serveOnce(ln)

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
	cproxySrv := NewSocks5Server(cn, nil)
	ln := cproxySrv.listen()
	defer ln.Close()
	go cproxySrv.serveOnce(ln)

	// http proxy server with chain
	pc, err := NewProxyChain(cn)
	if err != nil {
		t.Fatal(err)
	}
	n := NewHttpServer(&ProxyNode{}, pc)
	proxySrv := httptest.NewServer(n.GetHttpProxyHandler(true))
	defer proxySrv.Close()

	// http client
	proxyUrl, _ := url.Parse(proxySrv.URL)
	err = setupSrvAndClient(&http.Transport{Proxy: http.ProxyURL(proxyUrl)})
	if err != nil {
		t.Error(err)
	}
}
