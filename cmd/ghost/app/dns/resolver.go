package dns

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// ResolvError type
type ResolvError struct {
	qname, net  string
	nameservers []string
}

// Error formats a ResolvError
func (e ResolvError) Error() string {
	errmsg := fmt.Sprintf("%s resolv failed on %s (%s)", e.qname, strings.Join(e.nameservers, "; "), e.net)
	return errmsg
}

// Resolver type
type Resolver struct {
}

// Lookup will ask each nameserver in top-to-bottom fashion, starting a new request
// in every second, and return as early as possbile (have an answer).
// It returns an error if no request has succeeded.
func (r *Resolver) Lookup(net string, req *dns.Msg) (message *dns.Msg, err error) {
	c := &dns.Client{
		Net:          net,
		ReadTimeout:  r.Timeout(),
		WriteTimeout: r.Timeout(),
	}

	res := make(chan *dns.Msg, 1)
	var wg sync.WaitGroup
	L := func(nameserver, qname string) {
		defer wg.Done()
		log.Printf("lookuping %s on %s\n", qname, nameserver)

		r, _, err := c.Exchange(req, nameserver)
		if err != nil {
			log.Printf("failed to exchange with %s for %s: %s\n",
				nameserver, qname, err)
			return
		}
		if r != nil && r.Rcode != dns.RcodeSuccess {
			log.Printf("failed to get an valid answer for %s on %s, rcode:%s\n",
				qname, nameserver, dns.RcodeToString[r.Rcode])
			return
		} else {
			if checkFakeIP(r) {
				log.Printf("%s resolve on %s hit fakeip cache: %s\n",
					qname, nameserver, r)
				return
			}
		}

		select {
		case res <- r:
			log.Printf("%s resolv on %s (%s)\n", qname, nameserver, net)
		default:
		}
	}

	ticker := time.NewTicker(time.Duration(gConfig.Interval) * time.Millisecond)
	defer ticker.Stop()

	qname := req.Question[0].Name
	// Start lookup on each nameserver top-down, in every second
	for _, ns := range r.Nameservers() {
		wg.Add(1)
		go L(ns, qname)
		// but exit early, if we have an answer
		select {
		case r := <-res:
			return r, nil
		case <-ticker.C:
			continue
		}
	}

	// wait for all the namservers to finish
	wg.Wait()
	select {
	case r := <-res:
		return r, nil
	default:
		return nil, ResolvError{qname, net, r.Nameservers()}
	}
}

// Nameservers return the array of nameservers
func (r *Resolver) Nameservers() (ns []string) {
	return gConfig.Nameservers
}

// Timeout returns the resolver timeout
func (r *Resolver) Timeout() time.Duration {
	return time.Duration(gConfig.Timeout) * time.Second
}
