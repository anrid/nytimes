package datagen

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"regexp"
	"sort"
	"strings"
)

var (
	whitespace      = regexp.MustCompile(`[^a-z0-9\-_'’]+`)
	whitespacePunct = regexp.MustCompile(`[^a-z0-9\-_'’\.]+`)
	fullstop        = regexp.MustCompile(`([a-z0-9]+)(\.)(\s+|$)`)
)

type WordGraph struct {
	g     map[string]map[string]int64
	d     map[string]*WordDistribution
	ready bool
	words []string
}

func NewWordGraph() *WordGraph {
	return &WordGraph{
		g: make(map[string]map[string]int64),
		d: make(map[string]*WordDistribution),
	}
}

func (g *WordGraph) AddText(text string) {
	lower := strings.ToLower(text)
	fs := fullstop.ReplaceAllString(lower, "$1 $2$3")
	words := whitespacePunct.Split(fs, -1)

	// fmt.Printf("input 1: %v\n", text)
	// fmt.Printf("input 2: %v\n\n", words)

	var clean []string
	for _, w := range words {
		if len(w) > 0 {
			clean = append(clean, w)
		}
	}

	for i, w := range clean {
		next := "."
		if i+1 < len(clean) {
			next = clean[i+1]
		}
		if _, found := g.g[w]; !found {
			g.g[w] = make(map[string]int64)
		}
		g.g[w][next]++
	}
}

func (g *WordGraph) RandomSentence(numWords int) string {
	if !g.ready {
		// Create word distributions for all words in the
		// graph.
		for word, nextWords := range g.g {
			// Edge case for punct / fullstop.
			// We don't want the word distribution for words after fullstop
			// to contain another fullstop, so we remove the fullstop
			// from the distribution.
			if word == "." && len(nextWords) > 1 {
				m := make(map[string]int64)
				for k, v := range nextWords {
					if k != "." {
						m[k] = v
					}
				}
				nextWords = m
			}

			g.d[word] = NewWordDistribution(nextWords)
		}

		// Create an array of all words, for quick random
		// lookups.
		for word := range g.g {
			g.words = append(g.words, word)
		}

		g.ready = true

		// for _, w := range g.d["."].d {
		// 	fmt.Printf("word: %s  offset: %d\n", w.Word, w.Offset)
		// }
	}

	randomWordIndex := getRandomInt64(int64(len(g.words)))
	currentWord := g.words[randomWordIndex]
	sentence := []string{currentWord}
	var lastWasPunct bool

	for i := 0; i < numWords; i++ {
		if d, found := g.d[currentWord]; found {
			if lastWasPunct && d.Length > 10 {
				// Try not to have a punct following another punct.
				for j := 0; j < 10; j++ {
					currentWord = d.RandomWord()
					if currentWord != "." {
						break
					}
				}
				lastWasPunct = false
			} else {
				currentWord = d.RandomWord()
			}
		} else {
			log.Panicf("could not find a word distribution for word '%s'", currentWord)
		}

		sentence = append(sentence, currentWord)

		if currentWord == "." {
			lastWasPunct = true
		}
	}

	return strings.Join(sentence, " ")
}

func Dump(o interface{}) {
	b, _ := json.MarshalIndent(o, "", "  ")
	fmt.Printf("dump:\n\n%s\n", string(b))
}

func getRandomInt64(max int64) int64 {
	r, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		log.Panic(err)
	}
	return r.Int64()
}

type Dictionary struct {
	w map[string]int64
	d *WordDistribution
}

func NewDictionary() *Dictionary {
	return &Dictionary{
		w: make(map[string]int64),
	}
}

func (d *Dictionary) AddText(text string) {
	words := whitespace.Split(strings.ToLower(text), -1)

	for _, w := range words {
		if len(w) > 0 {
			d.w[w]++
		}
	}
}

func (d *Dictionary) RandomSentence(numWords int) string {
	var sentence []string

	for i := 0; i < numWords; i++ {
		sentence = append(sentence, d.RandomWord())
	}

	return strings.Join(sentence, " ")
}

func (d *Dictionary) RandomWord() string {
	if d.d == nil {
		d.d = NewWordDistribution(d.w)
	}

	return d.d.RandomWord()
}

func (d *Dictionary) Stats() {
	fmt.Printf("Dictionary contains %d words\n\n", len(d.w))

	var top []Count

	for word, count := range d.w {
		top = append(top, Count{word, count})
	}

	sort.Slice(top, func(i, j int) bool {
		return top[i].Count > top[j].Count
	})

	for i := 0; i < len(top) && i < 10; i++ {
		fmt.Printf("Top %d. %s (%d)\n", i+1, top[i].Word, top[i].Count)
	}

	fmt.Println("")
}

type Count struct {
	Word  string
	Count int64
}

type Offset struct {
	Word   string
	Offset int64
}
