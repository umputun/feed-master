package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeTwitter(t *testing.T) {
	opts := options{
		TwitterConsumerKey:    "a",
		TwitterConsumerSecret: "b",
		TwitterAccessToken:    "c",
		TwitterAccessSecret:   "d",
	}

	client := makeTwitter(opts)

	assert.Equal(t, client.ConsumerKey, "a")
	assert.Equal(t, client.ConsumerSecret, "b")
	assert.Equal(t, client.AccessToken, "c")
	assert.Equal(t, client.AccessSecret, "d")
}
