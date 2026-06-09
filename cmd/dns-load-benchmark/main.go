package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Daofengql/DNS-Load-Benchmark/internal/benchmark"
)

func main() {
	var cfg benchmark.Config
	var jsonOutput bool
	var resolvers resolverList
	var resolverFile string

	flag.Var(&resolvers, "resolver", "DNS resolver to use, host:port format. Can be repeated")
	flag.StringVar(&resolverFile, "resolver-file", "", "read resolvers from a text file, one host:port per line")
	flag.StringVar(&cfg.Domain, "domain", "", "base domain to query, required")
	flag.StringVar(&cfg.QueryType, "type", "A", "DNS query type, such as A, AAAA, CNAME, MX, NS, TXT")
	flag.StringVar(&cfg.Protocol, "protocol", "udp", "transport protocol: udp or tcp")
	flag.DurationVar(&cfg.Duration, "duration", 30*time.Second, "benchmark duration")
	flag.IntVar(&cfg.Rate, "rate", 100, "target query rate per second")
	flag.IntVar(&cfg.Concurrency, "concurrency", 16, "number of concurrent workers")
	flag.DurationVar(&cfg.Timeout, "timeout", 2*time.Second, "timeout for a single query")
	flag.BoolVar(&cfg.RandomPrefix, "random-prefix", true, "prepend random labels to avoid cache hits")
	flag.IntVar(&cfg.LabelDepth, "label-depth", 2, "number of random labels when random-prefix is enabled")
	flag.BoolVar(&jsonOutput, "json", false, "print machine-readable JSON summary")
	flag.Parse()

	cfg.Resolvers = append(cfg.Resolvers, resolvers...)
	fileResolvers, err := readResolverFile(resolverFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read resolver file: %v\n", err)
		os.Exit(2)
	}
	cfg.Resolvers = append(cfg.Resolvers, fileResolvers...)

	if cfg.Domain == "" {
		fmt.Fprintln(os.Stderr, "error: -domain is required")
		flag.Usage()
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if len(cfg.Resolvers) == 0 {
		cfg.Resolvers = []string{"127.0.0.1:53"}
	}

	fmt.Fprintf(os.Stderr, "Testing %d resolver(s) for %s at %d qps during %s\n", len(cfg.Resolvers), cfg.Domain, cfg.Rate, cfg.Duration)
	fmt.Fprintln(os.Stderr, "Only run this tool against DNS infrastructure you own or have explicit permission to test.")

	result, err := benchmark.Run(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "benchmark failed: %v\n", err)
		os.Exit(1)
	}

	summary := result.Summary()
	if jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(summary); err != nil {
			fmt.Fprintf(os.Stderr, "failed to encode JSON: %v\n", err)
			os.Exit(1)
		}
		return
	}

	fmt.Printf("\nDNS load benchmark summary\n")
	fmt.Printf("Resolvers:     %d\n", len(summary.Resolvers))
	fmt.Printf("Domain:        %s\n", summary.Domain)
	fmt.Printf("Type/Protocol: %s/%s\n", summary.QueryType, summary.Protocol)
	fmt.Printf("Duration:      %s\n", summary.Elapsed)
	fmt.Printf("Target QPS:    %d\n", summary.TargetQPS)
	fmt.Printf("Actual QPS:    %.2f\n", summary.ActualQPS)
	fmt.Printf("Total:         %d\n", summary.Total)
	fmt.Printf("Responses:     %d\n", summary.Responses)
	fmt.Printf("Errors:        %d\n", summary.Errors)
	fmt.Printf("Latency avg:   %s\n", summary.LatencyAvg)
	fmt.Printf("Latency p50:   %s\n", summary.LatencyP50)
	fmt.Printf("Latency p95:   %s\n", summary.LatencyP95)
	fmt.Printf("Latency p99:   %s\n", summary.LatencyP99)

	if len(summary.RCodes) > 0 {
		fmt.Println("RCODE:")
		for rcode, count := range summary.RCodes {
			fmt.Printf("  %-12s %d\n", rcode, count)
		}
	}

	if len(summary.Resolvers) > 1 {
		fmt.Println("Per resolver:")
		for _, resolver := range summary.Resolvers {
			fmt.Printf("  %s total=%d responses=%d errors=%d avg=%s p95=%s\n",
				resolver.Resolver,
				resolver.Total,
				resolver.Responses,
				resolver.Errors,
				resolver.LatencyAvg,
				resolver.LatencyP95,
			)
		}
	}
}

type resolverList []string

func (r *resolverList) String() string {
	return strings.Join(*r, ",")
}

func (r *resolverList) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("empty resolver")
	}
	*r = append(*r, value)
	return nil
}

func readResolverFile(path string) ([]string, error) {
	if path == "" {
		return nil, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var resolvers []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		resolvers = append(resolvers, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return resolvers, nil
}
