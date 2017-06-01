package tun

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"

	"github.com/lucas-clemente/quic-go/h2quic"
	"github.com/pkg/errors"
)

type QuicServer struct {
	pn        *ProxyNode
	pc        *ProxyChain
	Handler   http.Handler
	TLSConfig *tls.Config
}

func NewQuicServer(pn *ProxyNode, config *tls.Config) *QuicServer {
	return &QuicServer{
		pn:        pn,
		TLSConfig: config,
	}
}

func (s *QuicServer) ListenAndServe(pc *ProxyChain) error {
	s.pc = pc
	server := &h2quic.Server{
		Server: &http.Server{
			Addr:      s.pn.URL.Host,
			Handler:   http.HandlerFunc(s.HandleRequest),
			TLSConfig: s.TLSConfig,
		},
	}
	return server.ListenAndServe()
}

// Dial server or chain proxy
func (n *QuicServer) Dial(network, addr string) (net.Conn, error) {
	return n.pc.Dial(network, addr)
}

func (n *QuicServer) String() string {
	return fmt.Sprintf("%s", n.pn)
}

func (n *QuicServer) Connect() (net.Conn, error) {
	log.Printf("connect to chain first node: %s\n", n)
	c, err := net.Dial("tcp", n.pn.URL.Host)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	return c, nil
}

func (n *QuicServer) Handshake(c net.Conn) error {
	return nil
}

func (n *QuicServer) ForwardRequest(c net.Conn, addr string) error {
	// use CONNECT to create tunnel
	log.Printf("handshake with chain node: %s in conn:%s -> %s\n", n,
		c.LocalAddr(), c.RemoteAddr())
	req := &http.Request{
		Method:     http.MethodConnect,
		URL:        &url.URL{Host: addr},
		Host:       addr,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
	}
	req.Header.Set("Proxy-Connection", "keep-alive")
	if authStr := n.encodeBasicAuth(); authStr != "" {
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
		c.Close()
		return errors.New("proxy refused connection" + string(resp))
	}

	return nil
}

func (n *QuicServer) encodeBasicAuth() string {
	var authStr string
	user := n.pn.URL.User
	if user != nil {
		s := user.String()
		if _, set := user.Password(); !set {
			s += ":"
		}
		authStr = "Basic " + base64.StdEncoding.EncodeToString([]byte(s))
	}
	return authStr
}

func (s *QuicServer) HandleRequest(w http.ResponseWriter, req *http.Request) {
	target := req.Host
	log.Printf("[quic] %s %s - %s %s", req.Method, req.RemoteAddr, target, req.Proto)

	c, err := s.Dial("tcp", target)
	if err != nil {
		log.Printf("[quic] %s -> %s : %s", req.RemoteAddr, target, err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	defer c.Close()

	log.Printf("[quic] %s <-> %s", req.RemoteAddr, target)

	req.Header.Set("Connection", "Keep-Alive")
	if err = req.Write(c); err != nil {
		log.Printf("[quic] %s -> %s : %s", req.RemoteAddr, target, err)
		return
	}

	resp, err := http.ReadResponse(bufio.NewReader(c), req)
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(flushWriter{w}, resp.Body); err != nil {
		log.Printf("[quic] %s <- %s : %s", req.RemoteAddr, target, err)
	}

	log.Printf("[quic] %s >-< %s", req.RemoteAddr, target)
}

type flushWriter struct {
	w io.Writer
}

func (fw flushWriter) Write(p []byte) (n int, err error) {
	defer func() {
		if r := recover(); r != nil {
			if s, ok := r.(string); ok {
				err = errors.New(s)
				return
			}
			err = r.(error)
		}
	}()

	n, err = fw.w.Write(p)
	if err != nil {
		return
	}
	if f, ok := fw.w.(http.Flusher); ok {
		f.Flush()
	}
	return
}
