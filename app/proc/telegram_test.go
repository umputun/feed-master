package proc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	tb "gopkg.in/tucnak/telebot.v2"

	"github.com/umputun/feed-master/app/feed"
)

func TestNewTelegramClientIfTokenEmpty(t *testing.T) {
	client, err := NewTelegramClient("")

	assert.Nil(t, err)
	assert.Nil(t, client.Bot)
}

func TestSendIfBotIsNil(t *testing.T) {
	client, err := NewTelegramClient("")

	got := client.Send("@channel", feed.Item{})

	assert.Nil(t, err)
	assert.Nil(t, got)
}

func TestSendIfChannelIDEmpty(t *testing.T) {
	client := TelegramClient{
		Bot: &tb.Bot{},
	}

	got := client.Send("", feed.Item{})

	assert.Nil(t, got)
}
