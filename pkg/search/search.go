package search

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/anrid/nytimes/pkg/esfasthttp"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/goccy/go-json"
)

type Search struct {
	es *elasticsearch.Client

	bulkIndexDocs int64
	bulkIndexSecs float64
}

func New(addrs []string) *Search {
	var err error
	s := new(Search)

	config := elasticsearch.Config{
		Transport: new(esfasthttp.Transport),
	}
	config.Addresses = append(config.Addresses, addrs...)

	s.es, err = elasticsearch.NewClient(config)
	if err != nil {
		log.Fatalf("error creating the client: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Perform ping to ensure that ES can be reached.
	for retries := 10; retries > 0; retries-- {
		var retry bool

		res, err := s.es.Ping(
			s.es.Ping.WithContext(ctx),
			s.es.Ping.WithErrorTrace(),
		)
		if err != nil {
			fmt.Printf("Pinging ES failed: %s\n", err)
			retry = true
		}
		if res.IsError() {
			fmt.Printf("Pinging ES returned status code: %d\n", res.StatusCode)
			body, _ := io.ReadAll(res.Body)
			fmt.Printf("Got error response: %s", string(body))
			retry = true
		}

		if !retry {
			break
		}

		fmt.Printf("Retrying ping in 1 sec (%d retries remaining) ..", retries)
		time.Sleep(time.Second)
	}

	fmt.Println("Connected to ES successfully")

	return s
}

func (s *Search) CreateIndex(ctx context.Context, mappingsDir, indexName string) {
	truee := true

	// Delete test index if it exists.
	res, err := esapi.IndicesDeleteRequest{
		Index:             []string{indexName},
		IgnoreUnavailable: &truee,
		Pretty:            true,
	}.Do(ctx, s.es)
	PanicOnError(res, err)

	fmt.Printf("Deleted existing index `%s` (status: %d)\n", indexName, res.StatusCode)

	// Create a new test index.
	res, err = esapi.IndicesCreateRequest{
		Index:  indexName,
		Body:   bytes.NewReader(ReadJSONFile(mappingsDir, "index-mappings.json")),
		Pretty: true,
	}.Do(ctx, s.es)
	PanicOnError(res, err)

	fmt.Printf("Created new index `%s` (status: %d)\n", indexName, res.StatusCode)
}

func (s *Search) BulkIndex(ctx context.Context, indexName string, docIDs []string, docs []interface{}) {
	if len(docIDs) == 0 || len(docIDs) != len(docs) {
		log.Fatalf("got %d doc IDs but %d docs", len(docIDs), len(docs))
	}

	// Bulk index documents.
	var sb strings.Builder
	var count int64

	for i, id := range docIDs {
		count++

		sb.WriteString(`{"create":{"_id":"`)
		sb.WriteString(id)
		sb.WriteString(`"}}`)
		sb.WriteRune('\n')

		docJ, err := json.Marshal(docs[i])
		if err != nil {
			log.Fatalf("could not marshal doc id %s : %s", id, err)
		}

		sb.Write(docJ)
		sb.WriteRune('\n')
	}

	timer := time.Now()

	res, err := esapi.BulkRequest{
		Index: indexName,
		Body:  strings.NewReader(sb.String()),
	}.Do(ctx, s.es)
	PanicOnError(res, err)

	s.bulkIndexDocs += count
	s.bulkIndexSecs += time.Since(timer).Seconds()

	fmt.Printf("Bulk indexed %d docs (status: %d)\n", count, res.StatusCode)
}

func (s *Search) PrintBulkIndexingRate() {
	fmt.Printf("Bulk indexing rate: %.02f docs / sec\n", float64(s.bulkIndexDocs)/s.bulkIndexSecs)
}

func ReadJSONFile(mappingsDir, jsonFile string) []byte {
	data, err := os.ReadFile(path.Join("data", "mappings", mappingsDir, jsonFile))
	if err != nil {
		log.Panic(err)
	}
	return data
}

func PanicOnError(res *esapi.Response, err error) {
	if err != nil {
		log.Panicf("Error getting response: %s", err)
	}
	if res.IsError() {
		log.Panicf("Error response: %s", res)
	}
}
