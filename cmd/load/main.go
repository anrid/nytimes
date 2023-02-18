package main

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anrid/nytimes/pkg/domain"
	es "github.com/anrid/nytimes/pkg/search/elasticsearch"
	"github.com/anrid/nytimes/pkg/stats"
	"github.com/goccy/go-json"
	"github.com/spf13/pflag"
)

var (
	indexName = "nytimes-articles"

	gzipDir       = pflag.String("dir", "data/", "Directory with GZIP files containing New York Times articles (filenames must end in `.json.gz`)")
	startFrom     = pflag.String("start-from", "", "File to (re)start from")
	detailedStats = pflag.Bool("detailed-stats", false, "Calculate detailed stats (WARN: requires lots of memory, use only when indexing few docs)")
	maxBulk       = pflag.Int("max-bulk", 5_000, "Max number of docs to index in bulk")
	maxDocs       = pflag.Int("max-docs", 0, "Max number of docs to index")
	createIndex   = pflag.Bool("create-index", false, "Drop and recreate a new index")
	verbose       = pflag.BoolP("verbose", "v", false, "Verbose output")
	useIndexer    = pflag.String("indexer", "es", "Indexer to use, available: ['es']")
)

func main() {
	pflag.Parse()

	if *gzipDir == "" {
		pflag.Usage()
		log.Fatalf("missing --dir arg")
	}

	var indexer domain.Indexer
	switch strings.ToLower(*useIndexer) {
	case "es":
		indexer = es.New(nil, *verbose)
	default:
		pflag.Usage()
		log.Fatalf("incorrect --indexer arg")
	}

	fmt.Printf("Reading dir: %s\n", *gzipDir)
	des, err := os.ReadDir(*gzipDir)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3_000*time.Millisecond)
	defer cancel()

	if *createIndex {
		indexer.CreateIndex(ctx, "./assets/mappings/nytimes/index-mappings.json", indexName)
	}

	var articlesTotal int
	var docs []interface{}
	var docIDs []string
	var startFromFileFound bool
	var lastHeadline string
	var lastPubDate string

	st := stats.New()
	timer := time.Now()

	for _, de := range des {
		if !strings.HasSuffix(de.Name(), ".json.gz") {
			if *verbose {
				fmt.Printf("Skipping file: %s\n", de.Name())
			}
			continue
		} else if de.IsDir() {
			if *verbose {
				fmt.Printf("Skipping dir: %s\n", de.Name())
			}
			continue
		}

		if *startFrom != "" && !startFromFileFound {
			if !strings.Contains(de.Name(), *startFrom) {
				if *verbose {
					fmt.Printf("Skipping file: %s (--start flag set)\n", de.Name())
				}
				continue
			}
			startFromFileFound = true
		}

		if *verbose {
			fmt.Printf("Reading file: %s\n", de.Name())
		}
		f, err := os.Open(filepath.Join(*gzipDir, de.Name()))
		if err != nil {
			log.Fatal(err)
		}

		r, err := gzip.NewReader(f)
		if err != nil {
			log.Fatal(err)
		}

		data, err := io.ReadAll(r)
		if err != nil {
			log.Fatal(err)
		}

		var nyt domain.NYTimesMonthlyArticles

		err = json.Unmarshal(data, &nyt)
		if err != nil {
			log.Fatal(err)
		}

		for _, a := range nyt.Response.Docs {
			articlesTotal++

			if !strings.HasSuffix(a.PubDate, "+0000") {
				log.Fatalf("found unexpected data format `%s`", a.PubDate)
			}

			sa := &domain.SearchArticle{
				ID:            a.ID,
				Abstract:      a.Abstract,
				Headline:      a.Headline.Main,
				PrintHeadline: a.Headline.PrintHeadline,
				LeadParagraph: a.LeadParagraph,
				IsPublished:   true,
				PubDate:       a.PubDate,
				PubDateS:      a.PubDate,
				NumLikes:      uint(len(a.Keywords)),
				NumComments:   uint(len(a.Keywords) / 2),
			}

			// Add keywords.
			for _, kw := range a.Keywords {
				sa.Keywords = append(sa.Keywords, kw.Value)
			}

			docs = append(docs, sa)
			docIDs = append(docIDs, sa.ID)

			if *detailedStats {
				st.Read(a)
			}

			if len(docs) >= *maxBulk {
				lastHeadline = sa.Headline
				lastPubDate = sa.PubDate

				indexer.BulkIndex(context.Background(), indexName, docIDs, docs)

				docs = docs[:0]
				docIDs = docIDs[:0]
			}

			if articlesTotal%20_000 == 0 {
				if len(lastHeadline) > 50 {
					lastHeadline = lastHeadline[:46] + " .."
				}

				fmt.Printf("@ article %d  --  %s [%s]\n", articlesTotal, lastHeadline, lastPubDate)
			}
		}

		if len(docs) > 0 {
			sa := docs[len(docs)-1].(*domain.SearchArticle)

			lastHeadline = sa.Headline
			lastPubDate = sa.PubDate

			indexer.BulkIndex(context.Background(), indexName, docIDs, docs)

			docs = docs[:0]
			docIDs = docIDs[:0]
		}

		r.Close()
		f.Close()

		indexer.PrintBulkIndexingRate()

		if *maxDocs > 0 && articlesTotal > *maxDocs {
			fmt.Printf("Exceeded max after indexing %d articles (max: %d). Exiting!\n", articlesTotal, *maxDocs)
			break
		}
	}

	fmt.Printf("Done after %s. Indexed %d articles total.\n", time.Since(timer), articlesTotal)

	if *detailedStats {
		st.Print()
	}
}
