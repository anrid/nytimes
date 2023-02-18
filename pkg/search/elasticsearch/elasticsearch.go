package search

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/anrid/nytimes/pkg/esfasthttp"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/goccy/go-json"
)

var (
	truee = true
)

type ES struct {
	es *elasticsearch.Client

	bulkIndexDocs int64
	bulkIndexSecs float64
	verboseOutput bool
}

func New(addrs []string, verboseOutput bool) *ES {
	var err error

	s := &ES{
		verboseOutput: verboseOutput,
	}

	config := elasticsearch.Config{
		Transport: esfasthttp.NewLoggingTransport(),
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

	fmt.Println("Pinged ES successfully")

	return s
}

func (s *ES) Search(ctx context.Context, queryJSON []byte, indexName string, useCache bool) (hits map[string]interface{}) {
	res, err := esapi.SearchRequest{
		Index:        []string{indexName},
		Body:         bytes.NewReader(queryJSON),
		Pretty:       true,
		RequestCache: &useCache,
	}.Do(ctx, s.es)
	PanicOnError(res, err)
	defer res.Body.Close()

	hits = make(map[string]interface{})
	err = json.NewDecoder(res.Body).Decode(&hits)
	if err != nil {
		log.Panic(err)
	}

	return
}

func (s *ES) CreateIndex(ctx context.Context, mappingsJSONFile, indexName string) {
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
		Body:   bytes.NewReader(ReadJSONFile(mappingsJSONFile)),
		Pretty: true,
	}.Do(ctx, s.es)
	PanicOnError(res, err)

	fmt.Printf("Created new index `%s` (status: %d)\n", indexName, res.StatusCode)
}

func (s *ES) BulkIndex(ctx context.Context, indexName string, docIDs []string, docs []interface{}) {
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

	if s.verboseOutput {
		fmt.Printf("Bulk indexed %d docs (status: %d)\n", count, res.StatusCode)
	}
}

func (s *ES) PrintBulkIndexingRate() {
	fmt.Printf("Bulk indexing rate: %.02f docs / sec\n", float64(s.bulkIndexDocs)/s.bulkIndexSecs)
}

func ReadJSONFile(file string) []byte {
	data, err := os.ReadFile(file)
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
