package tun

import (
	"crypto/tls"
	"log"
)

type LocalNode interface {
	ListenAndServe(*ProxyChain) error
}

type ProxyServer struct {
	Node      LocalNode
	Chain     *ProxyChain
	TLSConfig *tls.Config
}

// func NewProxyServer(node *ProxyNode, config *tls.Config) *ProxyServer {
func NewProxyServer(pn *ProxyNode, pc *ProxyChain, config *tls.Config) *ProxyServer {
	if config == nil {
		config = &tls.Config{}
	}

	var n LocalNode
	switch pn.URL.Scheme {
	// case "socks":
	// 	return NewSocks5Server(cn)
	// case "tcp", "udp":
	// 	// local port forward: -L tcp://:2222/192.168.1.1:22
	// 	cn.Remote = strings.Trim(u.EscapedPath(), "/")
	// case "rtcp", "rudp":
	// 	// remote port forward: -L rtcp://:2222/192.168.1.1:22 -F socks://172.24.10.1:1080
	// 	cn.Remote = strings.Trim(u.EscapedPath(), "/")
	case "http":
		n = NewHttpNode(pn)
	}

	return &ProxyServer{
		Node:      n,
		Chain:     pc,
		TLSConfig: config,
	}
}

func (s *ProxyServer) ListenAndServe() error {
	log.Printf("proxy server starting: %s\n", s.Node)
	return s.Node.ListenAndServe(s.Chain)
}
