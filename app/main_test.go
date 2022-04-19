package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/feed-master/app/config"
)

func TestMakeTwitter(t *testing.T) {
	conf := config.Conf{
		Twitter: struct {
			ConsumerKey    string `yaml:"consumer-key"`
			ConsumerSecret string `yaml:"consumer-secret"`
			AccessToken    string `yaml:"access-token"`
			AccessSecret   string `yaml:"access-secret"`
			Template       string `yaml:"template"`
		}{
			ConsumerKey:    "a",
			ConsumerSecret: "b",
			AccessToken:    "c",
			AccessSecret:   "d",
		},
	}

	client := makeTwitter(conf)

	assert.Equal(t, client.ConsumerKey, "a")
	assert.Equal(t, client.ConsumerSecret, "b")
	assert.Equal(t, client.AccessToken, "c")
	assert.Equal(t, client.AccessSecret, "d")
}
