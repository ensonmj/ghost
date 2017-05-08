package tun

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/elazarl/goproxy"
)

func GetHttpProxyHandler(verbose bool) http.Handler {
	handler := goproxy.NewProxyHttpServer()
	handler.Verbose = verbose

	return handler
}

func TestHttpChainHttp(t *testing.T) {
	// http server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "success")
	}))
	defer srv.Close()
	t.Logf("server addr: %s\n", srv.URL)

	// http chained proxy server
	cproxySrv := httptest.NewServer(GetHttpProxyHandler(true))
	defer cproxySrv.Close()

	// http proxy server with chain
	chain, err := NewProxyChain(cproxySrv.URL)
	t.Logf("chain addr: %s\n", cproxySrv.URL)
	if err != nil {
		t.Fatal(err)
	}
	n := NewHttpNode(&ProxyNode{})
	n.chain = chain
	proxySrv := httptest.NewServer(n.GetHttpProxyHandlerWithProxy(true))
	defer proxySrv.Close()
	t.Logf("proxy addr: %s\n", proxySrv.URL)

	// http client
	proxyUrl, _ := url.Parse(proxySrv.URL)
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
		},
	}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Error(err)
	}
	txt, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Error(err)
	}
	if string(txt) != "success" {
		t.Errorf("expect success, but got %s\n", txt)
	}
}
