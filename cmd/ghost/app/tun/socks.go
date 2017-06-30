package tun

import (
	"fmt"
	"net"
	"net/url"

	socks5 "github.com/ensonmj/go-socks5"
	"github.com/pkg/errors"
)

type Socks5Server struct {
	pn      *ProxyNode
	pc      *ProxyChain
	handler *socks5.Server
}

func NewSocks5Server(pn *ProxyNode, pc *ProxyChain) *Socks5Server {
	return &Socks5Server{
		pn:      pn,
		pc:      pc,
		handler: GetSocks5Handler(pn.URL.User, pc.Dial),
	}
}

func (n *Socks5Server) ListenAndServe() error {
	return n.handler.ListenAndServe("tcp", n.pn.URL.Host)
}

type Socks5ChainNode struct {
	pn *ProxyNode
}

func NewSocks5ChainNode(pn *ProxyNode) *Socks5ChainNode {
	return &Socks5ChainNode{
		pn: pn,
	}
}

func (n *Socks5ChainNode) String() string {
	return fmt.Sprintf("%s", n.pn)
}

func (n *Socks5ChainNode) URL() *url.URL {
	return &n.pn.URL
}

func (n *Socks5ChainNode) Connect() (net.Conn, error) {
	c, err := net.Dial("tcp", n.pn.URL.Host)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return c, nil
}

func (n *Socks5ChainNode) Handshake(c net.Conn) error {
	return HandshakeForSocks5(c, &n.pn.URL)
}

func (n *Socks5ChainNode) ForwardRequest(c net.Conn, url *url.URL) error {
	return ForwardRequestBySocks5(c, url)
}
