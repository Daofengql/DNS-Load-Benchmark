package benchmark

import (
	"testing"
	"time"
)

func TestConfigNormalizedDefaultsResolver(t *testing.T) {
	cfg, err := Config{
		Domain:      "example.com",
		QueryType:   "a",
		Protocol:    "UDP",
		Duration:    time.Second,
		Rate:        10,
		Concurrency: 2,
		Timeout:     time.Second,
	}.normalized()
	if err != nil {
		t.Fatalf("normalized returned error: %v", err)
	}

	if got, want := cfg.Resolvers[0], "127.0.0.1:53"; got != want {
		t.Fatalf("resolver = %q, want %q", got, want)
	}
	if got, want := cfg.Domain, "example.com."; got != want {
		t.Fatalf("domain = %q, want %q", got, want)
	}
	if got, want := cfg.QueryType, "A"; got != want {
		t.Fatalf("query type = %q, want %q", got, want)
	}
}

func TestConfigNormalizedResolverList(t *testing.T) {
	cfg, err := Config{
		Resolvers:   []string{"127.0.0.1", "127.0.0.1:53", "dns.example.net:5353"},
		Domain:      "example.com",
		QueryType:   "AAAA",
		Protocol:    "tcp",
		Duration:    time.Second,
		Rate:        10,
		Concurrency: 2,
		Timeout:     time.Second,
	}.normalized()
	if err != nil {
		t.Fatalf("normalized returned error: %v", err)
	}

	if got, want := len(cfg.Resolvers), 2; got != want {
		t.Fatalf("resolver count = %d, want %d", got, want)
	}
}
