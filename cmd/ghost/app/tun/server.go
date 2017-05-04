package tun

import (
	"crypto/tls"
)

type ProxyServer struct {
	Node ProxyNode
	// Chain     *ProxyChain
	TLSConfig *tls.Config
}

// func NewProxyServer(node ProxyNode, chain *ProxyChain, config *tls.Config) *ProxyServer {
func NewProxyServer(node ProxyNode, config *tls.Config) *ProxyServer {
	// if chain == nil {
	// 	chain, _ = NewProxyChain()
	// }
	if config == nil {
		config = &tls.Config{}
	}

	return &ProxyServer{
		Node: node,
		// Chain:     chain,
		TLSConfig: config,
	}
}

func (s *ProxyServer) ListenAndServe() error {
	// log.Printf("proxy server starting: node[%s], chain[%s]\n", s.Node, s.Chain)
	// return s.transportMux()
	return s.Node.ListenAndServe()
}

// func (s *ProxyServer) transportMux() error {
// 	node := s.Node

// 	switch node.Transport {
// 	case "http2": // Standard HTTP2 proxy server, compatible with HTTP1.x.
// 		server := NewHttp2Server(s)
// 		server.Handler = http.HandlerFunc(server.HandleRequest)
// 		return server.ListenAndServeTLS(s.TLSConfig)
// 	case "tcp": // Local TCP port forwarding
// 		return NewTcpForwardServer(s).ListenAndServe()
// 	case "udp": // Local UDP port forwarding
// 		ttl, _ := strconv.Atoi(node.Get("ttl"))
// 		if ttl <= 0 {
// 			ttl = 600
// 		}
// 		return NewUdpForwardServer(s, ttl).ListenAndServe()
// 	case "tls": // tls connection
// 		return s.protocolMux(true)
// 	default:
// 		return s.protocolMux(false)
// 	}
// }

// func (s *ProxyServer) protocolMux(useTLS bool) error {
// 	var ln net.Listener
// 	var err error
// 	if useTLS {
// 		ln, err = tls.Listen("tcp", s.Node.Addr, s.TLSConfig)
// 	} else {
// 		ln, err = net.Listen("tcp", s.Node.Addr)
// 	}
// 	if err != nil {
// 		return errors.WithStack(err)
// 	}
// 	defer ln.Close()

// 	for {
// 		conn, err := ln.Accept()
// 		if err != nil {
// 			log.Println(errors.WithStack(err))
// 			continue
// 		}

// 		setKeepAlive(conn, KeepAliveTime)

// 		go s.handleConn(conn)
// 	}

// 	return nil
// }

// func (s *ProxyServer) handleConn(conn net.Conn) {
// 	defer conn.Close()

// 	switch s.Node.Scheme {
// 	case "socks", "socks5":
// 		NewSocks5Server(conn, s).Serve()
// 		return
// 	case "http":
// 		NewHttpServer(conn, s).Serve()
// 		return
// 	}

// 	// http or socks5
// 	b := make([]byte, SmallBufferSize)

// 	n, err := io.ReadAtLeast(conn, b, 1)
// 	if err != nil {
// 		log.Println("failed to read for guess protocol: ", err)
// 		return
// 	}

// 	// socks5
// 	if b[0] == gosocks5.Ver5 {
// 		NewSocks5Server(&fakeConn{buf: b[:n], conn: conn}, s).Serve()
// 		return
// 	}

// 	//http
// 	NewHttpServer(&fakeConn{buf: b[:n], conn: conn}, s).Serve()
// }

// func (_ *ProxyServer) transport(conn1, conn2 net.Conn) (err error) {
// 	errc := make(chan error, 2)

// 	go func() {
// 		_, err := io.Copy(conn1, conn2)
// 		errc <- err
// 	}()

// 	go func() {
// 		_, err := io.Copy(conn2, conn1)
// 		errc <- err
// 	}()

// 	select {
// 	case err = <-errc:
// 		//glog.V(LoadCertificate(certFile, keyFile string) (tls.Certificate, error) {
// 	}
// 	return
// }

// func setKeepAlive(conn net.Conn, d time.Duration) error {
// 	c, ok := conn.(*net.TCPConn)
// 	if !ok {
// 		return errors.New("Not a TCP connection")
// 	}
// 	if err := c.SetKeepAlive(true); err != nil {
// 		return err
// 	}
// 	if err := c.SetKeepAlivePeriod(d); err != nil {
// 		return err
// 	}
// 	return nil
// }

// type fakeConn struct {
// 	buf  []byte
// 	conn net.Conn
// }

// func (c *fakeConn) Read(p []byte) (n int, err error) {
// 	if len(c.buf) == 0 {
// 		return c.conn.Read(p)
// 	}
// 	n = copy(p, c.buf)
// 	c.buf = c.buf[n:]

// 	return
// }

// func (c *fakeConn) Write(b []byte) (n int, err error) {
// 	return c.conn.Write(b)
// }

// func (c *fakeConn) Close() error {
// 	return c.conn.Close()
// }

// func (c *fakeConn) LocalAddr() net.Addr {
// 	return c.conn.LocalAddr()
// }

// func (c *fakeConn) RemoteAddr() net.Addr {
// 	return c.conn.RemoteAddr()
// }

// func (c *fakeConn) SetDeadline(t time.Time) error {
// 	return c.conn.SetDeadline(t)
// }

// func (c *fakeConn) SetReadDeadline(t time.Time) error {
// 	return c.conn.SetReadDeadline(t)
// }

// func (c *fakeConn) SetWriteDeadline(t time.Time) error {
// 	return c.conn.SetWriteDeadline(t)
// }
