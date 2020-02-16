package proc

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanText(t *testing.T) {
	tbl := []struct {
		inp, out string
		max      int
	}{
		{"test", "test", 10},
		{"test 12345 aaaa", "test ...", 6},
		{"<b>test 12345 aaaa</b>", "test ...", 6},
		{"<b>test12345 aaaa</b>", "test12 ...", 6},
	}

	for i, tt := range tbl {
		i := i
		tt := tt
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			out := CleanText(tt.inp, tt.max)
			assert.Equal(t, tt.out, out)
		})
	}
}
