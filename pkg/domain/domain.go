package domain

import (
	"context"
)

type Indexer interface {
	CreateIndex(ctx context.Context, mappingsJSONFile, indexName string)
	BulkIndex(ctx context.Context, indexName string, docIDs []string, docs []interface{})
	PrintBulkIndexingRate()
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
	LeadParagraph string       `json:"lead_paragraph"`
	Multimedia    []Multimedia `json:"multimedia"`
	PubDate       string       `json:"pub_date"`
}

type SearchArticle struct {
	ID            string       `json:"id"`
	Abstract      string       `json:"abstract"`
	Headline      string       `json:"headline"`
	PrintHeadline string       `json:"print_headline"`
	LeadParagraph string       `json:"lead_paragraph"`
	Keywords      []string     `json:"keywords"`
	IsPublished   bool         `json:"is_published"`
	PubDate       string       `json:"pub_date"`
	NumLikes      uint         `json:"num_likes"`
	NumComments   uint         `json:"num_comments"`
	Multimedia    []Multimedia `json:"multimedia"`
}

type Multimedia struct {
	URL     string `json:"url"`
	Width   int64  `json:"width"`
	Height  int64  `json:"height"`
	SubType string `json:"subType"`
}
