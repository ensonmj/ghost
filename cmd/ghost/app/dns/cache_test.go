package dns

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func TestCache(t *testing.T) {
	const (
		testDomain = "www.google.com"
	)

	cache := &MemoryCache{
		Backend:  make(map[string]Mesg, gConfig.Maxcount),
		Expire:   time.Duration(gConfig.Expire) * time.Second,
		Maxcount: gConfig.Maxcount,
	}

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(testDomain), dns.TypeA)

	if err := cache.Set(testDomain, m, true); err != nil {
		t.Error(err)
	}

	if _, _, err := cache.Get(testDomain); err != nil && err.Error() != fmt.Sprintf("%s expired", testDomain) {
		t.Error(err)
	}

	cache.Remove(testDomain)

	if _, _, err := cache.Get(testDomain); err == nil {
		t.Error("cache entry still existed after remove")
	}
}

func TestBlockCache(t *testing.T) {
	const (
		testDomain = "www.google.com"
	)

	cache := &MemoryBlockCache{
		Backend: make(map[string]bool),
	}

	if err := cache.Set(testDomain, true); err != nil {
		t.Error(err)
	}

	if exists := cache.Exists(testDomain); !exists {
		t.Error(testDomain, "didnt exist in block cache")
	}

	if exists := cache.Exists(strings.ToUpper(testDomain)); !exists {
		t.Error(strings.ToUpper(testDomain), "didnt exist in block cache")
	}

	if _, err := cache.Get(testDomain); err != nil {
		t.Error(err)
	}

	if exists := cache.Exists(fmt.Sprintf("%sfuzz", testDomain)); exists {
		t.Error("fuzz existed in block cache")
	}
}
