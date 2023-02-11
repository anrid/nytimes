package main

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/anrid/nytimes/pkg/search"
	"github.com/goccy/go-json"
	"github.com/spf13/pflag"
)

var (
	indexName = "nytimes-articles"

	gzipDir       = pflag.String("dir", "data/", "Directory with GZIP files containing New York Times articles (filenames must end in `.json.gz`)")
	startFrom     = pflag.String("start-from", "", "File to (re)start from")
	detailedStats = pflag.Bool("detailed-stats", false, "Calculate detailed stats (WARN: requires lots of memory, use only when indexing few docs)")
	maxDocs       = pflag.Int("max-docs", 5_000, "Max number of docs to index in bulk.")
	createIndex   = pflag.Bool("create-index", false, "Drop and recreate a new index")
	useIndexer    = pflag.String("indexer", "es", "Indexer to use, available: ['es']")
)

type Indexer interface {
	CreateIndex(ctx context.Context, mappingsDir, indexName string)
	BulkIndex(ctx context.Context, indexName string, docIDs []string, docs []interface{})
	PrintBulkIndexingRate()
}

func main() {
	pflag.Parse()

	if *gzipDir == "" {
		pflag.Usage()
		log.Fatalf("missing --dir arg")
	}

	var indexer Indexer
	switch strings.ToLower(*useIndexer) {
	case "es":
		indexer = search.New(nil)
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
		indexer.CreateIndex(ctx, "nytimes", indexName)
	}

	var articlesTotal int
	var docs []interface{}
	var docIDs []string
	var startFromFileFound bool
	var lastHeadline string
	var lastPubDate string

	stats := new(Stats)

	for _, de := range des {
		if !strings.HasSuffix(de.Name(), ".json.gz") {
			fmt.Printf("Skipping file: %s\n", de.Name())
			continue
		} else if de.IsDir() {
			fmt.Printf("Skipping dir: %s\n", de.Name())
			continue
		}

		if *startFrom != "" && !startFromFileFound {
			if !strings.Contains(de.Name(), *startFrom) {
				fmt.Printf("Skipping file: %s (--start flag set)\n", de.Name())
				continue
			}
			startFromFileFound = true
		}

		fmt.Printf("Reading file: %s\n", de.Name())
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

		var nyt NYTimesMonthlyArticles

		err = json.Unmarshal(data, &nyt)
		if err != nil {
			log.Fatal(err)
		}

		for _, a := range nyt.Response.Docs {
			articlesTotal++

			if !strings.HasSuffix(a.PubDate, "+0000") {
				log.Fatalf("found unexpected data format `%s`", a.PubDate)
			}

			sa := &SearchArticle{
				ID:            a.ID,
				Abstract:      a.Abstract,
				Headline:      a.Headline.Main,
				PrintHeadline: a.Headline.PrintHeadline,
				LeadParagraph: a.LeadParagraph,
				IsPublished:   true,
				PubDate:       a.PubDate,
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
				stats.Read(a)
			}

			if len(docs) >= *maxDocs {
				lastHeadline = sa.Headline
				lastPubDate = sa.PubDate

				indexer.BulkIndex(context.Background(), indexName, docIDs, docs)

				docs = docs[:0]
				docIDs = docIDs[:0]
			}

			if articlesTotal%50_000 == 0 {
				if len(lastHeadline) > 50 {
					lastHeadline = lastHeadline[:46] + " .."
				}

				fmt.Printf("@ article %d  --  %s [%s]\n", articlesTotal, lastHeadline, lastPubDate)
			}
		}

		if len(docs) > 0 {
			sa := docs[len(docs)-1].(*SearchArticle)

			lastHeadline = sa.Headline
			lastPubDate = sa.PubDate

			indexer.BulkIndex(context.Background(), indexName, docIDs, docs)

			docs = docs[:0]
			docIDs = docIDs[:0]
		}

		r.Close()
		f.Close()

		indexer.PrintBulkIndexingRate()
	}

	if *detailedStats {
		stats.Print()
	}
}

func NewStats() *Stats {
	return &Stats{
		timer:     time.Now(),
		tokenizer: *regexp.MustCompile(`[^[[:alnum:]]+`),
		Words:     make(map[string]uint64),
		Keywords:  make(map[string]uint64),
	}
}

type Stats struct {
	timer             time.Time
	tokenizer         regexp.Regexp
	TotalArticleCount uint64
	TotalWordCount    uint64
	HeadlineWordCount uint64
	LeadWordCount     uint64
	KeywordCount      uint64
	Words             map[string]uint64
	Keywords          map[string]uint64
}

func (s *Stats) Read(a *NYTimesArticle) {
	hm := s.tokenizer.Split(strings.ToLower(a.Headline.Main), -1)
	hp := s.tokenizer.Split(strings.ToLower(a.Headline.PrintHeadline), -1)
	lp := s.tokenizer.Split(strings.ToLower(a.LeadParagraph), -1)

	s.TotalArticleCount++

	for _, w := range hm {
		if w != "" {
			s.Words[w]++
			s.HeadlineWordCount++
			s.TotalWordCount++
		}
	}
	for _, w := range hp {
		if w != "" {
			s.Words[w]++
			s.HeadlineWordCount++
			s.TotalWordCount++
		}
	}
	for _, w := range lp {
		if w != "" {
			s.Words[w]++
			s.LeadWordCount++
			s.TotalWordCount++
		}
	}
	for _, kw := range a.Keywords {
		s.Keywords[kw.Value]++
	}
}

func (s *Stats) Print() {
	type entry struct {
		W string
		C uint64
	}
	var sorted []entry

	secs := time.Since(s.timer).Seconds()

	fmt.Println("\nStats:")
	fmt.Printf("Article count           : %d\n", s.TotalArticleCount)
	fmt.Printf("Word count              : %d\n", s.TotalWordCount)
	fmt.Printf("Unique word count       : %d\n", len(s.Words))
	fmt.Printf("Headline word count     : %d\n", s.HeadlineWordCount)
	fmt.Printf("Lead word count         : %d\n", s.LeadWordCount)
	fmt.Printf("Unique keyword count    : %d\n", len(s.Keywords))
	fmt.Printf("Indexing rate           : %.02f / sec\n", float64(s.TotalArticleCount)/secs)

	for w, c := range s.Words {
		sorted = append(sorted, entry{W: w, C: c})
	}
	fmt.Println("")

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].C > sorted[j].C
	})

	fmt.Println("Common words:")
	for i, e := range sorted {
		fmt.Printf("%s:%d ", e.W, e.C)
		if i > 100 {
			break
		}
	}
	fmt.Println("")
}

type NYTimesMonthlyArticles struct {
	Response struct {
		Docs []*NYTimesArticle `json:"docs"`
	} `json:"response"`
}

type NYTimesArticle struct {
	ID       string `json:"_id"`
	Abstract string `json:"abstract"`
	Byline   struct {
		Organization string `json:"organization"`
		Original     string `json:"original"`
		Person       []struct {
			Firstname string `json:"firstname"`
			Lastname  string `json:"lastname"`
		} `json:"person"`
	} `json:"byline"`
	Headline struct {
		Main          string `json:"main"`
		PrintHeadline string `json:"print_headline"`
	} `json:"headline"`
	Keywords []struct {
		Name  string `json:"name"`
		Rank  int64  `json:"rank"`
		Value string `json:"value"`
	} `json:"keywords"`
	LeadParagraph string `json:"lead_paragraph"`
	Multimedia    []struct {
		URL     string `json:"url"`
		Width   int64  `json:"width"`
		Height  int64  `json:"height"`
		SubType string `json:"subType"`
	} `json:"multimedia"`
	PubDate    string    `json:"pub_date"`
	PubDateUTC time.Time `json:"pub_date_utc"`
}

type SearchArticle struct {
	ID            string   `json:"id"`
	Abstract      string   `json:"abstract"`
	Headline      string   `json:"headline"`
	PrintHeadline string   `json:"print_headline"`
	LeadParagraph string   `json:"lead_paragraph"`
	Keywords      []string `json:"keywords"`
	IsPublished   bool     `json:"is_published"`
	PubDate       string   `json:"pub_date"`
	NumLikes      uint     `json:"num_likes"`
	NumComments   uint     `json:"num_comments"`
}
