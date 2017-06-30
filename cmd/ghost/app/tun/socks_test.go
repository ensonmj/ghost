package tun

import (
	"net"
	"net/http"
	"testing"

	"github.com/pkg/errors"

	"golang.org/x/net/proxy"
)

func setupSocks5Server(pn *ProxyNode, pc *ProxyChain) net.Listener {
	ln, err := net.Listen("tcp", pn.URL.Host)
	if err != nil {
		panic(errors.WithStack(err))
	}
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			panic(errors.WithStack(err))
		}
		GetSocks5Handler(pn.URL.User, pc.Dial).ServeConn(conn)
	}()

	return ln
}

func TestSocks5Server(t *testing.T) {
	// socks5 proxy server
	n, _ := ParseProxyNode("socks://127.0.0.1:8080")
	defer setupSocks5Server(n, nil).Close()

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

func TestSocks5Auth(t *testing.T) {
	// socks5 proxy server
	n, _ := ParseProxyNode("socks://test:test@127.0.0.1:8080")
	defer setupSocks5Server(n, nil).Close()

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
