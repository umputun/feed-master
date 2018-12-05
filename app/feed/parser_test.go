package feed

import (
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFeedParse(t *testing.T) {
	r, err := Parse("http://feeds.rucast.net/radio-t")
	assert.Nil(t, err)
	log.Printf("%+v", r.ItemList[0])
}

func TestNormalizeDate(t *testing.T) {

	tbl := []struct {
		inp string
		err error
		out string
	}{
		{"05 Mar 14 22:08 +0400", nil, "05 Mar 14 22:08 +0400"},
	}

	rss := Rss2{}
	for _, tb := range tbl {
		ts, err := rss.normalizeDate(tb.inp)
		assert.Equal(t, tb.err, err)
		assert.Equal(t, tb.out, ts.Format(time.RFC822Z))
	}
}
