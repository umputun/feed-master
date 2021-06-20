package proc

import (
	"regexp/syntax"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/feed-master/app/feed"
)

func TestSetDefault(t *testing.T) {
	p := Processor{
		Conf: &Conf{},
	}

	p.setDefaults()

	expectedConf := Conf{
		System: struct {
			UpdateInterval time.Duration `yaml:"update"`
			MaxItems       int           `yaml:"max_per_feed"`
			MaxTotal       int           `yaml:"max_total"`
			MaxKeepInDB    int           `yaml:"max_keep"`
			Concurrent     int           `yaml:"concurrent"`
			BaseURL        string        `yaml:"base_url"`
		}{UpdateInterval: time.Minute * 5, MaxItems: 5, MaxTotal: 100, MaxKeepInDB: 5000, Concurrent: 8, BaseURL: ""},
	}

	assert.EqualValues(t, expectedConf.System, p.Conf.System)
}

func TestFilterAllCases(t *testing.T) {
	tbl := []struct {
		filter Filter
		inp    feed.Item
		err    error
		out    bool
	}{
		{
			Filter{Title: "(Part \\d+)"},
			feed.Item{Title: "Title (Part 1)"},
			nil,
			true,
		},
		{
			Filter{},
			feed.Item{Title: "Title"},
			nil,
			false,
		},
		{
			Filter{Title: "("},
			feed.Item{Title: "Title"},
			&syntax.Error{Code: "missing closing )", Expr: "("},
			false,
		},
	}

	for i, tb := range tbl {
		tb := tb
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			result, err := tb.filter.skip(tb.inp)
			assert.Equal(t, tb.out, result)
			assert.Equal(t, tb.err, err)
		})
	}
}

func TestReverse(t *testing.T) {
	p := Processor{
		Conf: &Conf{},
	}
	items := []feed.Item{{Title: "1"}, {Title: "2"}}
	p.reverse(items)
	assert.Equal(t, items, []feed.Item{{Title: "2"}, {Title: "1"}})
}
