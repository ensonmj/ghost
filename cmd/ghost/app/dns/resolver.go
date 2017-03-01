package dns

import (
	"context"
	"fmt"
	"log"
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
	return fmt.Sprintf("failed to resolv %s on %v (%s)",
		e.qname, e.nameservers, e.net)
}

// Resolver type
type Resolver struct {
}

// Lookup will ask each nameserver in top-to-bottom fashion, starting a new request
// in every second, and return as early as possbile (have an answer).
// It returns an error if no request has succeeded.
func (r *Resolver) Lookup(net string, req *dns.Msg) (*dns.Msg, error) {
	c := &dns.Client{
		Net:          net,
		ReadTimeout:  r.Timeout(),
		WriteTimeout: r.Timeout(),
	}

	var gMsg, cMsg *dns.Msg
	var gRes, cRes chan *dns.Msg
	ctx, _ := context.WithTimeout(context.Background(), r.SessionTimeout())

	if len(r.Nameservers()) > 0 {
		gRes = make(chan *dns.Msg, 1)
		go lookupFromServer(ctx, c, r.Nameservers(), req, gRes)
	}
	if len(r.CHNameservers()) > 0 {
		cRes = make(chan *dns.Msg, 1)
		go lookupFromServer(ctx, c, r.CHNameservers(), req, cRes)
	}

	for {
		select {
		case gMsg = <-gRes:
			gRes = nil
		case cMsg = <-cRes:
			cRes = nil
		}

		if gRes == nil && cRes == nil {
			break
		}
	}

	if gMsg != nil || cMsg != nil {
		return selectMsg(gMsg, cMsg)
	}

	qname := UnFqdn(req.Question[0].Name)
	if len(r.ISPNameservers()) <= 0 {
		return nil, ResolvError{qname, net, r.AllNameservers()}
	}

	ispRes := make(chan *dns.Msg, 1)
	ctx, _ = context.WithTimeout(context.Background(), r.SessionTimeout())
	lookupFromServer(ctx, c, r.ISPNameservers(), req, ispRes)
	msg := <-ispRes
	if msg != nil {
		return msg, nil
	}

	return nil, ResolvError{qname, net, r.AllNameservers()}
}

func lookupFromServer(ctx context.Context, c *dns.Client,
	nameservers []string, req *dns.Msg, res chan *dns.Msg) {
	defer close(res)

	msgChan := make(chan *dns.Msg, 1)
	wg := &sync.WaitGroup{}

	// Start lookup on each nameserver top-down, in every Interval millisecond
	ticker := time.NewTicker(time.Duration(gConfig.Interval) * time.Millisecond)
	defer ticker.Stop()

	for _, ns := range nameservers {
		wg.Add(1)
		go doLookup(c, ns, req, msgChan, wg)
		// but exit early, if we have an answer
		select {
		case <-ctx.Done():
			res <- nil
			qname := UnFqdn(req.Question[0].Name)
			log.Printf("resolve %s on %v timeout\n", qname, nameservers)
			return
		case r := <-msgChan:
			res <- r
			return
		case <-ticker.C:
			continue
		}
	}

	// wait for all the namservers to finish
	wg.Wait()
	select {
	case r := <-msgChan:
		res <- r
		return
	default:
		res <- nil
		qname := UnFqdn(req.Question[0].Name)
		log.Printf("resolve %s on %v get no valid answer\n", qname, nameservers)
		return
	}
}

func doLookup(c *dns.Client, nameserver string, req *dns.Msg,
	res chan *dns.Msg, wg *sync.WaitGroup) {
	defer wg.Done()

	qname := UnFqdn(req.Question[0].Name)
	log.Printf("lookuping %s on %s\n", qname, nameserver)

	r, _, err := c.Exchange(req, nameserver)
	if err != nil {
		log.Printf("failed to exchange with %s for %s: %s\n",
			nameserver, qname, err)
		return
	}
	if r != nil && r.Rcode != dns.RcodeSuccess {
		log.Printf("get an invalid answer for %s on %s, rcode:%s\n",
			qname, nameserver, dns.RcodeToString[r.Rcode])
		return
	} else {
		if checkFakeIP(r) {
			log.Printf("resolved %s on %s hit fake ip cache\n", qname, nameserver)
			return
		}
	}

	select {
	case res <- r:
		log.Printf("success to resolv %s on %s: %v\n", qname, nameserver, r)
	}
}

func selectMsg(gMsg, cMsg *dns.Msg) (*dns.Msg, error) {
	if gMsg != nil {
		return gMsg, nil
	}
	return cMsg, nil
}

func (r *Resolver) AllNameservers() []string {
	var ns []string
	ns = append(ns, gConfig.Nameservers...)
	ns = append(ns, gConfig.CHNameservers...)
	ns = append(ns, gConfig.ISPNameservers...)
	return ns
}

func (r *Resolver) NonISPNameservers() []string {
	var ns []string
	ns = append(ns, gConfig.Nameservers...)
	ns = append(ns, gConfig.CHNameservers...)
	return ns
}

// Nameservers return the array of nameservers
func (r *Resolver) Nameservers() []string {
	return gConfig.Nameservers
}

func (r *Resolver) CHNameservers() []string {
	return gConfig.CHNameservers
}

func (r *Resolver) ISPNameservers() []string {
	return gConfig.ISPNameservers
}

// Timeout returns the resolver timeout
func (r *Resolver) Timeout() time.Duration {
	return time.Duration(gConfig.Timeout) * time.Second
}

func (r *Resolver) SessionTimeout() time.Duration {
	return time.Duration(gConfig.SessionTimeout) * time.Second
}
