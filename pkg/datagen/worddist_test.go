package datagen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSomething(t *testing.T) {
	r := require.New(t)

	r.Equal(123, 123, "they should be equal")

	wd := NewWordDistribution(map[string]int64{
		"a": 50,
		"b": 20,
		"c": 10,
		"d": 5,
		"e": 2,
		"f": 1,
	})

	r.Equal("a", wd.GetWord(-1))
	r.Equal("a", wd.GetWord(0))
	r.Equal("a", wd.GetWord(49))
	r.Equal("b", wd.GetWord(50))
	r.Equal("c", wd.GetWord(70))
	r.Equal("d", wd.GetWord(80))
	r.Equal("e", wd.GetWord(85))
	r.Equal("f", wd.GetWord(87))
	r.Equal("f", wd.GetWord(90))
}
