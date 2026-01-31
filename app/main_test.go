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

	assert.Equal(t, "a", client.ConsumerKey)
	assert.Equal(t, "b", client.ConsumerSecret)
	assert.Equal(t, "c", client.AccessToken)
	assert.Equal(t, "d", client.AccessSecret)
}
