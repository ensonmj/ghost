package dns

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/miekg/dns"
)

func UpdateFakeIP(nameserver string) {
	c := &dns.Client{
		ReadTimeout:  gConfig.Timeout.Duration,
		WriteTimeout: gConfig.Timeout.Duration,
	}
	m := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			RecursionDesired: true,
		},
		Question: make([]dns.Question, 1),
	}
	m.Question[0] = dns.Question{
		Qtype:  dns.TypeA,
		Qclass: uint16(dns.ClassINET),
	}

	for _, ip := range gConfig.FakeIps {
		if !gFakeIPCache.Exists(ip) {
			gFakeIPCache.Set(ip, true)
			log.Printf("add cold fake ip:%s\n", ip)
		}
	}

	tInterval := time.NewTicker(gConfig.FakeInterval.Duration)
	defer tInterval.Stop()
	for {
		getFakeIP(c, m, nameserver)
		<-tInterval.C
	}
}

func getFakeIP(c *dns.Client, m *dns.Msg, nameserver string) {
	qname := fmt.Sprintf("r%d-1.googlevideo.com", rand.Int31())
	m.Question[0].Name = dns.Fqdn(qname)
	m.Id = dns.Id()

	r, _, err := c.Exchange(m, nameserver)
	if err != nil {
		log.Printf("failed to lookup fake ip for %s, err:%s\n", qname, err)
		return
	}
	if r.Id != m.Id {
		log.Println("Id mismatch")
		return
	}

	for _, answer := range r.Answer {
		switch t := answer.(type) {
		case *dns.A:
			ip := t.A.String()
			if !gFakeIPCache.Exists(ip) {
				gFakeIPCache.Set(ip, true)
				log.Printf("add fake ip:%s\n", ip)
			}
		}
	}
}

func checkFakeIP(m *dns.Msg) bool {
	for _, answer := range m.Answer {
		switch t := answer.(type) {
		case *dns.A:
			ip := t.A.String()
			if gFakeIPCache.Exists(ip) {
				log.Printf("%s hit fake ip cache:%s\n", ip, m)
				return true
			}
		}
	}

	return false
}
