package datagen

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"regexp"
	"sort"
	"strings"
)

var (
	whitespace = regexp.MustCompile(`[^a-z0-9\-_'’]+`)
)

type Dictionary struct {
	w               map[string]int64
	distrib         []*Pair
	distribMaxRange int64
}

func NewDict() *Dictionary {
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

func (d *Dictionary) createWordDistribution() {
	if len(d.w) == 0 {
		log.Fatal("dictionary is empty")
	}

	var ts []Pair

	for word, count := range d.w {
		ts = append(ts, Pair{word, count})
	}

	sort.Slice(ts, func(i, j int) bool {
		return ts[i].Value > ts[j].Value
	})

	var offset int64
	for _, t := range ts {
		d.distrib = append(d.distrib, &Pair{t.Word, offset})
		offset += t.Value
	}

	d.distribMaxRange = offset

	for i, p := range d.distrib {
		fmt.Printf("%03d: %s (%d)\n", i+1, p.Word, p.Value)
		if i+1 == 100 {
			break
		}
	}

	fmt.Println("")
}

func (d *Dictionary) RandomSentence(numWords int) string {
	var sentence []string

	for i := 0; i < numWords; i++ {
		sentence = append(sentence, d.RandomWord())
	}

	return strings.Join(sentence, " ")
}

func (d *Dictionary) RandomWord() string {
	if len(d.distrib) == 0 {
		d.createWordDistribution()
	}

	r, err := rand.Int(rand.Reader, big.NewInt(d.distribMaxRange))
	if err != nil {
		log.Panic(err)
	}

	randomStartRange := r.Int64()

	i := sort.Search(len(d.distrib), func(i int) bool {
		return d.distrib[i].Value >= randomStartRange
	})

	if i < len(d.distrib) {
		// Use the previous word in the word distribution.
		if d.distrib[i].Value != randomStartRange {
			if i-1 >= 0 {
				i--
			}
		}

		startRange := d.distrib[i].Value
		endRange := d.distribMaxRange
		if i+1 < len(d.distrib) {
			endRange = d.distrib[i+1].Value
		}

		// fmt.Printf("w: %-30s  -- range: [%d - %d) random value: %d\n", d.distrib[i].Word, startRange, endRange, randomStartRange)

		if randomStartRange == endRange {
			log.Panicf("random start range %d = end range %d word: %s range: [%d-%d)", randomStartRange, endRange, d.distrib[i].Word, startRange, endRange)
		}

		return d.distrib[i].Word
	}

	log.Fatalf("could not find a word within the word distribution for random value %d in range: [0-%d)", randomStartRange, d.distribMaxRange)
	return ""
}

func (d *Dictionary) Stats() {
	fmt.Printf("Dictionary contains %d words\n\n", len(d.w))

	var top []Pair

	for word, count := range d.w {
		top = append(top, Pair{word, count})
	}

	sort.Slice(top, func(i, j int) bool {
		return top[i].Value > top[j].Value
	})

	for i := 0; i < len(top) && i < 10; i++ {
		fmt.Printf("Top %d. %s (%d)\n", i+1, top[i].Word, top[i].Value)
	}

	fmt.Println("")
}

type Pair struct {
	Word  string
	Value int64
}
