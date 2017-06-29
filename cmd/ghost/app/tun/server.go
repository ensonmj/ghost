package tun

import (
	"crypto/tls"
	"log"
)

type LocalNode interface {
	ListenAndServe() error
}

// ----------------------------------------------------------------------------
// data stream
// ----------------------------------------------------------------------------
//                                    ______(chain)______
//                                   |                   |
// (cli) -<tcp/udp>- (proxy) -<tcp>- | - node ... node - | -<tcp/udp>- (app)
//                                   |___________________|
//
// ----------------------------------------------------------------------------
type ProxyServer struct {
	ln        LocalNode
	pc        *ProxyChain
	TLSConfig *tls.Config
}

func NewProxyServer(pn *ProxyNode, pc *ProxyChain, config *tls.Config) *ProxyServer {
	if config == nil {
		config = &tls.Config{}
	}

	var n LocalNode
	switch pn.URL.Scheme {
	// case "tcp", "udp":
	// 	// local port forward: -L tcp://:2222/192.168.1.1:22
	// 	cn.Remote = strings.Trim(u.EscapedPath(), "/")
	// case "rtcp", "rudp":
	// 	// remote port forward: -L rtcp://:2222/192.168.1.1:22 -F socks://172.24.10.1:1080
	// 	cn.Remote = strings.Trim(u.EscapedPath(), "/")
	case "http":
		n = NewHttpServer(pn, pc)
	case "socks5":
		n = NewSocks5Server(pn, pc)
	}

	return &ProxyServer{
		ln:        n,
		pc:        pc,
		TLSConfig: config,
	}
}

func (s *ProxyServer) ListenAndServe() error {
	log.Printf("proxy server starting: %s\n", s.ln)
	return s.ln.ListenAndServe()
}
