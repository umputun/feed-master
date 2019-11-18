package proc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	tb "gopkg.in/tucnak/telebot.v2"

	"github.com/umputun/feed-master/app/feed"
)

func TestNewTelegramClientIfTokenEmpty(t *testing.T) {
	token := ""
	client, err := NewTelegramClient(token)

	assert.Nil(t, err)
	assert.Nil(t, client)
}

func TestSendIfBotIsNil(t *testing.T) {
	client := TelegramClient{
		Bot: nil,
	}

	item := feed.Item{}
	got := client.Send("@channel", item)

	assert.Nil(t, got)
}

func TestSendIfChannelIDEmpty(t *testing.T) {
	client := TelegramClient{
		Bot: &tb.Bot{},
	}

	got := client.Send("", feed.Item{})

	assert.Nil(t, got)
}
