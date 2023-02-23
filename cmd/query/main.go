package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/anrid/nytimes/pkg/search/es"
	"github.com/anrid/nytimes/pkg/util"
	"github.com/spf13/pflag"
)

var (
	count      = pflag.Int("count", 10, "number of calls to search engine")
	indexName  = pflag.String("index", "nytimes-articles", "search engine index name")
	useCache   = pflag.Bool("cache", true, "enable search engine caching")
	dumpResult = pflag.Bool("dump", false, "dump search engine result of first query")
	queryJSON  = pflag.String("query", "./assets/mappings/nytimes/query-simple.json", "query to run (path to JSON file)")
	numThreads = pflag.Int("threads", 10, "number of threads to run benchmark in concurrently")
)

func main() {
	pflag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 1_000*time.Millisecond)
	defer cancel()

	s := es.New([]string{
		"http://localhost:9200",
		"http://localhost:9201",
		"http://localhost:9202",
	}, true)

	// Load query from JSON file.
	query := es.ReadJSONFile(*queryJSON)
	fmt.Printf("Query payload:\n%s\n", string(query))

	statsBefore := s.Stats(ctx)

	t := time.Now()
	hits := s.Search(ctx, query, *indexName, *useCache)

	if *dumpResult {
		util.Dump(hits)
	}

	fmt.Printf("Completed first request in %s\n", time.Since(t))

	if *count < 1 {
		fmt.Printf("count = 0, exiting!\n")
		os.Exit(0)
	}

	// Run benchmark
	var wg sync.WaitGroup
	var mux sync.Mutex
	t = time.Now()
	var totalReqs int

	for i := 0; i < *numThreads; i++ {
		wg.Add(1)

		go func(num int) {
			var durations []time.Duration
			t := time.Now()
			ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			fmt.Printf("[Thread %02d] running benchmark, %d iterations\n", num, *count)

			for i := 0; i < *count; i++ {
				t0 := time.Now()

				hits2 := s.Search(ctx, query, *indexName, *useCache)
				if len(hits) != len(hits2) {
					log.Panicf("hits size is %d but expected %d", len(hits2), len(hits))
				}

				durations = append(durations, time.Since(t0))

				mux.Lock()
				totalReqs++
				mux.Unlock()
			}

			sorted := append(make([]time.Duration, 0), durations...)
			sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

			fmt.Printf(
				"[Thread %02d] completed %d requests in %s (%.1f req/s) | min: %s / max: %s / mean: %s\n",
				num,
				*count,
				time.Since(t),
				float64(*count)/time.Since(t).Seconds(),
				sorted[0],
				sorted[(len(sorted)-1)],
				sorted[(len(sorted)-1)/2],
			)

			wg.Done()
		}(i + 1)
	}

	// Wait for all threads to finish.
	wg.Wait()

	fmt.Printf(
		"Done. Completed %d requests total in %s (%.1f req/s)\n",
		totalReqs,
		time.Since(t),
		float64(totalReqs)/time.Since(t).Seconds(),
	)

	statsAfter := s.Stats(ctx)

	qcMissDiff := statsAfter.All.Total.QueryCache.MissCount - statsBefore.All.Total.QueryCache.MissCount
	qcHitsDiff := statsAfter.All.Total.QueryCache.HitCount - statsBefore.All.Total.QueryCache.HitCount
	rcMissDiff := statsAfter.All.Total.RequestCache.MissCount - statsBefore.All.Total.RequestCache.MissCount
	rcHitsDiff := statsAfter.All.Total.RequestCache.HitCount - statsBefore.All.Total.RequestCache.HitCount

	fmt.Printf("Query cache   : +%-3d hits / +%-3d miss\n", qcHitsDiff, qcMissDiff)
	fmt.Printf("Request cache : +%-3d hits / +%-3d miss\n", rcHitsDiff, rcMissDiff)
}
