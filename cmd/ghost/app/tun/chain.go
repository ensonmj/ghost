package tun

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ginuerzh/gosocks5"
	"github.com/pkg/errors"
)

const DialTimeout = 1 * time.Second

type ChainNode interface {
	String() string
	URL() *url.URL
	// First node need Connect
	Connect() (net.Conn, error)
	// Handshake complete authentication with node
	Handshake(c net.Conn) error
	// ForwardRequest ask node to connect to next hop(proxy server or http server)
	ForwardRequest(c net.Conn, url *url.URL) error
}

func HandshakeForHttp(c net.Conn, url *url.URL) error {
	log.Println("handshake with http node")
	return nil
}

func ForwardRequestByHttp(c net.Conn, url *url.URL) error {
	log.Printf("forward request to hop[%s] by HTTP", url.String())
	req := &http.Request{
		Method:     http.MethodConnect,
		URL:        url,
		Host:       url.Host,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
	}
	req.Header.Set("Proxy-Connection", "keep-alive")
	if authStr := basicAuth(url); authStr != "" {
		req.Header.Set("Proxy-Authorization", authStr)
	}

	if err := req.Write(c); err != nil {
		return errors.Wrap(err, "forward request")
	}

	resp, err := http.ReadResponse(bufio.NewReader(c), req)
	if err != nil {
		return errors.Wrap(err, "forward request read response")
	}
	if resp.StatusCode != http.StatusOK {
		resp, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "forward request clear body")
		}
		return errors.New("proxy refused connection" + string(resp))
	}

	return nil
}

func basicAuth(url *url.URL) string {
	user := url.User
	if user != nil {
		s := user.String()
		if _, set := user.Password(); !set {
			s += ":"
		}
		return "Basic " + base64.StdEncoding.EncodeToString([]byte(s))
	}
	return ""
}

func HandshakeForSocks5(c net.Conn, url *url.URL) error {
	log.Println("handshake with socks5 node")
	selector := &clientSelector{
		methods: []uint8{
			gosocks5.MethodNoAuth,
			gosocks5.MethodUserPass,
		},
		user: url.User,
	}
	conn := gosocks5.ClientConn(c, selector)
	if err := conn.Handleshake(); err != nil {
		return errors.Wrap(err, "handleshake")
	}

	return nil
}

func ForwardRequestBySocks5(c net.Conn, url *url.URL) error {
	log.Printf("forward request to hop[%s] by socks5", url.String())
	addr, err := parseAddr(url.Host)
	if err != nil {
		return err
	}
	req := gosocks5.NewRequest(gosocks5.CmdConnect, addr)

	if err := req.Write(c); err != nil {
		return errors.Wrap(err, "forward request")
	}

	resp, err := gosocks5.ReadReply(c)
	if err != nil {
		return errors.Wrap(err, "read socks reply")
	}
	if resp.Rep != gosocks5.Succeeded {
		return errors.New("proxy refused connection")
	}

	return nil
}

type clientSelector struct {
	methods   []uint8
	user      *url.Userinfo
	tlsConfig *tls.Config
}

func (selector *clientSelector) Methods() []uint8 {
	return selector.methods
}

func (selector *clientSelector) Select(methods ...uint8) (method uint8) {
	return
}

func (selector *clientSelector) OnSelected(method uint8, conn net.Conn) (net.Conn, error) {
	switch method {
	case MethodTLS:
		conn = tls.Client(conn, selector.tlsConfig)
	case gosocks5.MethodUserPass, MethodTLSAuth:
		if method == MethodTLSAuth {
			conn = tls.Client(conn, selector.tlsConfig)
		}

		var username, password string
		if selector.user != nil {
			username = selector.user.Username()
			password, _ = selector.user.Password()
		}

		req := gosocks5.NewUserPassRequest(gosocks5.UserPassVer, username, password)
		if err := req.Write(conn); err != nil {
			log.Println("socks5 auth:", err)
			return nil, err
		}
		log.Println(req)

		resp, err := gosocks5.ReadUserPassResponse(conn)
		if err != nil {
			log.Println("socks5 auth:", err)
			return nil, err
		}
		log.Println(resp)

		if resp.Status != gosocks5.Succeeded {
			return nil, gosocks5.ErrAuthFailure
		}
	case gosocks5.MethodNoAcceptable:
		return nil, gosocks5.ErrBadMethod
	}

	return conn, nil
}

// Proxy chain holds a list of proxy nodes
type ProxyChain struct {
	cn   ChainNode
	next *ProxyChain
}

func ParseProxyChain(nodes ...string) (*ProxyChain, error) {
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
			cn = NewHttpChainNode(pn)
		case "socks5":
			cn = NewSocks5ChainNode(pn)
		default:
			return nil, errors.Errorf("unknown scheme:%s", pn.URL.Scheme)
		}

		chain.AddChainNode(cn)
	}
	log.Printf("success to parse chain: %s\n", chain.String())

	return chain, nil
}

func NewProxyChain(pns ...*ProxyNode) (*ProxyChain, error) {
	chain := &ProxyChain{}
	for _, pn := range pns {
		var cn ChainNode
		switch pn.URL.Scheme {
		case "http":
			cn = NewHttpChainNode(pn)
		case "socks5":
			cn = NewSocks5ChainNode(pn)
		default:
			return nil, errors.Errorf("unknown scheme:%s", pn.URL.Scheme)
		}

		chain.AddChainNode(cn)
	}
	log.Printf("success to create chain: %s\n", chain.String())

	return chain, nil
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

func (pc *ProxyChain) Dial(network, addr string) (net.Conn, error) {
	log.Printf("connect to server[%s] with chain[%s]\n", addr, pc)
	if pc == nil {
		// nil chain is also workable
		return net.DialTimeout(network, addr, DialTimeout)
	}

	c, err := pc.connect()
	if err != nil {
		return nil, err
	}

	if !strings.Contains(addr, "://") {
		addr = "http://" + addr
	}
	url, err := url.Parse(addr)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = pc.handshake(c, url)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (pc *ProxyChain) connect() (net.Conn, error) {
	log.Printf("connecting to chain[%s]\n", pc)
	return pc.cn.Connect()
}

func (pc *ProxyChain) handshake(c net.Conn, url *url.URL) error {
	log.Printf("handshaking with chain[%s] for url[%s]\n", pc, url.String())
	pc.cn.Handshake(c)

	if pc.next == nil {
		return pc.cn.ForwardRequest(c, url)
	}

	err := pc.cn.ForwardRequest(c, pc.next.cn.URL())
	if err != nil {
		return err
	}
	return pc.next.handshake(c, pc.next.cn.URL())
}
