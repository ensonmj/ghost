package dns

import (
	"log"
	"time"

	"github.com/miekg/dns"
)

// Server type
type Server struct {
	host     string
	rTimeout time.Duration
	wTimeout time.Duration
}

func NewServer(rTimeout, wTimeout time.Duration) *Server {
	return &Server{
		host:     gConfig.Bind,
		rTimeout: rTimeout,
		wTimeout: wTimeout,
	}
}

// Run starts the server
func (s *Server) AsyRun() {
	handler := NewHandler()

	tcpHandler := dns.NewServeMux()
	tcpHandler.HandleFunc(".", handler.DoTCP)

	udpHandler := dns.NewServeMux()
	udpHandler.HandleFunc(".", handler.DoUDP)

	tcpServer := &dns.Server{Addr: s.host,
		Net:          "tcp",
		Handler:      tcpHandler,
		ReadTimeout:  s.rTimeout,
		WriteTimeout: s.wTimeout}

	udpServer := &dns.Server{Addr: s.host,
		Net:          "udp",
		Handler:      udpHandler,
		UDPSize:      65535,
		ReadTimeout:  s.rTimeout,
		WriteTimeout: s.wTimeout}

	go s.start(udpServer)
	go s.start(tcpServer)
}

func (s *Server) start(ds *dns.Server) {
	log.Printf("start %s listener on %s\n", ds.Net, s.host)

	if err := ds.ListenAndServe(); err != nil {
		log.Printf("start %s listener on %s failed: %s\n", ds.Net, s.host, err.Error())
	}
}
