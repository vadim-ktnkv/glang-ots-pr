package hw03frequencyanalysis

import (
	"slices"
	"strings"
)

type item struct {
	word string

	count int
}

func wordsCompare(a *item, b *item) int {
	if a.count == b.count {
		return strings.Compare(a.word, b.word)
	}

	return b.count - a.count
}

var words []*item

var wordsLookup map[string]*item

func Top10(txt string) []string {
	words = []*item{}

	wordsLookup = map[string]*item{}

	for _, w := range strings.Fields(txt) {
		word := strings.ToLower(strings.Trim(w, "!\":;',.`"))

		if word == "-" {
			continue
		}

		itemPtr := wordsLookup[word]

		if itemPtr == nil {
			wd := item{word, 1}

			words = append(words, &wd)

			wordsLookup[word] = &wd
		} else {
			itemPtr.count++
		}
	}

	slices.SortStableFunc(words, wordsCompare)

	top10words := make([]string, 0, 10)

	for i, wd := range words {
		top10words = append(top10words, wd.word)

		if i >= 9 {
			break
		}
	}

	return top10words
}
