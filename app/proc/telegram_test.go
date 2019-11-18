package proc

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
