package tun

import (
	"bytes"
	"log"
	"net"
	"time"
)

const DialTimeout = 1 * time.Second

type ChainNode interface {
	GetProxyNode() *ProxyNode
	// First node need DialIn
	DialIn() (net.Conn, error)
	// Others need DialOut
	DialOut(c net.Conn, addr string) (net.Conn, error)
}

// Proxy chain holds a list of proxy nodes
type ProxyChain struct {
	cn   ChainNode
	next *ProxyChain
}

func (pc *ProxyChain) String() string {
	var buf bytes.Buffer
	buf.WriteString("&ProxyChain{")
	if pc.cn == nil {
		buf.WriteString("}")
	}
	if pc.next != nil {
		buf.WriteString(pc.next.String())
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
		}

		chain.AddChainNode(cn)
	}

	return chain, nil
}

func (pc *ProxyChain) Dial(network, addr string) (net.Conn, error) {
	// nil chain is also workable
	log.Printf("proxychian dial: %v\n", pc)
	if pc == nil {
		log.Println("no chain node, dial directly")
		return net.DialTimeout(network, addr, DialTimeout)
	}

	c, err := pc.DialIn()
	if err != nil {
		return nil, err
	}

	return pc.DialOut(c, addr)
}

func (pc *ProxyChain) DialIn() (net.Conn, error) {
	return net.DialTimeout("tcp", pc.cn.GetProxyNode().URL.Host, DialTimeout)
}

func (pc *ProxyChain) DialOut(c net.Conn, addr string) (net.Conn, error) {
	pc.cn.DialOut(c, addr)
	if pc.next == nil {
		return c, nil
	}
	return pc.next.DialOut(c, addr)
}
