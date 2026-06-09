package benchmark

import (
	"sort"
	"sync"
	"time"

	"github.com/miekg/dns"
)

type Result struct {
	cfg       Config
	startedAt time.Time
	endedAt   time.Time

	mu        sync.Mutex
	resolvers map[string]*resolverStats
}

type resolverStats struct {
	total     int
	responses int
	errors    int
	rcodes    map[string]int
	latencies []time.Duration
}

type Summary struct {
	Domain     string            `json:"domain"`
	QueryType  string            `json:"query_type"`
	Protocol   string            `json:"protocol"`
	Elapsed    time.Duration     `json:"elapsed"`
	TargetQPS  int               `json:"target_qps"`
	ActualQPS  float64           `json:"actual_qps"`
	Total      int               `json:"total"`
	Responses  int               `json:"responses"`
	Errors     int               `json:"errors"`
	RCodes     map[string]int    `json:"rcodes"`
	LatencyAvg time.Duration     `json:"latency_avg"`
	LatencyP50 time.Duration     `json:"latency_p50"`
	LatencyP95 time.Duration     `json:"latency_p95"`
	LatencyP99 time.Duration     `json:"latency_p99"`
	Resolvers  []ResolverSummary `json:"resolvers"`
}

type ResolverSummary struct {
	Resolver   string         `json:"resolver"`
	Total      int            `json:"total"`
	Responses  int            `json:"responses"`
	Errors     int            `json:"errors"`
	RCodes     map[string]int `json:"rcodes"`
	LatencyAvg time.Duration  `json:"latency_avg"`
	LatencyP50 time.Duration  `json:"latency_p50"`
	LatencyP95 time.Duration  `json:"latency_p95"`
	LatencyP99 time.Duration  `json:"latency_p99"`
}

func newResult(cfg Config) *Result {
	resolvers := make(map[string]*resolverStats, len(cfg.Resolvers))
	for _, resolver := range cfg.Resolvers {
		resolvers[resolver] = &resolverStats{
			rcodes: make(map[string]int),
		}
	}

	return &Result{
		cfg:       cfg,
		startedAt: time.Now(),
		resolvers: resolvers,
	}
}

func (r *Result) finish() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.endedAt = time.Now()
}

func (r *Result) recordError(resolver string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	stats := r.getResolverStats(resolver)
	stats.total++
	stats.errors++
}

func (r *Result) recordResponse(resolver string, latency time.Duration, rcode int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	stats := r.getResolverStats(resolver)
	stats.total++
	stats.responses++
	stats.latencies = append(stats.latencies, latency)
	stats.rcodes[dns.RcodeToString[rcode]]++
}

func (r *Result) getResolverStats(resolver string) *resolverStats {
	stats, ok := r.resolvers[resolver]
	if ok {
		return stats
	}

	stats = &resolverStats{rcodes: make(map[string]int)}
	r.resolvers[resolver] = stats
	return stats
}

func (r *Result) Summary() Summary {
	r.mu.Lock()
	defer r.mu.Unlock()

	endedAt := r.endedAt
	if endedAt.IsZero() {
		endedAt = time.Now()
	}
	elapsed := endedAt.Sub(r.startedAt)

	summary := Summary{
		Domain:    r.cfg.Domain,
		QueryType: r.cfg.QueryType,
		Protocol:  r.cfg.Protocol,
		Elapsed:   elapsed,
		TargetQPS: r.cfg.Rate,
		RCodes:    make(map[string]int),
	}

	var allLatencies []time.Duration
	for resolver, stats := range r.resolvers {
		resolverSummary := summarizeResolver(resolver, stats)
		summary.Resolvers = append(summary.Resolvers, resolverSummary)

		summary.Total += stats.total
		summary.Responses += stats.responses
		summary.Errors += stats.errors
		for rcode, count := range stats.rcodes {
			summary.RCodes[rcode] += count
		}
		allLatencies = append(allLatencies, stats.latencies...)
	}

	sort.Slice(summary.Resolvers, func(i, j int) bool {
		return summary.Resolvers[i].Resolver < summary.Resolvers[j].Resolver
	})

	if elapsed > 0 {
		summary.ActualQPS = float64(summary.Total) / elapsed.Seconds()
	}
	summary.LatencyAvg = averageDuration(allLatencies)
	summary.LatencyP50 = percentileDuration(allLatencies, 50)
	summary.LatencyP95 = percentileDuration(allLatencies, 95)
	summary.LatencyP99 = percentileDuration(allLatencies, 99)

	return summary
}

func summarizeResolver(resolver string, stats *resolverStats) ResolverSummary {
	rcodes := make(map[string]int, len(stats.rcodes))
	for rcode, count := range stats.rcodes {
		rcodes[rcode] = count
	}

	return ResolverSummary{
		Resolver:   resolver,
		Total:      stats.total,
		Responses:  stats.responses,
		Errors:     stats.errors,
		RCodes:     rcodes,
		LatencyAvg: averageDuration(stats.latencies),
		LatencyP50: percentileDuration(stats.latencies, 50),
		LatencyP95: percentileDuration(stats.latencies, 95),
		LatencyP99: percentileDuration(stats.latencies, 99),
	}
}

func averageDuration(values []time.Duration) time.Duration {
	if len(values) == 0 {
		return 0
	}

	var total time.Duration
	for _, value := range values {
		total += value
	}
	return total / time.Duration(len(values))
}

func percentileDuration(values []time.Duration, percentile int) time.Duration {
	if len(values) == 0 {
		return 0
	}

	sorted := append([]time.Duration(nil), values...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	if percentile <= 0 {
		return sorted[0]
	}
	if percentile >= 100 {
		return sorted[len(sorted)-1]
	}

	index := (len(sorted)*percentile + 99) / 100
	if index <= 0 {
		index = 1
	}
	return sorted[index-1]
}
