package main

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	spectator "github.com/Netflix/spectator-go/v2/spectator"
)

const maxDurationSecs = 2 * 60

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: perfTest [writer_type] [buffering]")
	fmt.Fprintln(os.Stderr, "  writer_type: udp or uds")
	fmt.Fprintln(os.Stderr, "  buffering: 0 for disabled, 1 for enabled (default is 0)")
}

func Run(args []string) int {
	if len(args) < 1 || len(args) > 2 {
		printUsage()
		return 1
	}

	writerType := args[0]
	if writerType != "udp" && writerType != "uds" {
		fmt.Fprintf(os.Stderr, "Invalid writer type: %s\n", writerType)
		printUsage()
		return 1
	}

	bufferingEnabled := false
	if len(args) == 2 {
		switch args[1] {
		case "1":
			bufferingEnabled = true
		case "0":
			bufferingEnabled = false
		default:
			fmt.Fprintf(os.Stderr, "Invalid buffering argument: %s\n", args[1])
			printUsage()
			return 1
		}
	}

	counterName := "udp_test_counter"
	locationTag := "udp"
	writerTypeName := "UDP"
	if writerType == "uds" {
		counterName = "unix_test_counter"
		locationTag = "unix"
		writerTypeName = "UDS"
	}

	numThreads := 1
	if bufferingEnabled {
		numThreads = 4
	}

	fmt.Println("Running performance test with the following configuration:")
	fmt.Printf("Writer Type: %s\n", writerTypeName)
	fmt.Printf("Buffering Enabled: %v\n", bufferingEnabled)

	// LowLatencyBuffer is the most performant Go option:
	// double-buffered, CPU-sharded, ~1M lines/sec, 0.6–7 us latency.
	// Minimum size: 2 * NumCPU * 60KB. Use a generous multiple to avoid overflows.
	var cfg *spectator.Config
	var err error
	if bufferingEnabled {
		bufferSize := 4 * runtime.NumCPU() * 60 * 1024
		cfg, err = spectator.NewConfigWithBuffer(writerType, nil, nil, bufferSize, 5*time.Second)
	} else {
		cfg, err = spectator.NewConfig(writerType, nil, nil)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create config: %v\n", err)
		return 1
	}

	registry, err := spectator.NewRegistry(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create registry: %v\n", err)
		return 1
	}
	defer registry.Close()

	tags := map[string]string{
		"location": locationTag,
		"version":  "correct-horse-battery-staple",
	}

	fmt.Printf("Running performance test with %d threads...\n", numThreads)

	var iterations atomic.Uint64
	var wg sync.WaitGroup
	stop := make(chan struct{})

	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					registry.Counter(counterName, tags).Increment()
					iterations.Add(1)
				}
			}
		}()
	}

	start := time.Now()
	time.Sleep(maxDurationSecs * time.Second)
	close(stop)
	wg.Wait()

	elapsed := time.Since(start).Seconds()
	total := iterations.Load()
	rate := float64(total) / elapsed

	fmt.Println("\nPerformance Test Summary:")
	fmt.Printf("Threads used: %d\n", numThreads)
	fmt.Printf("Iterations completed: %d\n", total)
	fmt.Printf("Total elapsed time: %.2f seconds\n", elapsed)
	fmt.Printf("Rate: %.2f iterations/second\n", rate)
	fmt.Printf("Rate per thread: %.2f iterations/second/thread\n", rate/float64(numThreads))
	return 0
}
