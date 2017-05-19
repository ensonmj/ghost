package tun

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/url"
	"strconv"

	"github.com/ginuerzh/gosocks5"
	"github.com/pkg/errors"
)

// var (
// 	ErrEmptyChain = errors.New("empty chain")
// )
// var (
// 	SmallBufferSize  = 1 * 1024  // 1KB small buffer
// 	MediumBufferSize = 8 * 1024  // 8KB medium buffer
// 	LargeBufferSize  = 32 * 1024 // 32KB large buffer
// )

const (
	MethodTLS     uint8 = 0x80 // extended method for tls
	MethodTLSAuth uint8 = 0x82 // extended method for tls+auth
)

// const (
// 	CmdUdpTun uint8 = 0xF3 // extended method for udp over tcp
// )

// type clientSelector struct {
// 	methods   []uint8
// 	user      *url.Userinfo
// 	tlsConfig *tls.Config
// }

// func (selector *clientSelector) Methods() []uint8 {
// 	return selector.methods
// }

// func (selector *clientSelector) Select(methods ...uint8) (method uint8) {
// 	return
// }

// func (selector *clientSelector) OnSelected(method uint8, conn net.Conn) (net.Conn, error) {
// 	switch method {
// 	case MethodTLS:
// 		conn = tls.Client(conn, selector.tlsConfig)

// 	case gosocks5.MethodUserPass, MethodTLSAuth:
// 		if method == MethodTLSAuth {
// 			conn = tls.Client(conn, selector.tlsConfig)
// 		}

// 		var username, password string
// 		if selector.user != nil {
// 			username = selector.user.Username()
// 			password, _ = selector.user.Password()
// 		}

// 		req := gosocks5.NewUserPassRequest(gosocks5.UserPassVer, username, password)
// 		if err := req.Write(conn); err != nil {
// 			log.Println("socks5 auth:", err)
// 			return nil, err
// 		}
// 		log.Println(req)

// 		resp, err := gosocks5.ReadUserPassResponse(conn)
// 		if err != nil {
// 			log.Println("socks5 auth:", err)
// 			return nil, err
// 		}
// 		log.Println(resp)

// 		if resp.Status != gosocks5.Succeeded {
// 			return nil, gosocks5.ErrAuthFailure
// 		}
// 	case gosocks5.MethodNoAcceptable:
// 		return nil, gosocks5.ErrBadMethod
// 	}

// 	return conn, nil
// }

type serverSelector struct {
	methods   []uint8
	user      *url.Userinfo
	tlsConfig *tls.Config
}

func (selector *serverSelector) Methods() []uint8 {
	return selector.methods
}

func (selector *serverSelector) Select(methods ...uint8) (method uint8) {
	log.Printf("%d %d %v\n", gosocks5.Ver5, len(methods), methods)

	method = gosocks5.MethodNoAuth
	for _, m := range methods {
		if m == MethodTLS {
			method = m
			break
		}
	}

	// when user/pass is set, auth is mandatory
	if selector.user != nil {
		if method == gosocks5.MethodNoAuth {
			method = gosocks5.MethodUserPass
		}
		if method == MethodTLS {
			method = MethodTLSAuth
		}
	}

	return
}

func (selector *serverSelector) OnSelected(method uint8, conn net.Conn) (net.Conn, error) {
	log.Printf("%d %d\n", gosocks5.Ver5, method)

	switch method {
	case MethodTLS:
		conn = tls.Server(conn, selector.tlsConfig)
	case gosocks5.MethodUserPass, MethodTLSAuth:
		if method == MethodTLSAuth {
			conn = tls.Server(conn, selector.tlsConfig)
		}

		req, err := gosocks5.ReadUserPassRequest(conn)
		if err != nil {
			log.Println("[socks5-auth]", err)
			return nil, err
		}
		log.Println("[socks5]", req.String())

		valid := false
		user := selector.user
		if user != nil {
			username := user.Username()
			password, _ := user.Password()
			if (req.Username == username && req.Password == password) ||
				(req.Username == username && password == "") ||
				(username == "" && req.Password == password) {
				valid = true
			}

			if !valid {
				resp := gosocks5.NewUserPassResponse(gosocks5.UserPassVer, gosocks5.Failure)
				if err := resp.Write(conn); err != nil {
					log.Println("[socks5-auth]", err)
					return nil, err
				}
				log.Println("[socks5]", resp)
				log.Println("[socks5-auth] proxy authentication required")

				return nil, gosocks5.ErrAuthFailure
			}
		}

		resp := gosocks5.NewUserPassResponse(gosocks5.UserPassVer, gosocks5.Succeeded)
		if err := resp.Write(conn); err != nil {
			log.Println("[socks5-auth]", err)
			return nil, err
		}
		log.Println(resp)
	case gosocks5.MethodNoAcceptable:
		return nil, gosocks5.ErrBadMethod
	}

	return conn, nil
}

type Socks5Server struct {
	pn       *ProxyNode
	pc       *ProxyChain
	selector *serverSelector
}

func NewSocks5Server(pn *ProxyNode) *Socks5Server {
	return &Socks5Server{
		pn: pn,
		selector: &serverSelector{
			methods: []uint8{
				gosocks5.MethodNoAuth,
				gosocks5.MethodUserPass,
				MethodTLS,
				MethodTLSAuth,
			},
			// user:      base.Node.User,
			// tlsConfig: base.TLSConfig,
		},
	}
}

func (n *Socks5Server) ListenAndServe(pc *ProxyChain) {
	n.pc = pc
	ln := n.Listen()
	n.Serve(ln)
}

func (n *Socks5Server) Listen() net.Listener {
	ln, err := net.Listen("tcp", n.pn.URL.Host)
	if err != nil {
		panic(errors.Wrap(err, "socks server listen"))
	}
	return ln
}

func (n *Socks5Server) Serve(ln net.Listener) {
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		go func() {
			conn := gosocks5.ServerConn(conn, n.selector)
			req, err := gosocks5.ReadRequest(conn)
			if err != nil {
				log.Printf("[socks5]: %s\n", err)
				return
			}
			log.Printf("[socks5] %s -> %s\n%s\n", conn.RemoteAddr(), req.Addr, req)

			switch req.Cmd {
			case gosocks5.CmdConnect:
				log.Printf("[socks5-connect] %s -> %s\n", conn.RemoteAddr(), req.Addr)
				n.handleConnect(conn, req)
			// case gosocks5.CmdBind:
			// 	log.Printf("[socks5-bind] %s - %s\n", conn.RemoteAddr(), req.Addr)
			// 	n.handleBind(req)
			// case gosocks5.CmdUdp:
			// 	log.Printf("[socks5-udp] %s - %s\n", conn.RemoteAddr(), req.Addr)
			// 	n.handleUDPRelay(req)
			// case CmdUdpTun:
			// 	log.Printf("[socks5-udp] %s - %s\n", conn.RemoteAddr(), req.Addr)
			// 	n.handleUDPTunnel(req)
			default:
				log.Println("[socks5] Unrecognized request:", req.Cmd)
			}
			return
		}()
	}
}

// Dial server or chain proxy
func (n *Socks5Server) Dial(network, addr string) (net.Conn, error) {
	return n.pc.Dial(network, addr)
}

func (n *Socks5Server) GetProxyNode() *ProxyNode {
	return n.pn
}

func (n *Socks5Server) DialIn() (net.Conn, error) {
	log.Printf("dial to chain node: %s\n", n)
	c, err := net.Dial("tcp", n.pn.URL.Host)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	return c, nil
}

func (n *Socks5Server) DialOut(c net.Conn, addr string) (net.Conn, error) {
	log.Printf("handshake with chain node: %s\n", n)
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, errors.Wrap(err, "parse addr")
	}
	p, _ := strconv.Atoi(port)
	req := gosocks5.NewRequest(gosocks5.CmdConnect, &gosocks5.Addr{
		Type: gosocks5.AddrDomain,
		Host: host,
		Port: uint16(p),
	})
	if err := req.Write(c); err != nil {
		return nil, errors.Wrap(err, "write socks connect")
	}

	resp, err := gosocks5.ReadReply(c)
	if err != nil {
		return nil, errors.Wrap(err, "read socks reply")
	}
	if resp.Rep != gosocks5.Succeeded {
		return nil, errors.New("proxy refused connection")
	}

	return c, nil
}

func (n *Socks5Server) handleConnect(c net.Conn, req *gosocks5.Request) {
	cc, err := n.Dial("tcp", req.Addr.String())
	if err != nil {
		log.Printf("[socks5-connect] %s -> %s : %s\n", c.RemoteAddr(), req.Addr, err)
		rep := gosocks5.NewReply(gosocks5.HostUnreachable, nil)
		rep.Write(c)
		log.Printf("[socks5-connect] %s <- %s\n%s\n", c.RemoteAddr(), req.Addr, rep)
		return
	}
	defer cc.Close()

	rep := gosocks5.NewReply(gosocks5.Succeeded, nil)
	if err := rep.Write(c); err != nil {
		log.Printf("[socks5-connect] %s <- %s : %s\n", c.RemoteAddr(), req.Addr, err)
		return
	}
	log.Printf("[socks5-connect] %s <- %s\n%s\n", c.RemoteAddr(), req.Addr, rep)

	log.Printf("[socks5-connect] %s <-> %s\n", c.RemoteAddr(), req.Addr)
	Connect(cc, c)
	log.Printf("[socks5-connect] %s >-< %s\n", c.RemoteAddr(), req.Addr)
}

func Connect(c1, c2 net.Conn) {
	errCh := make(chan error, 2)

	go func() {
		_, err := io.Copy(c1, c2)
		errCh <- err
	}()
	go func() {
		_, err := io.Copy(c2, c1)
		errCh <- err
	}()
	select {
	case err := <-errCh:
		if err != nil {
			log.Printf("[proxy-connect]: copy data err: %s", err.Error())
		}
	}
	return
}

// func (s *Socks5Server) handleBind(req *gosocks5.Request) {
// 	cc, err := s.Base.Chain.GetConn()

// 	// connection error
// 	if err != nil && err != ErrEmptyChain {
// 		log.Printf("[socks5-bind] %s <- %s : %s\n", s.conn.RemoteAddr(), req.Addr, err)
// 		reply := gosocks5.NewReply(gosocks5.Failure, nil)
// 		reply.Write(s.conn)
// 		log.Printf("[socks5-bind] %s <- %s\n%s\n", s.conn.RemoteAddr(), req.Addr, reply)
// 		return
// 	}
// 	// serve socks5 bind
// 	if err == ErrEmptyChain {
// 		s.bindOn(req.Addr.String())
// 		return
// 	}

// 	defer cc.Close()
// 	// forward request
// 	req.Write(cc)

// 	log.Printf("[socks5-bind] %s <-> %s\n", s.conn.RemoteAddr(), cc.RemoteAddr())
// 	s.Base.transport(s.conn, cc)
// 	log.Printf("[socks5-bind] %s >-< %s\n", s.conn.RemoteAddr(), cc.RemoteAddr())
// }

// func (s *Socks5Server) handleUDPRelay(req *gosocks5.Request) {
// 	bindAddr, _ := net.ResolveUDPAddr("udp", req.Addr.String())
// 	relay, err := net.ListenUDP("udp", bindAddr) // udp associate, strict mode: if the port already in use, it will return error
// 	if err != nil {
// 		log.Printf("[socks5-udp] %s -> %s : %s\n", s.conn.RemoteAddr(), req.Addr, err)
// 		reply := gosocks5.NewReply(gosocks5.Failure, nil)
// 		reply.Write(s.conn)
// 		log.Printf("[socks5-udp] %s <- %s\n%s\n", s.conn.RemoteAddr(), req.Addr, reply)
// 		return
// 	}
// 	defer relay.Close()

// 	socksAddr := ToSocksAddr(relay.LocalAddr())
// 	socksAddr.Host, _, _ = net.SplitHostPort(s.conn.LocalAddr().String())
// 	reply := gosocks5.NewReply(gosocks5.Succeeded, socksAddr)
// 	if err := reply.Write(s.conn); err != nil {
// 		log.Printf("[socks5-udp] %s <- %s : %s\n", s.conn.RemoteAddr(), req.Addr, err)
// 		return
// 	}
// 	log.Printf("[socks5-udp] %s <- %s\n%s\n", s.conn.RemoteAddr(), reply.Addr, reply)
// 	log.Printf("[socks5-udp] %s - %s BIND ON %s OK\n", s.conn.RemoteAddr(), req.Addr, socksAddr)

// 	cc, err := s.Base.Chain.GetConn()
// 	// connection error
// 	if err != nil && err != ErrEmptyChain {
// 		log.Printf("[socks5-udp] %s -> %s : %s\n", s.conn.RemoteAddr(), socksAddr, err)
// 		return
// 	}

// 	// serve as standard socks5 udp relay local <-> remote
// 	if err == ErrEmptyChain {
// 		peer, er := net.ListenUDP("udp", nil)
// 		if er != nil {
// 			log.Printf("[socks5-udp] %s -> %s : %s\n", s.conn.RemoteAddr(), socksAddr, er)
// 			return
// 		}
// 		defer peer.Close()

// 		go s.transportUDP(relay, peer)
// 	}

// 	// forward udp local <-> tunnel
// 	if err == nil {
// 		defer cc.Close()

// 		cc.SetWriteDeadline(time.Now().Add(WriteTimeout))
// 		req := gosocks5.NewRequest(CmdUdpTun, nil)
// 		if err := req.Write(cc); err != nil {
// 			log.Printf("[socks5-udp] %s -> %s : %s\n", s.conn.RemoteAddr(), cc.RemoteAddr(), err)
// 			return
// 		}
// 		cc.SetWriteDeadline(time.Time{})
// 		log.Printf("[socks5-udp] %s -> %s\n%s\n", s.conn.RemoteAddr(), cc.RemoteAddr(), req)

// 		cc.SetReadDeadline(time.Now().Add(ReadTimeout))
// 		reply, err = gosocks5.ReadReply(cc)
// 		if err != nil {
// 			log.Printf("[socks5-udp] %s -> %s : %s\n", s.conn.RemoteAddr(), cc.RemoteAddr(), err)
// 			return
// 		}
// 		log.Printf("[socks5-udp] %s <- %s\n%s\n", s.conn.RemoteAddr(), cc.RemoteAddr(), reply)

// 		if reply.Rep != gosocks5.Succeeded {
// 			log.Printf("[socks5-udp] %s <- %s : udp associate failed\n", s.conn.RemoteAddr(), cc.RemoteAddr())
// 			return
// 		}
// 		cc.SetReadDeadline(time.Time{})
// 		log.Printf("[socks5-udp] %s <-> %s [tun: %s]\n", s.conn.RemoteAddr(), socksAddr, reply.Addr)

// 		go s.tunnelClientUDP(relay, cc)
// 	}

// 	log.Printf("[socks5-udp] %s <-> %s\n", s.conn.RemoteAddr(), socksAddr)
// 	b := make([]byte, SmallBufferSize)
// 	for {
// 		_, err := s.conn.Read(b) // discard any data from tcp connection
// 		if err != nil {
// 			log.Printf("[socks5-udp] %s - %s : %s\n", s.conn.RemoteAddr(), socksAddr, err)
// 			break // client disconnected
// 		}
// 	}
// 	log.Printf("[socks5-udp] %s >-< %s\n", s.conn.RemoteAddr(), socksAddr)
// }

// func (s *Socks5Server) handleUDPTunnel(req *gosocks5.Request) {
// 	cc, err := s.Base.Chain.GetConn()

// 	// connection error
// 	if err != nil && err != ErrEmptyChain {
// 		log.Printf("[socks5-udp] %s -> %s : %s\n", s.conn.RemoteAddr(), req.Addr, err)
// 		reply := gosocks5.NewReply(gosocks5.Failure, nil)
// 		reply.Write(s.conn)
// 		log.Printf("[socks5-udp] %s -> %s\n%s\n", s.conn.RemoteAddr(), req.Addr, reply)
// 		return
// 	}

// 	// serve tunnel udp, tunnel <-> remote, handle tunnel udp request
// 	if err == ErrEmptyChain {
// 		bindAddr, _ := net.ResolveUDPAddr("udp", req.Addr.String())
// 		uc, err := net.ListenUDP("udp", bindAddr)
// 		if err != nil {
// 			log.Printf("[socks5-udp] %s -> %s : %s\n", s.conn.RemoteAddr(), req.Addr, err)
// 			return
// 		}
// 		defer uc.Close()

// 		socksAddr := ToSocksAddr(uc.LocalAddr())
// 		socksAddr.Host, _, _ = net.SplitHostPort(s.conn.LocalAddr().String())
// 		reply := gosocks5.NewReply(gosocks5.Succeeded, socksAddr)
// 		if err := reply.Write(s.conn); err != nil {
// 			log.Printf("[socks5-udp] %s <- %s : %s\n", s.conn.RemoteAddr(), socksAddr, err)
// 			return
// 		}
// 		log.Printf("[socks5-udp] %s <- %s\n%s\n", s.conn.RemoteAddr(), socksAddr, reply)

// 		log.Printf("[socks5-udp] %s <-> %s\n", s.conn.RemoteAddr(), socksAddr)
// 		s.tunnelServerUDP(s.conn, uc)
// 		log.Printf("[socks5-udp] %s >-< %s\n", s.conn.RemoteAddr(), socksAddr)
// 		return
// 	}

// 	defer cc.Close()

// 	// tunnel <-> tunnel, direct forwarding
// 	req.Write(cc)

// 	log.Printf("[socks5-udp] %s <-> %s [tun]\n", s.conn.RemoteAddr(), cc.RemoteAddr())
// 	s.Base.transport(s.conn, cc)
// 	log.Printf("[socks5-udp] %s >-< %s [tun]\n", s.conn.RemoteAddr(), cc.RemoteAddr())
// }

// func (s *Socks5Server) bindOn(addr string) {
// 	bindAddr, _ := net.ResolveTCPAddr("tcp", addr)
// 	ln, err := net.ListenTCP("tcp", bindAddr) // strict mode: if the port already in use, it will return error
// 	if err != nil {
// 		log.Printf("[socks5-bind] %s -> %s : %s\n", s.conn.RemoteAddr(), addr, err)
// 		gosocks5.NewReply(gosocks5.Failure, nil).Write(s.conn)
// 		return
// 	}

// 	socksAddr := ToSocksAddr(ln.Addr())
// 	// Issue: may not reachable when host has multi-interface
// 	socksAddr.Host, _, _ = net.SplitHostPort(s.conn.LocalAddr().String())
// 	reply := gosocks5.NewReply(gosocks5.Succeeded, socksAddr)
// 	if err := reply.Write(s.conn); err != nil {
// 		log.Printf("[socks5-bind] %s <- %s : %s\n", s.conn.RemoteAddr(), addr, err)
// 		ln.Close()
// 		return
// 	}
// 	log.Printf("[socks5-bind] %s <- %s\n%s\n", s.conn.RemoteAddr(), addr, reply)
// 	log.Printf("[socks5-bind] %s - %s BIND ON %s OK\n", s.conn.RemoteAddr(), addr, socksAddr)

// 	var pconn net.Conn
// 	accept := func() <-chan error {
// 		errc := make(chan error, 1)

// 		go func() {
// 			defer close(errc)
// 			defer ln.Close()

// 			c, err := ln.AcceptTCP()
// 			if err != nil {
// 				errc <- err
// 				return
// 			}
// 			pconn = c
// 		}()

// 		return errc
// 	}

// 	pc1, pc2 := net.Pipe()
// 	pipe := func() <-chan error {
// 		errc := make(chan error, 1)

// 		go func() {
// 			defer close(errc)
// 			defer pc1.Close()

// 			errc <- s.Base.transport(s.conn, pc1)
// 		}()

// 		return errc
// 	}

// 	defer pc2.Close()

// 	for {
// 		select {
// 		case err := <-accept():
// 			if err != nil || pconn == nil {
// 				log.Printf("[socks5-bind] %s <- %s : %s\n", s.conn.RemoteAddr(), addr, err)
// 				return
// 			}
// 			defer pconn.Close()

// 			reply := gosocks5.NewReply(gosocks5.Succeeded, ToSocksAddr(pconn.RemoteAddr()))
// 			if err := reply.Write(pc2); err != nil {
// 				log.Printf("[socks5-bind] %s <- %s : %s\n", s.conn.RemoteAddr(), addr, err)
// 			}
// 			log.Printf("[socks5-bind] %s <- %s\n%s\n", s.conn.RemoteAddr(), addr, reply)
// 			log.Printf("[socks5-bind] %s <- %s PEER %s ACCEPTED\n", s.conn.RemoteAddr(), socksAddr, pconn.RemoteAddr())

// 			log.Printf("[socks5-bind] %s <-> %s\n", s.conn.RemoteAddr(), pconn.RemoteAddr())
// 			if err = s.Base.transport(pc2, pconn); err != nil {
// 				log.Println(err)
// 			}
// 			log.Printf("[socks5-bind] %s >-< %s\n", s.conn.RemoteAddr(), pconn.RemoteAddr())
// 			return
// 		case err := <-pipe():
// 			log.Printf("[socks5-bind] %s -> %s : %v\n", s.conn.RemoteAddr(), addr, err)
// 			ln.Close()
// 			return
// 		}
// 	}
// }

// func (s *Socks5Server) transportUDP(relay, peer *net.UDPConn) (err error) {
// 	errc := make(chan error, 2)

// 	var clientAddr *net.UDPAddr

// 	go func() {
// 		b := make([]byte, LargeBufferSize)

// 		for {
// 			n, laddr, err := relay.ReadFromUDP(b)
// 			if err != nil {
// 				errc <- err
// 				return
// 			}
// 			if clientAddr == nil {
// 				clientAddr = laddr
// 			}
// 			dgram, err := gosocks5.ReadUDPDatagram(bytes.NewReader(b[:n]))
// 			if err != nil {
// 				errc <- err
// 				return
// 			}

// 			raddr, err := net.ResolveUDPAddr("udp", dgram.Header.Addr.String())
// 			if err != nil {
// 				continue // drop silently
// 			}
// 			if _, err := peer.WriteToUDP(dgram.Data, raddr); err != nil {
// 				errc <- err
// 				return
// 			}
// 			log.Printf("[socks5-udp] %s >>> %s length: %d\n", relay.LocalAddr(), raddr, len(dgram.Data))
// 		}
// 	}()

// 	go func() {
// 		b := make([]byte, LargeBufferSize)

// 		for {
// 			n, raddr, err := peer.ReadFromUDP(b)
// 			if err != nil {
// 				errc <- err
// 				return
// 			}
// 			if clientAddr == nil {
// 				continue
// 			}
// 			buf := bytes.Buffer{}
// 			dgram := gosocks5.NewUDPDatagram(gosocks5.NewUDPHeader(0, 0, ToSocksAddr(raddr)), b[:n])
// 			dgram.Write(&buf)
// 			if _, err := relay.WriteToUDP(buf.Bytes(), clientAddr); err != nil {
// 				errc <- err
// 				return
// 			}
// 			log.Printf("[socks5-udp] %s <<< %s length: %d\n", relay.LocalAddr(), raddr, len(dgram.Data))
// 		}
// 	}()

// 	select {
// 	case err = <-errc:
// 		//log.Println("w exit", err)
// 	}

// 	return
// }

// func (s *Socks5Server) tunnelClientUDP(uc *net.UDPConn, cc net.Conn) (err error) {
// 	errc := make(chan error, 2)

// 	var clientAddr *net.UDPAddr

// 	go func() {
// 		b := make([]byte, LargeBufferSize)

// 		for {
// 			n, addr, err := uc.ReadFromUDP(b)
// 			if err != nil {
// 				log.Printf("[udp-tun] %s <- %s : %s\n", cc.RemoteAddr(), addr, err)
// 				errc <- err
// 				return
// 			}

// 			// glog.V(LDEBUG).Infof("read udp %d, % #x", n, b[:n])
// 			// pipe from relay to tunnel
// 			dgram, err := gosocks5.ReadUDPDatagram(bytes.NewReader(b[:n]))
// 			if err != nil {
// 				errc <- err
// 				return
// 			}
// 			if clientAddr == nil {
// 				clientAddr = addr
// 			}
// 			dgram.Header.Rsv = uint16(len(dgram.Data))
// 			if err := dgram.Write(cc); err != nil {
// 				errc <- err
// 				return
// 			}
// 			log.Printf("[udp-tun] %s >>> %s length: %d\n", uc.LocalAddr(), dgram.Header.Addr, len(dgram.Data))
// 		}
// 	}()

// 	go func() {
// 		for {
// 			dgram, err := gosocks5.ReadUDPDatagram(cc)
// 			if err != nil {
// 				log.Printf("[udp-tun] %s -> 0 : %s\n", cc.RemoteAddr(), err)
// 				errc <- err
// 				return
// 			}

// 			// pipe from tunnel to relay
// 			if clientAddr == nil {
// 				continue
// 			}
// 			dgram.Header.Rsv = 0

// 			buf := bytes.Buffer{}
// 			dgram.Write(&buf)
// 			if _, err := uc.WriteToUDP(buf.Bytes(), clientAddr); err != nil {
// 				errc <- err
// 				return
// 			}
// 			log.Printf("[udp-tun] %s <<< %s length: %d\n", uc.LocalAddr(), dgram.Header.Addr, len(dgram.Data))
// 		}
// 	}()

// 	select {
// 	case err = <-errc:
// 	}

// 	return
// }

// func (s *Socks5Server) tunnelServerUDP(cc net.Conn, uc *net.UDPConn) (err error) {
// 	errc := make(chan error, 2)

// 	go func() {
// 		b := make([]byte, LargeBufferSize)

// 		for {
// 			n, addr, err := uc.ReadFromUDP(b)
// 			if err != nil {
// 				log.Printf("[udp-tun] %s <- %s : %s\n", cc.RemoteAddr(), addr, err)
// 				errc <- err
// 				return
// 			}

// 			// pipe from peer to tunnel
// 			dgram := gosocks5.NewUDPDatagram(
// 				gosocks5.NewUDPHeader(uint16(n), 0, ToSocksAddr(addr)), b[:n])
// 			if err := dgram.Write(cc); err != nil {
// 				log.Printf("[udp-tun] %s <- %s : %s\n", cc.RemoteAddr(), dgram.Header.Addr, err)
// 				errc <- err
// 				return
// 			}
// 			log.Printf("[udp-tun] %s <<< %s length: %d\n", cc.RemoteAddr(), dgram.Header.Addr, len(dgram.Data))
// 		}
// 	}()

// 	go func() {
// 		for {
// 			dgram, err := gosocks5.ReadUDPDatagram(cc)
// 			if err != nil {
// 				log.Printf("[udp-tun] %s -> 0 : %s\n", cc.RemoteAddr(), err)
// 				errc <- err
// 				return
// 			}

// 			// pipe from tunnel to peer
// 			addr, err := net.ResolveUDPAddr("udp", dgram.Header.Addr.String())
// 			if err != nil {
// 				continue // drop silently
// 			}
// 			if _, err := uc.WriteToUDP(dgram.Data, addr); err != nil {
// 				log.Printf("[udp-tun] %s -> %s : %s\n", cc.RemoteAddr(), addr, err)
// 				errc <- err
// 				return
// 			}
// 			log.Printf("[udp-tun] %s >>> %s length: %d\n", cc.RemoteAddr(), addr, len(dgram.Data))
// 		}
// 	}()

// 	select {
// 	case err = <-errc:
// 	}

// 	return
// }

// func ToSocksAddr(addr net.Addr) *gosocks5.Addr {
// 	host := "0.0.0.0"
// 	port := 0
// 	if addr != nil {
// 		h, p, _ := net.SplitHostPort(addr.String())
// 		host = h
// 		port, _ = strconv.Atoi(p)
// 	}
// 	return &gosocks5.Addr{
// 		Type: gosocks5.AddrIPv4,
// 		Host: host,
// 		Port: uint16(port),
// 	}
// }
