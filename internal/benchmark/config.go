package benchmark

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

type Config struct {
	Resolvers    []string
	Domain       string
	QueryType    string
	Protocol     string
	Duration     time.Duration
	Rate         int
	Concurrency  int
	Timeout      time.Duration
	RandomPrefix bool
	LabelDepth   int
}

func (c Config) normalized() (Config, error) {
	if len(c.Resolvers) == 0 {
		c.Resolvers = []string{"127.0.0.1:53"}
	}

	cleanResolvers := make([]string, 0, len(c.Resolvers))
	seen := make(map[string]struct{}, len(c.Resolvers))
	for _, resolver := range c.Resolvers {
		resolver = normalizeResolver(resolver)
		if resolver == "" {
			continue
		}
		if _, ok := seen[resolver]; ok {
			continue
		}
		seen[resolver] = struct{}{}
		cleanResolvers = append(cleanResolvers, resolver)
	}
	if len(cleanResolvers) == 0 {
		return c, fmt.Errorf("at least one resolver is required")
	}
	c.Resolvers = cleanResolvers

	if strings.TrimSpace(c.Domain) == "" {
		return c, fmt.Errorf("domain is required")
	}
	c.Domain = dns.Fqdn(strings.TrimSpace(c.Domain))

	c.QueryType = strings.ToUpper(strings.TrimSpace(c.QueryType))
	if _, ok := dns.StringToType[c.QueryType]; !ok {
		return c, fmt.Errorf("unsupported DNS query type: %s", c.QueryType)
	}

	c.Protocol = strings.ToLower(strings.TrimSpace(c.Protocol))
	if c.Protocol != "udp" && c.Protocol != "tcp" {
		return c, fmt.Errorf("protocol must be udp or tcp")
	}

	if c.Duration <= 0 {
		return c, fmt.Errorf("duration must be greater than zero")
	}
	if c.Rate <= 0 {
		return c, fmt.Errorf("rate must be greater than zero")
	}
	if c.Concurrency <= 0 {
		return c, fmt.Errorf("concurrency must be greater than zero")
	}
	if c.Timeout <= 0 {
		return c, fmt.Errorf("timeout must be greater than zero")
	}
	if c.LabelDepth < 1 {
		c.LabelDepth = 1
	}
	if c.LabelDepth > 8 {
		return c, fmt.Errorf("label-depth should not exceed 8")
	}

	return c, nil
}

func normalizeResolver(resolver string) string {
	resolver = strings.TrimSpace(resolver)
	if resolver == "" {
		return ""
	}

	if _, _, err := net.SplitHostPort(resolver); err == nil {
		return resolver
	}

	if strings.Count(resolver, ":") == 0 {
		return net.JoinHostPort(resolver, "53")
	}

	return resolver
}
