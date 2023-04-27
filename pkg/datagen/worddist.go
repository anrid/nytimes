package datagen

import (
	"fmt"
	"log"
	"sort"
	"strings"
)

// A WordDistribution is a an array of words associated with numeric ranges
// that represent how often the word occurs in a given body of text.
// It can be used to randomly select words from a text according to the natural
// distribution of those words in a given text.
// For example, the word `and` will occur far more often than the word `house`
// in a typical English text. This should be reflected in randomly generated
// sentences used to test / benchmark databases or search enginees.
type WordDistribution struct {
	d         []*Offset
	Length    int
	MaxOffset int64
}

func NewWordDistribution(wordCounts map[string]int64) *WordDistribution {
	wd := new(WordDistribution)

	var words []*Count
	for word, count := range wordCounts {
		if word == "" || count == 0 {
			log.Panicf("illegal word '%s' or count = 0  --  word counts: %v", word, wordCounts)
		}
		words = append(words, &Count{word, count})
	}

	// Sort by count descending.
	sort.Slice(words, func(i, j int) bool {
		return words[i].Count > words[j].Count
	})

	var offset int64
	for _, w := range words {
		wd.d = append(wd.d, &Offset{w.Word, offset})
		offset += w.Count
	}

	wd.MaxOffset = offset

	if len(wd.d) == 0 {
		log.Panicf("empty word distribution  --  word counts: %v", wordCounts)
	}

	wd.Length = len(wd.d)

	return wd
}

func (wd *WordDistribution) RandomSentence(numWords int) string {
	var sentence []string

	for i := 0; i < numWords; i++ {
		sentence = append(sentence, wd.RandomWord())
	}

	return strings.Join(sentence, " ")
}

func (wd *WordDistribution) RandomWord() string {
	if len(wd.d) == 1 {
		return wd.d[0].Word
	}

	return wd.GetWord(getRandomInt64(wd.MaxOffset))
}

func (wd *WordDistribution) GetWord(offset int64) string {
	if len(wd.d) == 1 {
		return wd.d[0].Word
	}

	// Perform a binary search on our word distribution to find
	// the interval closest to our offset.
	i := sort.Search(len(wd.d), func(i int) bool {
		return wd.d[i].Offset >= offset
	})

	if i <= len(wd.d) {
		// If we've ended up past the end of the array, step back 1.
		if i == len(wd.d) {
			i--
		} else if wd.d[i].Offset != offset {
			// Use the previous word in the word distribution unless we found
			// an exact match. If we found an exact match use it as the start
			// of our range / offset.
			if i-1 >= 0 {
				i--
			}
		}

		// Perform a sanity check:
		// wd.d[i].Offset (selected word) <= offset < wd.d[i+1].Offset (next word)
		start := wd.d[i].Offset
		end := wd.MaxOffset
		if i+1 < len(wd.d) {
			end = wd.d[i+1].Offset
		}

		// This should never happen!
		if offset == end {
			log.Panicf("random offset %d = end %d word: %s range [%d-%d)", offset, end, wd.d[i].Word, start, end)
		}

		return wd.d[i].Word
	}

	// We're past the end of the array. This should never happen!
	for _, w := range wd.d {
		fmt.Printf("word: %s (offset: %d)\n", w.Word, w.Offset)
	}
	fmt.Printf("max offset: %d\n", wd.MaxOffset)

	log.Panicf("could not find a word in word distribution for offset %d in range: [0-%d)", offset, wd.MaxOffset)
	return ""
}
