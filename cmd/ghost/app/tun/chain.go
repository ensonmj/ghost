package tun

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"golang.org/x/net/http2"
)

const DialTimeout = 1 * time.Second
const KeepAliveTime = 1 * time.Second

// Proxy chain holds a list of proxy nodes
type ProxyChain struct {
	nodes []ProxyNode
}

func NewProxyChain(nodes ...string) (*ProxyChain, error) {
	chain := &ProxyChain{}

	for _, n := range nodes {
		node, err := ParseProxyNode(n)
		if err != nil {
			return nil, err
		}
		chain.AddProxyNode(node)
	}

	return chain, nil
}

func (c *ProxyChain) AddProxyNode(node ProxyNode) {
	c.nodes = append(c.nodes, node)
}

func (c *ProxyChain) Nodes() []ProxyNode {
	return c.nodes
}

func (c *ProxyChain) GetNode(index int) *ProxyNode {
	if index < len(c.nodes) {
		return &c.nodes[index]
	}
	return nil
}

func (c *ProxyChain) SetNode(index int, node ProxyNode) {
	if index < len(c.nodes) {
		c.nodes[index] = node
	}
}

func enablePing(conn net.Conn, interval time.Duration) {
	if conn == nil || interval == 0 {
		return
	}

	log.Println("[http2] ping enabled, interval:", interval)
	go func() {
		t := time.NewTicker(interval)
		var framer *http2.Framer
		for {
			select {
			case <-t.C:
				if framer == nil {
					framer = http2.NewFramer(conn, conn)
				}

				var p [8]byte
				rand.Read(p[:])
				err := framer.WritePing(false, p)
				if err != nil {
					t.Stop()
					framer = nil
					log.Println("[http2] ping:", err)
					return
				}
			}
		}
	}()
}

// Connect to addr through proxy chain
func (c *ProxyChain) Dial(addr string) (net.Conn, error) {
	if !strings.Contains(addr, ":") {
		addr += ":80"
	}
	return c.dialWithNodes(addr, c.nodes...)
}

// GetConn initializes a proxy chain connection,
// if no proxy nodes on this chain, it will return error
func (c *ProxyChain) GetConn() (net.Conn, error) {
	nodes := c.nodes
	if len(nodes) == 0 {
		return nil, errors.New("empty")
	}

	return c.travelNodes(nodes...)
}

func (c *ProxyChain) dialWithNodes(addr string, nodes ...ProxyNode) (conn net.Conn, err error) {
	if len(nodes) == 0 {
		return net.DialTimeout("tcp", addr, DialTimeout)
	}

	pc, err := c.travelNodes(nodes...)
	if err != nil {
		return
	}
	if err = pc.Connect(addr); err != nil {
		pc.Close()
		return
	}
	conn = pc
	return
}

func (c *ProxyChain) travelNodes(nodes ...ProxyNode) (conn *ProxyConn, err error) {
	defer func() {
		if err != nil && conn != nil {
			conn.Close()
			conn = nil
		}
	}()

	var cc net.Conn
	node := nodes[0]

	cc, err = net.DialTimeout("tcp", node.Addr, DialTimeout)
	if err != nil {
		return
	}
	setKeepAlive(cc, KeepAliveTime)

	pc := NewProxyConn(cc, node)
	conn = pc
	if err = pc.Handshake(); err != nil {
		return
	}

	for _, node := range nodes[1:] {
		if err = conn.Connect(node.Addr); err != nil {
			return
		}
		pc := NewProxyConn(conn, node)
		conn = pc
		if err = pc.Handshake(); err != nil {
			return
		}
	}
	return
}

func (c *ProxyChain) String() string {
	var buf bytes.Buffer
	for i, n := range c.nodes {
		buf.WriteString(fmt.Sprintf("<%d: %s>", i, n))
	}
	return buf.String()
}
