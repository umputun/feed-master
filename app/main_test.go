package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/feed-master/app/config"
)

func TestMakeTwitter(t *testing.T) {
	conf := config.Conf{}
	conf.System.Notifications.Twitter.ConsumerKey = "a"
	conf.System.Notifications.Twitter.ConsumerSecret = "b"
	conf.System.Notifications.Twitter.AccessToken = "c"
	conf.System.Notifications.Twitter.AccessSecret = "d"

	client := makeTwitter(conf)

	assert.Equal(t, client.ConsumerKey, "a")
	assert.Equal(t, client.ConsumerSecret, "b")
	assert.Equal(t, client.AccessToken, "c")
	assert.Equal(t, client.AccessSecret, "d")
}
