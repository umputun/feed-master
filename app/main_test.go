package main

import (
	"testing"
	"time"

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

func TestYtdlpUpdateStatus(t *testing.T) {
	tbl := []struct {
		name      string
		cmd       string
		interval  time.Duration
		onStartup bool
		want      string
	}{
		{"no command", "", time.Hour, true, "disabled, no command set"},
		{"startup and interval", "pip -U yt-dlp", 24 * time.Hour, true, "on startup and every 24h0m0s"},
		{"startup only", "pip -U yt-dlp", 0, true, "on startup only, no interval set"},
		{"interval only", "pip -U yt-dlp", 24 * time.Hour, false, "every 24h0m0s"},
		{"command never runs", "pip -U yt-dlp", 0, false, "never, command set but force_on_startup is false and no interval given"},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ytdlpUpdateStatus(tt.cmd, tt.interval, tt.onStartup))
		})
	}
}
