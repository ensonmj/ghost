package tun

import (
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

type ProxyNode interface {
	Handshake() error
	ListenAndServe() error
}

type CommNode struct {
	Scheme string        // http/https/http2/socks5/tcp/udp/rtcp/rudp/ss/ws/wss
	Addr   string        // [host]:port
	Remote string        // remote address, used by tcp/udp port forwarding
	User   *url.Userinfo // authentication for proxy
	values url.Values
	tls    bool
	// conn   net.Conn
}

// The proxy node string pattern is [scheme://][user:pass@host]:port.
func ParseProxyNode(s string) (ProxyNode, error) {
	if !strings.Contains(s, "://") {
		s = "//" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return nil, errors.Wrap(err, "proxy node parse")
	}

	cn := CommNode{
		Scheme: u.Scheme,
		Addr:   u.Host,
		User:   u.User,
		values: u.Query(),
	}

	switch cn.Scheme {
	// case "socks":
	// 	return NewSocks5Server(cn)
	// case "tcp", "udp":
	// 	// local port forward: -L tcp://:2222/192.168.1.1:22
	// 	cn.Remote = strings.Trim(u.EscapedPath(), "/")
	// case "rtcp", "rudp":
	// 	// remote port forward: -L rtcp://:2222/192.168.1.1:22 -F socks://172.24.10.1:1080
	// 	cn.Remote = strings.Trim(u.EscapedPath(), "/")
	// case "https", "http2":
	// 	cn.tls = true
	case "", "http":
	default:
		return nil, errors.Errorf("Scheme:%s not support\n", cn.Scheme)
	}

	// http as default
	return NewHttpServer(cn), nil
}

// Get get node parameter by key
// func (node *ProxyNode) Get(key string) string {
// 	return node.values.Get(key)
// }

// func (node *ProxyNode) getBool(key string) bool {
// 	s := node.Get(key)
// 	if b, _ := strconv.ParseBool(s); b {
// 		return b
// 	}
// 	n, _ := strconv.Atoi(s)
// 	return n > 0
// }

// func (node *ProxyNode) Set(key, value string) {
// 	node.values.Set(key, value)
// }

// func (node *ProxyNode) insecureSkipVerify() bool {
// 	return !node.getBool("secure")
// }

// func (node ProxyNode) String() string {
// 	return fmt.Sprintf("transport: %s, protocol: %s, addr: %s",
// 		node.Transport, node.Scheme, node.Addr)
// }
