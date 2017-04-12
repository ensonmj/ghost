package tun

import (
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/ginuerzh/gosocks5"
)

const WriteTimeout = 1 * time.Second
const ReadTimeout = 1 * time.Second

type TcpForwardServer struct {
	Base    *ProxyServer
	Handler func(conn net.Conn, raddr net.Addr)
}

func NewTcpForwardServer(base *ProxyServer) *TcpForwardServer {
	return &TcpForwardServer{Base: base}
}

func (s *TcpForwardServer) ListenAndServe() error {
	raddr, err := net.ResolveTCPAddr("tcp", s.Base.Node.Remote)
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", s.Base.Node.Addr)
	if err != nil {
		return err
	}
	defer ln.Close()

	if s.Handler == nil {
		s.Handler = s.handleTcpForward
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		setKeepAlive(conn, KeepAliveTime)

		go s.Handler(conn, raddr)
	}
}

func (s *TcpForwardServer) handleTcpForward(conn net.Conn, raddr net.Addr) {
	defer conn.Close()

	log.Printf("[tcp] %s - %s\n", conn.RemoteAddr(), raddr)
	cc, err := s.Base.Chain.Dial(raddr.String())
	if err != nil {
		log.Printf("[tcp] %s -> %s : %s\n", conn.RemoteAddr(), raddr, err)
		return
	}
	defer cc.Close()

	log.Printf("[tcp] %s <-> %s\n", conn.RemoteAddr(), raddr)
	s.Base.transport(conn, cc)
	log.Printf("[tcp] %s >-< %s\n", conn.RemoteAddr(), raddr)
}

type packet struct {
	srcAddr string // src address
	dstAddr string // dest address
	data    []byte
}

type cnode struct {
	chain            *ProxyChain
	conn             net.Conn
	srcAddr, dstAddr string
	rChan, wChan     chan *packet
	err              error
	ttl              time.Duration
}

func (node *cnode) getUDPTunnel() (net.Conn, error) {
	conn, err := node.chain.GetConn()
	if err != nil {
		return nil, err
	}

	conn.SetWriteDeadline(time.Now().Add(WriteTimeout))
	if err = gosocks5.NewRequest(CmdUdpTun, nil).Write(conn); err != nil {
		conn.Close()
		return nil, err
	}
	conn.SetWriteDeadline(time.Time{})

	conn.SetReadDeadline(time.Now().Add(ReadTimeout))
	reply, err := gosocks5.ReadReply(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	conn.SetReadDeadline(time.Time{})

	if reply.Rep != gosocks5.Succeeded {
		conn.Close()
		return nil, errors.New("UDP tunnel failure")
	}

	return conn, nil
}

func (node *cnode) run() {
	if len(node.chain.Nodes()) == 0 {
		lconn, err := net.ListenUDP("udp", nil)
		if err != nil {
			log.Printf("[udp] %s -> %s : %s\n", node.srcAddr, node.dstAddr, err)
			node.err = err
			return
		}
		node.conn = lconn
	} else {
		tc, err := node.getUDPTunnel()
		if err != nil {
			log.Printf("[udp-tun] %s -> %s : %s\n", node.srcAddr, node.dstAddr, err)
			node.err = err
			return
		}
		node.conn = tc
	}

	defer node.conn.Close()

	timer := time.NewTimer(node.ttl)
	errChan := make(chan error, 2)

	go func() {
		for {
			switch c := node.conn.(type) {
			case *net.UDPConn:
				b := make([]byte, MediumBufferSize)
				n, addr, err := c.ReadFromUDP(b)
				if err != nil {
					log.Printf("[udp] %s <- %s : %s\n", node.srcAddr, node.dstAddr, err)
					node.err = err
					errChan <- err
					return
				}

				timer.Reset(node.ttl)
				log.Printf("[udp] %s <<< %s : length %d\n", node.srcAddr, addr, n)

				select {
				// swap srcAddr with dstAddr
				case node.rChan <- &packet{srcAddr: addr.String(), dstAddr: node.srcAddr, data: b[:n]}:
				case <-time.After(time.Second * 3):
					log.Printf("[udp] %s <- %s : %s\n", node.srcAddr, node.dstAddr, "recv queue is full, discard")
				}

			default:
				dgram, err := gosocks5.ReadUDPDatagram(c)
				if err != nil {
					log.Printf("[udp-tun] %s <- %s : %s\n", node.srcAddr, node.dstAddr, err)
					node.err = err
					errChan <- err
					return
				}

				timer.Reset(node.ttl)
				log.Printf("[udp-tun] %s <<< %s : length %d\n", node.srcAddr, dgram.Header.Addr.String(), len(dgram.Data))

				select {
				// swap srcAddr with dstAddr
				case node.rChan <- &packet{srcAddr: dgram.Header.Addr.String(), dstAddr: node.srcAddr, data: dgram.Data}:
				case <-time.After(time.Second * 3):
					log.Printf("[udp-tun] %s <- %s : %s\n", node.srcAddr, node.dstAddr, "recv queue is full, discard")
				}
			}
		}
	}()

	go func() {
		for pkt := range node.wChan {
			timer.Reset(node.ttl)

			dstAddr, err := net.ResolveUDPAddr("udp", pkt.dstAddr)
			if err != nil {
				log.Printf("[udp] %s -> %s : %s\n", pkt.srcAddr, pkt.dstAddr, err)
				continue
			}

			switch c := node.conn.(type) {
			case *net.UDPConn:
				if _, err := c.WriteToUDP(pkt.data, dstAddr); err != nil {
					log.Printf("[udp] %s -> %s : %s\n", pkt.srcAddr, pkt.dstAddr, err)
					node.err = err
					errChan <- err
					return
				}
				log.Printf("[udp] %s >>> %s : length %d\n", pkt.srcAddr, pkt.dstAddr, len(pkt.data))

			default:
				dgram := gosocks5.NewUDPDatagram(gosocks5.NewUDPHeader(uint16(len(pkt.data)), 0, ToSocksAddr(dstAddr)), pkt.data)
				if err := dgram.Write(c); err != nil {
					log.Printf("[udp-tun] %s -> %s : %s\n", pkt.srcAddr, pkt.dstAddr, err)
					node.err = err
					errChan <- err
					return
				}
				log.Printf("[udp-tun] %s >>> %s : length %d\n", pkt.srcAddr, pkt.dstAddr, len(pkt.data))
			}
		}
	}()

	select {
	case <-errChan:
	case <-timer.C:
	}
}

type UdpForwardServer struct {
	Base *ProxyServer
	TTL  int
}

func NewUdpForwardServer(base *ProxyServer, ttl int) *UdpForwardServer {
	return &UdpForwardServer{Base: base, TTL: ttl}
}

func (s *UdpForwardServer) ListenAndServe() error {
	laddr, err := net.ResolveUDPAddr("udp", s.Base.Node.Addr)
	if err != nil {
		return err
	}

	raddr, err := net.ResolveUDPAddr("udp", s.Base.Node.Remote)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		log.Printf("[udp] %s -> %s : %s\n", laddr, raddr, err)
		return err
	}
	defer conn.Close()

	rChan, wChan := make(chan *packet, 128), make(chan *packet, 128)
	// start send queue
	go func(ch chan<- *packet) {
		for {
			b := make([]byte, MediumBufferSize)
			n, addr, err := conn.ReadFromUDP(b)
			if err != nil {
				log.Printf("[udp] %s -> %s : %s\n", laddr, raddr, err)
				continue
			}

			select {
			case ch <- &packet{srcAddr: addr.String(), dstAddr: raddr.String(), data: b[:n]}:
			case <-time.After(time.Second * 3):
				log.Printf("[udp] %s -> %s : %s\n", addr, raddr, "send queue is full, discard")
			}
		}
	}(wChan)
	// start recv queue
	go func(ch <-chan *packet) {
		for pkt := range ch {
			dstAddr, err := net.ResolveUDPAddr("udp", pkt.dstAddr)
			if err != nil {
				log.Printf("[udp] %s <- %s : %s\n", pkt.dstAddr, pkt.srcAddr, err)
				continue
			}
			if _, err := conn.WriteToUDP(pkt.data, dstAddr); err != nil {
				log.Printf("[udp] %s <- %s : %s\n", pkt.dstAddr, pkt.srcAddr, err)
				return
			}
		}
	}(rChan)

	// mapping client to node
	m := make(map[string]*cnode)

	// start dispatcher
	for pkt := range wChan {
		// clear obsolete nodes
		for k, node := range m {
			if node != nil && node.err != nil {
				close(node.wChan)
				delete(m, k)
				log.Printf("[udp] clear node %s\n", k)
			}
		}

		node, ok := m[pkt.srcAddr]
		if !ok {
			node = &cnode{
				chain:   s.Base.Chain,
				srcAddr: pkt.srcAddr,
				dstAddr: pkt.dstAddr,
				rChan:   rChan,
				wChan:   make(chan *packet, 32),
				ttl:     time.Duration(s.TTL) * time.Second,
			}
			m[pkt.srcAddr] = node
			go node.run()
			log.Printf("[udp] %s -> %s : new client (%d)\n", pkt.srcAddr, pkt.dstAddr, len(m))
		}

		select {
		case node.wChan <- pkt:
		case <-time.After(time.Second * 3):
			log.Printf("[udp] %s -> %s : %s\n", pkt.srcAddr, pkt.dstAddr, "node send queue is full, discard")
		}
	}

	return nil
}

type RTcpForwardServer struct {
	Base *ProxyServer
}

func NewRTcpForwardServer(base *ProxyServer) *RTcpForwardServer {
	return &RTcpForwardServer{Base: base}
}

func (s *RTcpForwardServer) Serve() error {
	if len(s.Base.Chain.nodes) == 0 {
		return errors.New("rtcp: at least one -F must be assigned")
	}

	laddr, err := net.ResolveTCPAddr("tcp", s.Base.Node.Addr)
	if err != nil {
		return err
	}
	raddr, err := net.ResolveTCPAddr("tcp", s.Base.Node.Remote)
	if err != nil {
		return err
	}

	retry := 0
	for {
		conn, err := s.Base.Chain.GetConn()
		if err != nil {
			log.Printf("[rtcp] %s - %s : %s\n", laddr, raddr, err)
			time.Sleep((1 << uint(retry)) * time.Second)
			if retry < 5 {
				retry++
			}
			continue
		}
		retry = 0

		if err := s.connectRTcpForward(conn, laddr, raddr); err != nil {
			conn.Close()
			time.Sleep(6 * time.Second)
		}
	}
}

func (s *RTcpForwardServer) connectRTcpForward(conn net.Conn, laddr, raddr net.Addr) error {
	log.Printf("[rtcp] %s - %s\n", laddr, raddr)

	req := gosocks5.NewRequest(gosocks5.CmdBind, ToSocksAddr(laddr))
	if err := req.Write(conn); err != nil {
		log.Printf("[rtcp] %s -> %s : %s\n", laddr, raddr, err)
		return err
	}

	// first reply, bind status
	conn.SetReadDeadline(time.Now().Add(ReadTimeout))
	rep, err := gosocks5.ReadReply(conn)
	if err != nil {
		log.Printf("[rtcp] %s -> %s : %s\n", laddr, raddr, err)
		return err
	}
	conn.SetReadDeadline(time.Time{})
	if rep.Rep != gosocks5.Succeeded {
		log.Printf("[rtcp] %s -> %s : bind on %s failure\n", laddr, raddr, laddr)
		return errors.New("Bind on " + laddr.String() + " failure")
	}
	log.Printf("[rtcp] %s - %s BIND ON %s OK\n", laddr, raddr, rep.Addr)

	// second reply, peer connection
	rep, err = gosocks5.ReadReply(conn)
	if err != nil {
		log.Printf("[rtcp] %s -> %s : %s\n", laddr, raddr, err)
		return err
	}
	if rep.Rep != gosocks5.Succeeded {
		log.Printf("[rtcp] %s -> %s : peer connect failure\n", laddr, raddr)
		return errors.New("peer connect failure")
	}

	log.Printf("[rtcp] %s -> %s PEER %s CONNECTED\n", laddr, raddr, rep.Addr)

	go func() {
		defer conn.Close()

		lconn, err := net.DialTimeout("tcp", raddr.String(), time.Second*180)
		if err != nil {
			log.Printf("[rtcp] %s -> %s : %s\n", rep.Addr, raddr, err)
			return
		}
		defer lconn.Close()

		log.Printf("[rtcp] %s <-> %s\n", rep.Addr, lconn.RemoteAddr())
		s.Base.transport(lconn, conn)
		log.Printf("[rtcp] %s >-< %s\n", rep.Addr, lconn.RemoteAddr())
	}()

	return nil
}

type RUdpForwardServer struct {
	Base *ProxyServer
}

func NewRUdpForwardServer(base *ProxyServer) *RUdpForwardServer {
	return &RUdpForwardServer{Base: base}
}

func (s *RUdpForwardServer) Serve() error {
	if len(s.Base.Chain.nodes) == 0 {
		return errors.New("rudp: at least one -F must be assigned")
	}

	laddr, err := net.ResolveUDPAddr("udp", s.Base.Node.Addr)
	if err != nil {
		return err
	}
	raddr, err := net.ResolveUDPAddr("udp", s.Base.Node.Remote)
	if err != nil {
		return err
	}

	retry := 0
	for {
		conn, err := s.Base.Chain.GetConn()
		if err != nil {
			log.Printf("[rudp] %s - %s : %s\n", laddr, raddr, err)
			time.Sleep((1 << uint(retry)) * time.Second)
			if retry < 5 {
				retry++
			}
			continue
		}
		retry = 0

		if err := s.connectRUdpForward(conn, laddr, raddr); err != nil {
			conn.Close()
			time.Sleep(6 * time.Second)
		}
	}
}

func (s *RUdpForwardServer) connectRUdpForward(conn net.Conn, laddr, raddr *net.UDPAddr) error {
	log.Printf("[rudp] %s - %s\n", laddr, raddr)

	req := gosocks5.NewRequest(CmdUdpTun, ToSocksAddr(laddr))
	conn.SetWriteDeadline(time.Now().Add(WriteTimeout))
	if err := req.Write(conn); err != nil {
		log.Printf("[rudp] %s -> %s : %s\n", laddr, raddr, err)
		return err
	}
	conn.SetWriteDeadline(time.Time{})

	conn.SetReadDeadline(time.Now().Add(ReadTimeout))
	rep, err := gosocks5.ReadReply(conn)
	if err != nil {
		log.Printf("[rudp] %s <- %s : %s\n", laddr, raddr, err)
		return err
	}
	conn.SetReadDeadline(time.Time{})

	if rep.Rep != gosocks5.Succeeded {
		log.Printf("[rudp] %s <- %s : bind on %s failure\n", laddr, raddr, laddr)
		return errors.New(fmt.Sprintf("bind on %s failure", laddr))
	}

	log.Printf("[rudp] %s - %s BIND ON %s OK\n", laddr, raddr, rep.Addr)

	for {
		dgram, err := gosocks5.ReadUDPDatagram(conn)
		if err != nil {
			log.Printf("[rudp] %s <- %s : %s\n", laddr, raddr, err)
			return err
		}

		go func() {
			b := make([]byte, MediumBufferSize)

			relay, err := net.DialUDP("udp", nil, raddr)
			if err != nil {
				log.Printf("[rudp] %s -> %s : %s\n", laddr, raddr, err)
				return
			}
			defer relay.Close()

			if _, err := relay.Write(dgram.Data); err != nil {
				log.Printf("[rudp] %s -> %s : %s\n", laddr, raddr, err)
				return
			}
			log.Printf("[rudp] %s >>> %s length: %d\n", laddr, raddr, len(dgram.Data))

			relay.SetReadDeadline(time.Now().Add(ReadTimeout))
			n, err := relay.Read(b)
			if err != nil {
				log.Printf("[rudp] %s <- %s : %s\n", laddr, raddr, err)
				return
			}
			relay.SetReadDeadline(time.Time{})

			log.Printf("[rudp] %s <<< %s length: %d\n", laddr, raddr, n)

			conn.SetWriteDeadline(time.Now().Add(WriteTimeout))
			if err := gosocks5.NewUDPDatagram(gosocks5.NewUDPHeader(uint16(n), 0, dgram.Header.Addr), b[:n]).Write(conn); err != nil {
				log.Printf("[rudp] %s <- %s : %s\n", laddr, raddr, err)
				return
			}
			conn.SetWriteDeadline(time.Time{})
		}()
	}
}
