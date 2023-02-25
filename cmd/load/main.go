package main

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/anrid/nytimes/pkg/domain"
	"github.com/anrid/nytimes/pkg/loader"
	"github.com/anrid/nytimes/pkg/search/es"
	"github.com/spf13/pflag"
)

var (
	indexName = "nytimes-articles"

	gzipDir     = pflag.String("dir", "data/", "Directory with GZIP files containing New York Times articles (filenames must end in `.json.gz`)")
	startFrom   = pflag.String("start-from", "", "File to (re)start from")
	maxBulk     = pflag.Int("max-bulk", 5_000, "Max number of docs to index in bulk")
	maxDocs     = pflag.Int("max-docs", 0, "Max number of docs to index")
	createIndex = pflag.Bool("create-index", false, "Drop and recreate a new index")
	verbose     = pflag.BoolP("verbose", "v", false, "Verbose output")
	useIndexer  = pflag.String("indexer", "es", "Indexer to use, available: ['es']")
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if *createIndex {
		indexer.CreateIndex(ctx, "./assets/mappings/nytimes/index-mappings.json", indexName)
	}

	ld := loader.New(indexName, *maxBulk, indexer)

	loader.ReadDirWithArticles(loader.ReadDirWithArticlesParams{
		Path:        *gzipDir,
		Suffix:      ".json.gz",
		Verbose:     *verbose,
		StartFrom:   *startFrom,
		Max:         *maxDocs,
		EachArticle: ld.IndexArticle,
	})

}
