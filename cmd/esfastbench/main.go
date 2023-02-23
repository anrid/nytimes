package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	search "github.com/anrid/nytimes/pkg/search/es"
	"github.com/anrid/nytimes/pkg/util"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/spf13/pflag"
)

var (
	count               = pflag.Int("count", 100, "number of calls to Elasticsearch")
	useDefaultTransport = pflag.Bool("slow", false, "use default HTTP transport instead of FastHTTP")
	buildIndex          = pflag.Bool("index", false, "create a new test index")
)

func main() {
	pflag.Parse()

	// var durations []time.Duration
	config := elasticsearch.Config{}
	if !*useDefaultTransport {
		fmt.Printf("Using FastHTTP transport\n")
		config.Transport = new(search.Transport)
	}

	es, err := elasticsearch.NewClient(config)
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1_000*time.Millisecond)
	defer cancel()

	indexName := "test-index"

	if *buildIndex {
		truee := true

		// Delete test index if it exists.
		res, err := esapi.IndicesDeleteRequest{
			Index:             []string{indexName},
			IgnoreUnavailable: &truee,
			Pretty:            true,
		}.Do(ctx, es)
		search.PanicOnError(res, err)

		fmt.Printf("Deleted existing index `%s` (status: %d)\n", indexName, res.StatusCode)

		// Create a new test index.
		res, err = esapi.IndicesCreateRequest{
			Index:  indexName,
			Body:   bytes.NewReader(search.ReadJSONFile("./assets/mappings/test/index-mappings.json")),
			Pretty: true,
		}.Do(ctx, es)
		search.PanicOnError(res, err)

		fmt.Printf("Created new index `%s` (status: %d)\n", indexName, res.StatusCode)

		res, err = esapi.IndexRequest{
			Index:      indexName,
			DocumentID: "a1",
			Body:       bytes.NewReader(search.ReadJSONFile("./assets/mappings/test/article-a1.json")),
		}.Do(ctx, es)
		search.PanicOnError(res, err)

		fmt.Printf("Indexed doc: `%s` (status: %d)\n", "a1", res.StatusCode)

		res, err = esapi.IndexRequest{
			Index:      indexName,
			DocumentID: "a2",
			Body:       bytes.NewReader(search.ReadJSONFile("./assets/mappings/test/article-a2.json")),
			Refresh:    "true",
		}.Do(ctx, es)
		search.PanicOnError(res, err)

		fmt.Printf("Indexed doc: `%s` (status: %d)\n", "a2", res.StatusCode)
	}

	// Test simple query.
	query := search.ReadJSONFile("./assets/mappings/test/query.json")

	res, err := esapi.SearchRequest{
		Index:  []string{indexName},
		Body:   bytes.NewReader(query),
		Pretty: true,
	}.Do(ctx, es)
	search.PanicOnError(res, err)

	hits := make(map[string]interface{})
	err = json.NewDecoder(res.Body).Decode(&hits)
	if err != nil {
		log.Panic(err)
	}

	fmt.Printf("Got search result (status: %d)\n", res.StatusCode)
	util.Dump(hits)

	// Run benchmark
	var durations []time.Duration
	t := time.Now()
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Printf("Running benchmark, iteration count: %d\n", *count)

	for i := 0; i < *count; i++ {
		t0 := time.Now()

		res, err := esapi.SearchRequest{
			Index:  []string{indexName},
			Body:   bytes.NewReader(query),
			Pretty: true,
		}.Do(ctx, es)
		search.PanicOnError(res, err)

		durations = append(durations, time.Since(t0))
		if err != nil {
			log.Fatalf("Error: %s", err)
		}
		res.Body.Close()
	}

	sorted := append(make([]time.Duration, 0), durations...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	log.Printf(
		"%d requests in %s (%.1f req/s) | min: %s / max: %s / mean: %s",
		*count,
		time.Since(t),
		float64(*count)/time.Since(t).Seconds(),
		sorted[0],
		sorted[(len(sorted)-1)],
		sorted[(len(sorted)-1)/2],
	)
}
