package tun

import (
	"fmt"
	"net"
	"net/url"

	"github.com/pkg/errors"
)

type Socks5Server struct {
	pn      *ProxyNode
	pc      *ProxyChain
	handler Socks5Handler
}

func NewSocks5Server(pn *ProxyNode, pc *ProxyChain) *Socks5Server {
	return &Socks5Server{
		pn:      pn,
		pc:      pc,
		handler: GetSocks5Handler(pn.URL.User, pc.Dial),
	}
}

func (n *Socks5Server) ListenAndServe() error {
	n.serve(n.listen())
	return nil
}

func (n *Socks5Server) listen() net.Listener {
	ln, err := net.Listen("tcp", n.pn.URL.Host)
	if err != nil {
		panic(errors.Wrap(err, "socks server listen"))
	}
	return ln
}

func (n *Socks5Server) serve(ln net.Listener) {
	defer ln.Close()
	for {
		n.serveOnce(ln)
	}
}

func (n *Socks5Server) serveOnce(ln net.Listener) {
	conn, err := ln.Accept()
	if err != nil {
		return
	}

	go func() {
		n.handler.ServeConn(conn)
	}()
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
