package stats

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/anrid/nytimes/pkg/domain"
)

func New() *Stats {
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

func (s *Stats) Read(a *domain.NYTimesArticle) {
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
