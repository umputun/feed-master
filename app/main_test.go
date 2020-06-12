package main

import (
	"io/ioutil"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	data := []byte(`
feeds:
 first:
  title: "blah 1"
  sources:
   - name: nnn1
     url: http://aa.com/u1
   - name: nnn2
     url: http://aa.com/u2

 second:
  title: "blah 2"
  description: "some 2"
  sources:
   - name: mmm1
     url: https://bbb.com/u1

update: 600
`)

	assert.Nil(t, ioutil.WriteFile("/tmp/fm.yml", data, 0777), "failed write yml") // nolint

	r, err := loadConfig("/tmp/fm.yml")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(r.Feeds), "2 sets")
	assert.Equal(t, 2, len(r.Feeds["first"].Sources), "2 feeds in first")
	assert.Equal(t, 1, len(r.Feeds["second"].Sources), "1 feed in second")
	assert.Equal(t, "https://bbb.com/u1", r.Feeds["second"].Sources[0].URL)
}

func TestLoadConfigNotFoundFile(t *testing.T) {
	r, err := loadConfig("/tmp/29e28b3c-e1a4-4269-a10b-3e9a89a08d45.txt")

	assert.Nil(t, r)
	assert.EqualError(t, err, "open /tmp/29e28b3c-e1a4-4269-a10b-3e9a89a08d45.txt: no such file or directory")
}

func TestLoadConfigInvalidYaml(t *testing.T) {
	assert.Nil(t, ioutil.WriteFile("/tmp/fm.txt", []byte(`Not Yaml`), 0777), "failed write yml") // nolint

	r, err := loadConfig("/tmp/fm.txt")

	assert.Nil(t, r)
	assert.EqualError(t, err, "yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `Not Yaml` into proc.Conf")
}


func TestSingleFeedConf(t *testing.T) {
	cases := []struct{
		feedURL, channel string
		updateInterval time.Duration
	}{
		{"example.com/feed", "Feed", 10},
		{"example.com/my/feed", "My feed", 20},
	}

	for i, tc := range cases {
		i := i
		tc := tc
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			conf := singleFeedConf(tc.feedURL, tc.channel, tc.updateInterval)

			assert.Len(t, conf.Feeds, 1)
			assert.Equal(t, conf.System.UpdateInterval, tc.updateInterval)

			feed := conf.Feeds["auto"]
			assert.Equal(t, feed.TelegramChannel, tc.channel)
			assert.Len(t, feed.Sources, 1)
			assert.Equal(t, feed.Sources[0].Name, "auto")
			assert.Equal(t, feed.Sources[0].URL, tc.feedURL)
		})
	}
}

func TestMakeTwitter(t *testing.T) {
	opts := options{
		TwitterConsumerKey: "a",
		TwitterConsumerSecret: "b",
		TwitterAccessToken: "c",
		TwitterAccessSecret: "d",
	}

	client := makeTwitter(opts)

	assert.Equal(t, client.ConsumerKey, "a")
	assert.Equal(t, client.ConsumerSecret, "b")
	assert.Equal(t, client.AccessToken, "c")
	assert.Equal(t, client.AccessSecret, "d")
}

func TestSetupLog(t *testing.T) { //nolint
	setupLog(true)
	setupLog(false)
}