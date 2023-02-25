// Loader package handles loading NY Times articles into a
// search engine.
package loader

import (
	"context"
	"fmt"

	"github.com/anrid/nytimes/pkg/domain"
)

// Indexer defines methods for creating and loading indexes.
type Indexer interface {
	CreateIndex(ctx context.Context, mappingsJSONFile, indexName string)
	BulkIndex(ctx context.Context, indexName string, docIDs []string, docs []interface{})
	PrintBulkIndexingRate()
}

type Loader struct {
	indexName    string
	maxBulk      int
	i            Indexer
	docs         []interface{}
	docIDs       []string
	lastHeadline string
	lastPubDate  string
}

func New(indexName string, maxBulk int, i Indexer) *Loader {
	return &Loader{indexName: indexName, maxBulk: maxBulk, i: i}
}

func (l *Loader) IndexArticle(articlesTotal int, isLast bool, a *domain.NYTimesArticle) error {
	if isLast && len(l.docs) > 0 {
		l.i.BulkIndex(context.Background(), l.indexName, l.docIDs, l.docs)

		fmt.Printf("Indexed %d articles total\n", articlesTotal)
		return nil
	}

	sa := &domain.SearchArticle{
		ID:            a.ID,
		Abstract:      a.Abstract,
		Headline:      a.Headline.Main,
		PrintHeadline: a.Headline.PrintHeadline,
		LeadParagraph: a.LeadParagraph,
		IsPublished:   true,
		PubDate:       a.PubDate,
		NumLikes:      uint(len(a.Keywords)),
		NumComments:   uint(len(a.Keywords) / 2),
		Multimedia:    a.Multimedia,
	}

	// Add keywords.
	for _, kw := range a.Keywords {
		sa.Keywords = append(sa.Keywords, kw.Value)
	}

	l.docs = append(l.docs, sa)
	l.docIDs = append(l.docIDs, sa.ID)

	if len(l.docs) >= l.maxBulk {
		l.lastHeadline = sa.Headline
		l.lastPubDate = sa.PubDate

		l.i.BulkIndex(context.Background(), l.indexName, l.docIDs, l.docs)

		l.docs = l.docs[:0]
		l.docIDs = l.docIDs[:0]
	}

	if articlesTotal%20_000 == 0 {
		if len(l.lastHeadline) > 50 {
			l.lastHeadline = l.lastHeadline[:46] + " .."
		}

		fmt.Printf("@ article %d  --  %s [%s]\n", articlesTotal, l.lastHeadline, l.lastPubDate)

		l.i.PrintBulkIndexingRate()
	}

	return nil
}
