package dns

import (
	"log"
	"net"
	"time"

	"github.com/miekg/dns"
)

const (
	notIPQuery = 0
	_IP4Query  = 4
	_IP6Query  = 6
)

// Question type
type Question struct {
	Qname  string `json:"name"`
	Qtype  string `json:"type"`
	Qclass string `json:"class"`
	Qnet   string `json:"net"`
}

// String formats a question
func (q *Question) String() string {
	return q.Qname + " " + q.Qclass + " " + q.Qtype
}

// QuestionCacheEntry represents a full query from a client with metadata
type QuestionCacheEntry struct {
	Date    int64    `json:"date"`
	Remote  string   `json:"client"`
	Blocked bool     `json:"blocked"`
	Query   Question `json:"query"`
}

// DNSHandler type
type DNSHandler struct {
	resolver *Resolver
	cache    Cache // cache success
	negCache Cache // cache failure
}

// NewHandler returns a new DNSHandler
func NewHandler() *DNSHandler {
	resolver := &Resolver{}

	cache := &MemoryCache{
		Backend:  make(map[string]Mesg, gConfig.Maxcount),
		Expire:   time.Duration(gConfig.Expire) * time.Second,
		Maxcount: gConfig.Maxcount,
	}
	negCache := &MemoryCache{
		Backend:  make(map[string]Mesg),
		Expire:   time.Duration(gConfig.Expire) * time.Second / 2,
		Maxcount: gConfig.Maxcount,
	}

	return &DNSHandler{resolver, cache, negCache}
}

func (h *DNSHandler) do(Net string, w dns.ResponseWriter, req *dns.Msg) {
	defer w.Close()
	q := req.Question[0]
	Q := Question{
		UnFqdn(q.Name),
		dns.TypeToString[q.Qtype],
		dns.ClassToString[q.Qclass],
		Net,
	}

	var remote net.IP
	if Net == "tcp" {
		remote = w.RemoteAddr().(*net.TCPAddr).IP
	} else {
		remote = w.RemoteAddr().(*net.UDPAddr).IP
	}
	log.Printf("%s lookup %s\n", remote, Q)

	// Only lookup cache when qclass == 'IN', qtype == 'A'|'AAAA'
	// tcp and udp use same cache key
	key := Q.String()
	IPQuery := h.isIPQuery(q)
	if IPQuery != notIPQuery {
		mesg, blocked, err := h.cache.Get(key)
		if err != nil {
			if _, _, err = h.negCache.Get(key); err != nil {
				log.Printf("%s didn't hit cache\n", Q)
			} else {
				log.Printf("%s hit negative cache\n", Q)
				dns.HandleFailed(w, req)
				return
			}
		} else {
			if blocked {
				log.Printf("%s hit blocked cache\n", Q)
			} else {
				log.Printf("%s hit cache\n", Q)
			}

			// we need this copy against concurrent modification of Id
			msg := *mesg
			msg.Id = req.Id
			h.WriteReplyMsg(w, &msg)
			return
		}

		if gBlockCache.Exists(Q.Qname) {
			log.Printf("%s found in blocklist\n", Q.Qname)

			m := new(dns.Msg)
			m.SetReply(req)
			switch IPQuery {
			case _IP4Query:
				rrHeader := dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    gConfig.TTL,
				}
				a := &dns.A{Hdr: rrHeader, A: net.ParseIP(gConfig.Nullroute)}
				m.Answer = append(m.Answer, a)
			case _IP6Query:
				rrHeader := dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeAAAA,
					Class:  dns.ClassINET,
					Ttl:    gConfig.TTL,
				}
				a := &dns.AAAA{Hdr: rrHeader, AAAA: net.ParseIP(gConfig.Nullroutev6)}
				m.Answer = append(m.Answer, a)
			}
			h.WriteReplyMsg(w, m)

			// cache the block
			err := h.cache.Set(key, m, true)
			if err != nil {
				log.Printf("failed to set %s block cache: %s\n", Q, err)
			}

			// log query
			NewEntry := QuestionCacheEntry{Date: time.Now().Unix(), Remote: remote.String(), Query: Q, Blocked: true}
			gQuestionCache.Add(NewEntry)

			return
		}
	}

	// log query
	NewEntry := QuestionCacheEntry{Date: time.Now().Unix(), Remote: remote.String(), Query: Q, Blocked: false}
	go gQuestionCache.Add(NewEntry)

	mesg, err := h.resolver.Lookup(Net, req)
	if err != nil {
		log.Printf("failed to resolve query %s: %s\n", Q, err)
		dns.HandleFailed(w, req)

		// cache the failure, too!
		if err = h.negCache.Set(key, nil, false); err != nil {
			log.Printf("failed to set %s negative cache: %s\n", Q, err)
		}
		return
	}
	if mesg.Truncated && Net == "udp" {
		mesg, err = h.resolver.Lookup("tcp", req)
		if err != nil {
			log.Printf("failed to resolve backup tcp query %s: %s\n", Q, err)
			dns.HandleFailed(w, req)

			// cache the failure, too!
			if err = h.negCache.Set(key, nil, false); err != nil {
				log.Printf("failed to set %s negative cache: %s\n", Q, err)
			}
			return
		}
	}

	h.WriteReplyMsg(w, mesg)

	if IPQuery != notIPQuery && len(mesg.Answer) > 0 {
		err = h.cache.Set(key, mesg, false)
		if err != nil {
			log.Printf("failed to set %s cache: %s\n", Q, err)
		}
		log.Printf("insert %s into cache\n", Q)
	}
}

// DoTCP begins a tcp query
func (h *DNSHandler) DoTCP(w dns.ResponseWriter, req *dns.Msg) {
	go h.do("tcp", w, req)
}

// DoUDP begins a udp query
func (h *DNSHandler) DoUDP(w dns.ResponseWriter, req *dns.Msg) {
	go h.do("udp", w, req)
}

func (h *DNSHandler) WriteReplyMsg(w dns.ResponseWriter, message *dns.Msg) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("recovered in WriteReplyMsg: %s\n", r)
		}
	}()

	err := w.WriteMsg(message)
	if err != nil {
		log.Println(err)
	}
}

func (h *DNSHandler) isIPQuery(q dns.Question) int {
	if q.Qclass != dns.ClassINET {
		return notIPQuery
	}

	switch q.Qtype {
	case dns.TypeA:
		return _IP4Query
	case dns.TypeAAAA:
		return _IP6Query
	default:
		return notIPQuery
	}
}

// UnFqdn function
func UnFqdn(s string) string {
	if dns.IsFqdn(s) {
		return s[:len(s)-1]
	}
	return s
}
