package tun

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"

	"github.com/elazarl/goproxy"
	"github.com/pkg/errors"
)

type HttpServer struct {
	pn *ProxyNode
	pc *ProxyChain
}

func NewHttpServer(pn *ProxyNode, pc *ProxyChain) *HttpServer {
	return &HttpServer{
		pn: pn,
		pc: pc,
	}
}

func (n *HttpServer) ListenAndServe() error {
	return http.ListenAndServe(n.pn.URL.Host, n.GetHttpProxyHandler(true))
}

func (n *HttpServer) GetHttpProxyHandler(verbose bool) http.Handler {
	return &goproxy.ProxyHttpServer{
		Tr: &http.Transport{
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			MaxIdleConnsPerHost: 1000,
			DisableKeepAlives:   true,
			Dial:                n.pc.Dial,
		},
		NonproxyHandler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			http.Error(w, "This is a proxy server. Does not respond to non-proxy requests.", 500)
		}),
		Verbose: verbose,
		Logger:  log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds),
	}
}

type HttpChainNode struct {
	pn *ProxyNode
}

func NewHttpChainNode(pn *ProxyNode) *HttpChainNode {
	return &HttpChainNode{
		pn: pn,
	}
}

func (n *HttpChainNode) String() string {
	return fmt.Sprintf("%s", n.pn)
}

func (n *HttpChainNode) URL() *url.URL {
	return &n.pn.URL
}

func (n *HttpChainNode) Connect() (net.Conn, error) {
	c, err := net.Dial("tcp", n.pn.URL.Host)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	return c, nil
}

func (n *HttpChainNode) Handshake(c net.Conn) error {
	return HandshakeForHttp(c)
}

func (n *HttpChainNode) ForwardRequest(c net.Conn, url *url.URL) error {
	return ForwardRequestByHttp(c, url)
}
