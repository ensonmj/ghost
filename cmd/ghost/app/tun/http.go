package tun

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"

	"github.com/elazarl/goproxy"
	"github.com/pkg/errors"
)

type HttpNode struct {
	pn *ProxyNode
	pc *ProxyChain
}

func NewHttpNode(pn *ProxyNode) *HttpNode {
	return &HttpNode{
		pn: pn,
	}
}

func (n *HttpNode) ListenAndServe(pc *ProxyChain) error {
	n.pc = pc
	return http.ListenAndServe(n.pn.URL.Host, n.GetHttpProxyHandler(true))
}

func (n *HttpNode) GetHttpProxyHandler(verbose bool) http.Handler {
	return &goproxy.ProxyHttpServer{
		Tr: &http.Transport{
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			MaxIdleConnsPerHost: 1000,
			DisableKeepAlives:   true,
			Dial:                n.Dial,
		},
		NonproxyHandler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			http.Error(w, "This is a proxy server. Does not respond to non-proxy requests.", 500)
		}),
		Verbose: verbose,
		Logger:  log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds),
	}
}

// Dial server or chain proxy
func (n *HttpNode) Dial(network, addr string) (net.Conn, error) {
	return n.pc.Dial(network, addr)
}

func (n *HttpNode) String() string {
	return fmt.Sprintf("%s", n.pn)
}

func (n *HttpNode) Connect() (net.Conn, error) {
	log.Printf("connect to chain first node: %s\n", n)
	c, err := net.Dial("tcp", n.pn.URL.Host)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	return c, nil
}

func (n *HttpNode) Handshake(c net.Conn) error {
	return nil
}

func (n *HttpNode) ForwardRequest(c net.Conn, addr string) error {
	// use CONNECT to create tunnel
	log.Printf("handshake with chain node: %s\n", n)
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

func (n *HttpNode) encodeBasicAuth() string {
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

// type Http2Server struct {
// 	Base      *ProxyServer
// 	TLSConfig *tls.Config
// 	Handler   http.Handler
// }

// func NewHttp2Server(base *ProxyServer) *Http2Server {
// 	return &Http2Server{Base: base}
// }

// func (s *Http2Server) ListenAndServeTLS(config *tls.Config) error {
// 	srv := http.Server{
// 		Addr:      s.Base.Node.Addr,
// 		Handler:   s.Handler,
// 		TLSConfig: config,
// 	}
// 	if srv.Handler == nil {
// 		srv.Handler = http.HandlerFunc(s.HandleRequest)
// 	}
// 	http2.ConfigureServer(&srv, nil)
// 	return srv.ListenAndServeTLS("", "")
// }

// // Default HTTP2 server handler
// func (s *Http2Server) HandleRequest(w http.ResponseWriter, req *http.Request) {
// 	target := req.Header.Get("Ghost-Target")
// 	if target == "" {
// 		target = req.Host
// 	}
// 	log.Printf("[http2] %s %s - %s %s\n", req.Method, req.RemoteAddr, target, req.Proto)

// 	w.Header().Set("Proxy-Agent", "ghost/"+Version)

// 	// HTTP2 as transport
// 	if req.Header.Get("Proxy-Switch") == "ghost" {
// 		conn, err := s.Upgrade(w, req)
// 		if err != nil {
// 			log.Printf("[http2] %s -> %s : %s\n", req.RemoteAddr, target, err)
// 			return
// 		}
// 		log.Printf("[http2] %s - %s : switch to HTTP2 transport mode OK\n",
// 			req.RemoteAddr, target)
// 		s.Base.handleConn(conn)
// 		return
// 	}

// 	valid := false
// 	u, p, _ := basicProxyAuth(req.Header.Get("Proxy-Authorization"))
// 	// for _, user := range s.Base.Node.Users {
// 	user := s.Base.Node.User
// 	if user != nil {
// 		username := user.Username()
// 		password, _ := user.Password()
// 		if (u == username && p == password) ||
// 			(u == username && password == "") ||
// 			(username == "" && p == password) {
// 			valid = true
// 		}

// 		if !valid {
// 			log.Printf("[http2] %s <- %s : proxy authentication required\n",
// 				req.RemoteAddr, target)
// 			w.WriteHeader(http.StatusProxyAuthRequired)
// 			return
// 		}
// 	}

// 	req.Header.Del("Proxy-Authorization")

// 	c, err := s.Base.Chain.Dial(target)
// 	if err != nil {
// 		log.Printf("[http2] %s -> %s : %s\n", req.RemoteAddr, target, err)
// 		w.WriteHeader(http.StatusServiceUnavailable)
// 		return
// 	}
// 	defer c.Close()

// 	log.Printf("[http2] %s <-> %s\n", req.RemoteAddr, target)

// 	if req.Method == http.MethodConnect {
// 		w.WriteHeader(http.StatusOK)
// 		if fw, ok := w.(http.Flusher); ok {
// 			fw.Flush()
// 		}

// 		// compatible with HTTP1.x
// 		if hj, ok := w.(http.Hijacker); ok && req.ProtoMajor == 1 {
// 			// we take over the underly connection
// 			conn, _, err := hj.Hijack()
// 			if err != nil {
// 				log.Printf("[http2] %s -> %s : %s\n", req.RemoteAddr, target, err)
// 				w.WriteHeader(http.StatusInternalServerError)
// 				return
// 			}
// 			defer conn.Close()

// 			s.Base.transport(conn, c)
// 			return
// 		}

// 		errc := make(chan error, 2)

// 		go func() {
// 			_, err := io.Copy(c, req.Body)
// 			errc <- err
// 		}()
// 		go func() {
// 			_, err := io.Copy(flushWriter{w}, c)
// 			errc <- err
// 		}()

// 		select {
// 		case <-errc:
// 			// glog.V(LWARNING).Infoln("exit", err)
// 		}
// 		log.Printf("[http2] %s >-< %s\n", req.RemoteAddr, target)
// 		return
// 	}

// 	req.Header.Set("Connection", "Keep-Alive")
// 	if err = req.Write(c); err != nil {
// 		log.Printf("[http2] %s -> %s : %s\n", req.RemoteAddr, target, err)
// 		return
// 	}

// 	resp, err := http.ReadResponse(bufio.NewReader(c), req)
// 	if err != nil {
// 		log.Println(err)
// 		return
// 	}
// 	defer resp.Body.Close()

// 	for k, v := range resp.Header {
// 		for _, vv := range v {
// 			w.Header().Add(k, vv)
// 		}
// 	}
// 	w.WriteHeader(resp.StatusCode)
// 	if _, err := io.Copy(flushWriter{w}, resp.Body); err != nil {
// 		log.Printf("[http2] %s <- %s : %s\n", req.RemoteAddr, target, err)
// 	}

// 	log.Printf("[http2] %s >-< %s\n", req.RemoteAddr, target)
// }

// // Upgrade upgrade an HTTP2 request to a bidirectional connection that preparing for tunneling other protocol, just like a websocket connection.
// func (s *Http2Server) Upgrade(w http.ResponseWriter, r *http.Request) (net.Conn, error) {
// 	if r.Method != http.MethodConnect {
// 		w.WriteHeader(http.StatusMethodNotAllowed)
// 		return nil, errors.New("Method not allowed")
// 	}

// 	w.WriteHeader(http.StatusOK)

// 	if fw, ok := w.(http.Flusher); ok {
// 		fw.Flush()
// 	}

// 	conn := &http2Conn{r: r.Body, w: flushWriter{w}}
// 	conn.remoteAddr, _ = net.ResolveTCPAddr("tcp", r.RemoteAddr)
// 	conn.localAddr, _ = net.ResolveTCPAddr("tcp", r.Host)
// 	return conn, nil
// }

// // HTTP2 client connection, wrapped up just like a net.Conn
// type http2Conn struct {
// 	r          io.Reader
// 	w          io.Writer
// 	remoteAddr net.Addr
// 	localAddr  net.Addr
// }

// func (c *http2Conn) Read(b []byte) (n int, err error) {
// 	return c.r.Read(b)
// }

// func (c *http2Conn) Write(b []byte) (n int, err error) {
// 	return c.w.Write(b)
// }

// func (c *http2Conn) Close() (err error) {
// 	if rc, ok := c.r.(io.Closer); ok {
// 		err = rc.Close()
// 	}
// 	if w, ok := c.w.(io.Closer); ok {
// 		err = w.Close()
// 	}
// 	return
// }

// func (c *http2Conn) LocalAddr() net.Addr {
// 	return c.localAddr
// }

// func (c *http2Conn) RemoteAddr() net.Addr {
// 	return c.remoteAddr
// }

// func (c *http2Conn) SetDeadline(t time.Time) error {
// 	return &net.OpError{Op: "set", Net: "http2", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
// }

// func (c *http2Conn) SetReadDeadline(t time.Time) error {
// 	return &net.OpError{Op: "set", Net: "http2", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
// }

// func (c *http2Conn) SetWriteDeadline(t time.Time) error {
// 	return &net.OpError{Op: "set", Net: "http2", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
// }

// type flushWriter struct {
// 	w io.Writer
// }

// func (fw flushWriter) Write(p []byte) (n int, err error) {
// 	defer func() {
// 		if r := recover(); r != nil {
// 			if s, ok := r.(string); ok {
// 				err = errors.New(s)
// 				return
// 			}
// 			err = r.(error)
// 		}
// 	}()

// 	n, err = fw.w.Write(p)
// 	if err != nil {
// 		// glog.V(LWARNING).Infoln("flush writer:", err)
// 		return
// 	}
// 	if f, ok := fw.w.(http.Flusher); ok {
// 		f.Flush()
// 	}
// 	return
// }

// func basicProxyAuth(proxyAuth string) (username, password string, ok bool) {
// 	if proxyAuth == "" {
// 		return
// 	}

// 	if !strings.HasPrefix(proxyAuth, "Basic ") {
// 		return
// 	}
// 	c, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(proxyAuth, "Basic "))
// 	if err != nil {
// 		return
// 	}
// 	cs := string(c)
// 	s := strings.IndexByte(cs, ':')
// 	if s < 0 {
// 		return
// 	}

// 	return cs[:s], cs[s+1:], true
// }
