package tun

import (
	"net/http"
	"testing"

	"golang.org/x/net/proxy"
)

func TestSocks5Server(t *testing.T) {
	// socks5 proxy server
	n, _ := ParseProxyNode("socks://127.0.0.1:8080")
	proxySrv := NewSocks5Server(n)
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
