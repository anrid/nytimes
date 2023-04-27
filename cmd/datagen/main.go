package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/anrid/nytimes/pkg/datagen"
	"github.com/anrid/nytimes/pkg/domain"
	"github.com/anrid/nytimes/pkg/loader"
	"github.com/spf13/pflag"
)

var (
	gzipDir   = pflag.String("dir", "data/", "Directory with GZIP files containing New York Times articles (filenames must end in `.json.gz`)")
	startFrom = pflag.String("start-from", "", "File to (re)start from")
	maxDocs   = pflag.Int("max-docs", 0, "Max number of docs to index")
	num       = pflag.Int("num", 10, "Number of sentences to generate")
	maxWords  = pflag.Int("max-words", 10, "Max number of words per generated sentence")
	verbose   = pflag.BoolP("verbose", "v", false, "Verbose output")
)

func main() {
	pflag.Parse()

	if *gzipDir == "" {
		pflag.Usage()
		log.Fatalf("missing --dir arg")
	}

	d := datagen.NewDictionary()
	wg := datagen.NewWordGraph()

	loader.ReadDirWithArticles(loader.ReadDirWithArticlesParams{
		Path:      *gzipDir,
		Suffix:    ".json.gz",
		Verbose:   *verbose,
		StartFrom: *startFrom,
		Max:       *maxDocs,
		EachArticle: func(articlesTotal int, isLast bool, a *domain.NYTimesArticle) error {
			// fmt.Printf("%d. article: %v\n", articlesTotal, a)

			if a != nil {
				d.AddText(a.Abstract + " " + a.Headline.Main + " " + a.LeadParagraph)
				wg.AddText(a.Abstract + " " + a.Headline.Main + " " + a.LeadParagraph)
			}

			if articlesTotal >= *maxDocs {
				return errors.New("nope")
			}

			return nil
		},
	})

	// d.Stats()
	// wg.Dump()

	for i := 0; i < *num; i++ {
		fmt.Printf("%s\n", wg.RandomSentence(*maxWords))
	}
}
