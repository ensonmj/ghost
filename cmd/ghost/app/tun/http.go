package tun

import (
	"fmt"
	"net"
	"net/http"
	"net/url"

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
	return http.ListenAndServe(n.pn.URL.Host, GetHttpHandler(n.pc.Dial, true))
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
	return HandshakeForHttp(c, &n.pn.URL)
}

func (n *HttpChainNode) ForwardRequest(c net.Conn, url *url.URL) error {
	return ForwardRequestByHttp(c, url)
}
