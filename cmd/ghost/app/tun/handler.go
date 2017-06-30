package tun

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"

	"github.com/elazarl/goproxy"
	socks5 "github.com/ensonmj/go-socks5"
)

// http handler
func GetHttpHandler(dial func(network, addr string) (net.Conn, error), verbose bool) http.Handler {
	return &goproxy.ProxyHttpServer{
		Tr: &http.Transport{
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			MaxIdleConnsPerHost: 1000,
			DisableKeepAlives:   true,
			Dial:                dial,
		},
		NonproxyHandler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			http.Error(w, "This is a proxy server. Does not respond to non-proxy requests.", 500)
		}),
		Verbose: verbose,
		Logger:  log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds),
	}
}

type Socks5Handler interface {
	ServeConn(net.Conn) error
}

func GetSocks5Handler(user *url.Userinfo,
	dial func(network, addr string) (net.Conn, error)) Socks5Handler {
	dialCtx := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dial(network, addr)
	}
	return socks5.New(&socks5.Config{
		Dial: dialCtx,
	})
}
