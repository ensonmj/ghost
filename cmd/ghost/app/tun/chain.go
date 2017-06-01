package tun

import (
	"bytes"
	"log"
	"net"
	"time"

	"github.com/pkg/errors"
)

const DialTimeout = 1 * time.Second

type ChainNode interface {
	String() string
	// First node need Connect
	Connect() (net.Conn, error)
	// Handshake complete authentication with node
	Handshake(c net.Conn) error
	// ForwardRequest ask node to connect next hop
	ForwardRequest(c net.Conn, addr string) error
}

// Proxy chain holds a list of proxy nodes
type ProxyChain struct {
	cn   ChainNode
	next *ProxyChain
}

func (pc *ProxyChain) String() string {
	if pc == nil {
		return "<nil>"
	}

	var buf bytes.Buffer
	buf.WriteString("&ProxyChain{")
	if pc.cn == nil {
		buf.WriteString("}")
		return buf.String()
	}

	buf.WriteString(pc.cn.String())
	if pc.next != nil {
		buf.WriteString(pc.next.String())
	} else {
		buf.WriteString("}")
	}
	return buf.String()
}

func (pc *ProxyChain) AddChainNode(cn ChainNode) {
	if pc.cn == nil {
		pc.cn = cn
		return
	}

	if pc.next == nil {
		pc.next = &ProxyChain{cn: cn}
		return
	}

	pc.next.AddChainNode(cn)
}

func NewProxyChain(nodes ...string) (*ProxyChain, error) {
	if len(nodes) <= 0 {
		log.Println("no chain node")
		return nil, nil
	}

	chain := &ProxyChain{}
	for _, n := range nodes {
		pn, err := ParseProxyNode(n)
		if err != nil {
			return nil, err
		}
		var cn ChainNode
		switch pn.URL.Scheme {
		case "http":
			cn = NewHttpNode(pn)
		case "socks5":
			cn = NewSocks5Server(pn)
		case "quic":
			cn = NewQuicServer(pn, nil)
		default:
			return nil, errors.Errorf("unknown scheme:%s", pn.URL.Scheme)
		}

		chain.AddChainNode(cn)
	}

	return chain, nil
}

func (pc *ProxyChain) Dial(network, addr string) (net.Conn, error) {
	log.Printf("connect to chian: %s\n", pc)
	if pc == nil {
		// nil chain is also workable
		return net.DialTimeout(network, addr, DialTimeout)
	}

	c, err := pc.Connect()
	if err != nil {
		return nil, err
	}

	err = pc.Handshake(c, addr)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (pc *ProxyChain) Connect() (net.Conn, error) {
	return pc.cn.Connect()
}

func (pc *ProxyChain) Handshake(c net.Conn, addr string) error {
	pc.cn.Handshake(c)
	err := pc.cn.ForwardRequest(c, addr)
	if err != nil {
		return err
	}
	if pc.next == nil {
		return nil
	}

	return pc.next.Handshake(c, addr)
}
