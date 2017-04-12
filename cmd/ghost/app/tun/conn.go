package tun

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/ginuerzh/gosocks5"
)

type ProxyConn struct {
	conn           net.Conn
	Node           ProxyNode
	handshaked     bool
	handshakeMutex sync.Mutex
	handshakeErr   error
}

func NewProxyConn(conn net.Conn, node ProxyNode) *ProxyConn {
	return &ProxyConn{
		conn: conn,
		Node: node,
	}
}

// Handshake handshake with this proxy node based on the proxy node info: transport, protocol, authentication, etc.
//
// NOTE: any HTTP2 scheme will be treated as http (for protocol) or tls (for transport).
func (c *ProxyConn) Handshake() error {
	c.handshakeMutex.Lock()
	defer c.handshakeMutex.Unlock()

	if err := c.handshakeErr; err != nil {
		return err
	}
	if c.handshaked {
		return nil
	}
	c.handshakeErr = c.handshake()
	return c.handshakeErr
}

func (c *ProxyConn) handshake() error {
	var tlsUsed bool

	switch c.Node.Transport {
	case "tls", "http2": // tls connection
		tlsUsed = true
		cfg := &tls.Config{
			InsecureSkipVerify: c.Node.insecureSkipVerify(),
			ServerName:         c.Node.serverName,
		}
		c.conn = tls.Client(c.conn, cfg)
	}

	switch c.Node.Protocol {
	case "socks", "socks5": // socks5 handshake with auth and tls supported
		selector := &clientSelector{
			methods: []uint8{
				gosocks5.MethodNoAuth,
				gosocks5.MethodUserPass,
			},
			user: c.Node.User,
		}

		if !tlsUsed { // if transport is not security, enable security socks5
			selector.methods = append(selector.methods, MethodTLS)
			selector.tlsConfig = &tls.Config{
				InsecureSkipVerify: c.Node.insecureSkipVerify(),
				ServerName:         c.Node.serverName,
			}
		}

		conn := gosocks5.ClientConn(c.conn, selector)
		if err := conn.Handleshake(); err != nil {
			return err
		}
		c.conn = conn
	}

	c.handshaked = true

	return nil
}

// Connect connect to addr through this proxy node
func (c *ProxyConn) Connect(addr string) error {
	switch c.Node.Protocol {
	case "socks", "socks5":
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return err
		}
		p, _ := strconv.Atoi(port)
		req := gosocks5.NewRequest(gosocks5.CmdConnect, &gosocks5.Addr{
			Type: gosocks5.AddrDomain,
			Host: host,
			Port: uint16(p),
		})
		if err := req.Write(c); err != nil {
			return err
		}
		log.Println("[socks5]", req)

		reply, err := gosocks5.ReadReply(c)
		if err != nil {
			return err
		}
		log.Println("[socks5]", reply)
		if reply.Rep != gosocks5.Succeeded {
			return errors.New("Service unavailable")
		}
	case "http":
		fallthrough
	default:
		req := &http.Request{
			Method:     http.MethodConnect,
			URL:        &url.URL{Host: addr},
			Host:       addr,
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     make(http.Header),
		}
		req.Header.Set("Proxy-Connection", "keep-alive")
		if c.Node.User != nil {
			user := c.Node.User
			s := user.String()
			if _, set := user.Password(); !set {
				s += ":"
			}
			req.Header.Set("Proxy-Authorization",
				"Basic "+base64.StdEncoding.EncodeToString([]byte(s)))
		}
		if err := req.Write(c); err != nil {
			return err
		}

		resp, err := http.ReadResponse(bufio.NewReader(c), req)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return errors.New(resp.Status)
		}
	}

	return nil
}

func (c *ProxyConn) Read(b []byte) (n int, err error) {
	return c.conn.Read(b)
}

func (c *ProxyConn) Write(b []byte) (n int, err error) {
	return c.conn.Write(b)
}

func (c *ProxyConn) Close() error {
	return c.conn.Close()
}

func (c *ProxyConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *ProxyConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *ProxyConn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *ProxyConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *ProxyConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}
