package proc

import (
	"html/template"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/umputun/feed-master/app/duration"
	tb "gopkg.in/tucnak/telebot.v2"

	"github.com/umputun/feed-master/app/feed"
)

func TestNewTelegramClientIfTokenEmpty(t *testing.T) {
	client, err := NewTelegramClient("", "", 0, &duration.Service{})
	assert.NoError(t, err)
	assert.Nil(t, client.Bot)
}

func TestNewTelegramClientCheckTimeout(t *testing.T) {
	tbl := []struct {
		timeout, expected time.Duration
	}{
		{0, time.Second * 60},
		{300, 300},
		{100500, 100500},
	}

	for i, tt := range tbl {
		i := i
		tt := tt
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			client, err := NewTelegramClient("", "", tt.timeout, &duration.Service{})
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, client.Timeout)
		})
	}
}

func TestSendIfBotIsNil(t *testing.T) {
	client, err := NewTelegramClient("", "", 0, &duration.Service{})
	require.NoError(t, err)
	err = client.Send("@channel", feed.Item{})
	assert.NoError(t, err)
}

func TestSendIfChannelIDEmpty(t *testing.T) {
	client := TelegramClient{
		Bot: &tb.Bot{},
	}

	err := client.Send("", feed.Item{})
	assert.NoError(t, err)
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

	htmlExpected := `
Особое канадское искусство. 
Результаты нашего странного эксперимента.
Теперь можно и в <a href="https://t.me/uwp_podcast">телеграмме</a>
Саботаж на местах.
Их нравы: кумовство и коррупция.
Вопросы и ответы

<a href="https://podcast.umputun.com/media/ump_podcast437.mp3">аудио</a>`

	client := TelegramClient{}
	got := client.tagLinkOnlySupport(html)
	assert.Equal(t, htmlExpected, got, "support only html tag a")
}

func TestFormattedMessage(t *testing.T) {
	client := TelegramClient{}
	cases := []struct {
		item         feed.Item
		expectedHTML string
	}{
		{item: feed.Item{}, expectedHTML: ""},
		{item: feed.Item{Description: "plain text", Title: "test title"}, expectedHTML: "test title\n\nplain text"},
		{
			item: feed.Item{
				Description: `<![CDATA[<p><img src="https://podcast.umputun.com/images/uwp/uwp463.jpg" alt=""></p>
<ul>
<li>Дела рабочие.</li>
<li>Цепная реакция в деле "умного дома".</li>
<li>Снегопад помешал собачьему дню.</li>
<li>Судорожные поиски аккумулятора и капризность хонды.</li>
<li>Вопросы и ответы</li>
</ul>
<p><a href="https://podcast.umputun.com/media/ump_podcast463.mp3">аудио</a></p>
<audio src="https://podcast.umputun.com/media/ump_podcast463.mp3" preload="none"></audio>]]>`},
			expectedHTML: "Дела рабочие.\nЦепная реакция в деле \"умного дома\".\nСнегопад помешал собачьему дню.\nСудорожные поиски аккумулятора и капризность хонды.\nВопросы и ответы\n\n<a href=\"https://podcast.umputun.com/media/ump_podcast463.mp3\">аудио</a>",
		},
		{
			item: feed.Item{
				Title: "Код доступа : Юлия Латынина",
				Link:  "https://echo.msk.ru/programs/code/2868346-echo/",
				Description: `&lt;img align=&quot;left&quot; width=&quot;50&quot; height=&quot;50&quot; alt=&quot;&quot; title=&quot;&quot; src=&quot;https://echo.msk.ru/files/avatar_s/783858.jpg&quot; /&gt;
 &lt;p&gt;Ведущие:

   &lt;a href=&quot;https://echo.msk.ru/contributors/324/&quot;&gt;Юлия Латынина&lt;/a&gt;
 &lt;/p&gt;
&lt;p&gt;Есть резиновая кукла под названием Явлинский, которую надувает Кириенко. И есть системная кампания Кремля. Вот заказчик — Путин, организатор — Кириенко, а исполнителей, имя им — легион. Это компания по уничтожению сентябрьских выборов и «Умного голосования»...&lt;/p&gt;`,
				Enclosure: feed.Enclosure{
					URL: "https://echo.msk.ru/programs/code/2868346-echo/",
				},
			},
			expectedHTML: "<a href=\"https://echo.msk.ru/programs/code/2868346-echo/\">Код доступа : Юлия Латынина</a>\n\nВедущие:\n\n   <a href=\"https://echo.msk.ru/contributors/324/\">Юлия Латынина</a>\n \nЕсть резиновая кукла под названием Явлинский, которую надувает Кириенко. И есть системная кампания Кремля. Вот заказчик — Путин, организатор — Кириенко, а исполнителей, имя им — легион. Это компания по уничтожению сентябрьских выборов и «Умного голосования»...",
		},
	}

	for i, tc := range cases {
		i := i
		tc := tc
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			htmlMessage := client.getMessageHTML(tc.item, htmlMessageParams{})
			assert.Equal(t, tc.expectedHTML, htmlMessage)
		})
	}
}

func TestTruncatedMessage(t *testing.T) {
	client := TelegramClient{}
	htmlMessage := client.getMessageHTML(
		feed.Item{
			Title:       "title",
			Enclosure:   feed.Enclosure{URL: "https://example.com/some.mp3"},
			Description: template.HTML(strings.Repeat("test", 1000)), //nolint:gosec // test case, no security issues
		},
		htmlMessageParams{WithMp3Link: true, TrimCaption: true})
	assert.True(t, strings.HasPrefix(htmlMessage, "title\n\n"))
	assert.True(t, strings.HasSuffix(htmlMessage, "\n\nhttps://example.com/some.mp3"))
	assert.LessOrEqual(t, len(htmlMessage), 1024)
}

func TestGetMessageHTML(t *testing.T) {
	item := feed.Item{
		Title:       "\tPodcast\n\t",
		Description: "<p>News <a href='/test'>Podcast Link</a></p>\n",
		Enclosure: feed.Enclosure{
			URL: "https://example.com",
		},
		Link: "https://example.com/xyz",
	}

	expected := "<a href=\"https://example.com/xyz\">Podcast</a>\n\nNews <a href=\"/test\">Podcast Link</a>\n\nhttps://example.com"

	client := TelegramClient{}
	msg := client.getMessageHTML(item, htmlMessageParams{WithMp3Link: true})
	assert.Equal(t, expected, msg)
}

func TestRecipientChannelIDNotStartWithAt(t *testing.T) {
	cases := []string{"channel", "@channel"}
	expected := "@channel"

	for i, channelID := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got := recipient{chatID: channelID} // nolint
			assert.Equal(t, expected, got.Recipient())
		})
	}
}
