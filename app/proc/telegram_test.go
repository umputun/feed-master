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

func TestTagLinkOnlySupport(t *testing.T) {
	html := `
<li>Особое канадское искусство. </li>
<li>Результаты нашего странного эксперимента.</li>
<li>Теперь можно и в <a href="https://t.me/uwp_podcast">телеграмме</a></li>
<li>Саботаж на местах.</li>
<li>Их нравы: кумовство и коррупция.</li>
<li>Вопросы и ответы</li>
</ul>
<p><a href="https://podcast.umputun.com/media/ump_podcast437.mp3"><em>аудио</em></a></p>`

	html_expected := `
Особое канадское искусство. 
Результаты нашего странного эксперимента.
Теперь можно и в <a href="https://t.me/uwp_podcast">телеграмме</a>
Саботаж на местах.
Их нравы: кумовство и коррупция.
Вопросы и ответы

<a href="https://podcast.umputun.com/media/ump_podcast437.mp3">аудио</a>`

	client := TelegramClient{}

	got := client.tagLinkOnlySupport(html)

	assert.Equal(t, got, html_expected, "support only html tag a")
}

func TestGetMessageHTML(t *testing.T) {
	item := feed.Item{
		Title:       "\tPodcast\n\t",
		Description: "<p>News <a href='#'>Podcast Link</a></p>\n",
		Enclosure: feed.Enclosure{
			URL: "https://example.com",
		},
	}

	expected := "Podcast\n\nNews <a href=\"#\">Podcast Link</a>\n\nhttps://example.com"

	client := TelegramClient{}
	got := client.getMessageHTML(item)

	assert.Equal(t, got, expected)
}
