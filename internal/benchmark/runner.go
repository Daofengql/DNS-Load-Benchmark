package benchmark

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
)

type job struct {
	id       uint64
	resolver string
}

func Run(ctx context.Context, cfg Config) (*Result, error) {
	cfg, err := cfg.normalized()
	if err != nil {
		return nil, err
	}

	runCtx, cancel := context.WithTimeout(ctx, cfg.Duration)
	defer cancel()

	result := newResult(cfg)
	jobs := make(chan job, cfg.Concurrency*2)

	var sequence uint64
	var workers sync.WaitGroup
	for i := 0; i < cfg.Concurrency; i++ {
		workers.Add(1)
		go func(workerID int) {
			defer workers.Done()
			runWorker(runCtx, cfg, jobs, result, workerID)
		}(i)
	}

	interval := time.Second / time.Duration(cfg.Rate)
	if interval <= 0 {
		interval = time.Nanosecond
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	resolverIndex := 0
	for {
		select {
		case <-runCtx.Done():
			close(jobs)
			workers.Wait()
			result.finish()
			return result, nil
		case <-ticker.C:
			id := atomic.AddUint64(&sequence, 1)
			nextJob := job{
				id:       id,
				resolver: cfg.Resolvers[resolverIndex%len(cfg.Resolvers)],
			}
			select {
			case jobs <- nextJob:
				resolverIndex++
			case <-runCtx.Done():
				close(jobs)
				workers.Wait()
				result.finish()
				return result, nil
			}
		}
	}
}

func runWorker(ctx context.Context, cfg Config, jobs <-chan job, result *Result, workerID int) {
	client := &dns.Client{
		Net:     cfg.Protocol,
		Timeout: cfg.Timeout,
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)*7919))
	qtype := dns.StringToType[cfg.QueryType]

	for item := range jobs {
		name := cfg.Domain
		if cfg.RandomPrefix {
			name = randomQName(rng, cfg.LabelDepth, cfg.Domain)
		}

		msg := new(dns.Msg)
		msg.SetQuestion(name, qtype)
		msg.Id = dns.Id()

		start := time.Now()
		queryCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
		response, _, err := client.ExchangeContext(queryCtx, msg, item.resolver)
		cancel()
		elapsed := time.Since(start)

		if err != nil {
			result.recordError(item.resolver)
			continue
		}

		result.recordResponse(item.resolver, elapsed, response.Rcode)
		_ = item.id
	}
}

func randomQName(rng *rand.Rand, depth int, domain string) string {
	labels := make([]byte, 0, depth*9+len(domain))
	for i := 0; i < depth; i++ {
		if i > 0 {
			labels = append(labels, '.')
		}
		labels = append(labels, randomLabel(rng)...)
	}
	labels = append(labels, '.')
	labels = append(labels, domain...)
	return string(labels)
}

func randomLabel(rng *rand.Rand) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	const minLength = 6
	const maxLength = 12

	length := minLength + rng.Intn(maxLength-minLength+1)
	buf := make([]byte, length)
	for i := range buf {
		buf[i] = charset[rng.Intn(len(charset))]
	}
	return string(buf)
}
