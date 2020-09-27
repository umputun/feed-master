package proc

import (
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

func TestFilter(t *testing.T) {
	feedItem := feed.Item{Title: "Foo Bar (Part 1)"}
	filter := Filter{Title: "(Part \\d+)"}
	result, _ := filter.skip(feedItem)
	assert.True(t, result)
}

func TestBlankFilter(t *testing.T) {
	feedItem := feed.Item{Title: "Foo Bar (Part 1)"}
	filter := Filter{}
	result, _ := filter.skip(feedItem)
	assert.False(t, result)
}

func TestErrorFilter(t *testing.T) {
	feedItem := feed.Item{Title: "Foo Bar (Part 1)"}
	filter := Filter{"("}
	result, _ := filter.skip(feedItem)
	assert.False(t, result)
}
