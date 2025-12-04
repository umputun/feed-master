package proc

import (
	"errors"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tb "gopkg.in/tucnak/telebot.v2"

	"github.com/umputun/feed-master/app/duration"
	"github.com/umputun/feed-master/app/feed"
	"github.com/umputun/feed-master/app/proc/mocks"
)

func TestNewTelegramClientIfTokenEmpty(t *testing.T) {
	client, err := NewTelegramClient("", "", 0, &duration.Service{}, &TelegramSenderImpl{})
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
			client, err := NewTelegramClient("", "", tt.timeout, &duration.Service{}, &TelegramSenderImpl{})
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, client.Timeout)
		})
	}
}

func TestSendIfBotIsNil(t *testing.T) {
	client, err := NewTelegramClient("", "", 0, &duration.Service{}, &TelegramSenderImpl{})
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
	testData := []struct {
		channel  string
		expected string
	}{
		{channel: "channel", expected: "@channel"},
		{channel: "@channel", expected: "@channel"},
		{channel: "107401628", expected: "107401628"}, // numeric ChanID should be preserved
		{channel: "-1001484738202", expected: "-1001484738202"},
	}
	for i, entry := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got := recipient{chatID: entry.channel} // nolint
			assert.Equal(t, entry.expected, got.Recipient())
		})
	}
}

func TestTelegramClient_sendAudio(t *testing.T) {
	ts := mockTelegramServer(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("req: %+v", r)
		require.Equal(t, "GET", r.Method)
		fh, err := os.Open("testdata/audio.mp3")
		require.NoError(t, err)
		defer fh.Close() //nolint
		_, err = io.Copy(w, fh)
		assert.NoError(t, err)
	})
	defer ts.Close()

	dur := &mocks.DurationServiceMock{
		FileFunc: func(string) int {
			return 12345
		},
	}

	snd := &mocks.TelegramSenderMock{
		SendFunc: func(tb.Audio, *tb.Bot, tb.Recipient, *tb.SendOptions) (*tb.Message, error) {
			return nil, nil
		},
	}

	client := TelegramClient{DurationService: dur, TelegramSender: snd}
	_, err := client.sendAudio("chan1", feed.Item{Duration: "5678", Enclosure: feed.Enclosure{URL: ts.URL}})
	require.NoError(t, err)

	assert.Equal(t, 1, len(snd.SendCalls()))
	assert.Equal(t, 0, len(dur.FileCalls()), "duration service is not used because item has duration")
	assert.Equal(t, 5678, snd.SendCalls()[0].Audio.Duration)

	_, err = client.sendAudio("chan2", feed.Item{Enclosure: feed.Enclosure{URL: ts.URL}})
	require.NoError(t, err)
	assert.Equal(t, 2, len(snd.SendCalls()))
	assert.Equal(t, 1, len(dur.FileCalls()), "duration service used because item has no duration")
	assert.Equal(t, 12345, snd.SendCalls()[1].Audio.Duration)
}

func TestSendIfSendAudioFailed(t *testing.T) {
	ts := mockTelegramServer(nil)
	defer ts.Close()

	dur := &mocks.DurationServiceMock{
		FileFunc: func(string) int {
			return 12345
		},
	}

	snd := &mocks.TelegramSenderMock{
		SendFunc: func(tb.Audio, *tb.Bot, tb.Recipient, *tb.SendOptions) (*tb.Message, error) {
			return nil, errors.New("error while sending audio")
		},
	}

	tc, err := NewTelegramClient("test-token", ts.URL, 900*time.Millisecond, dur, snd)

	require.NoError(t, err)
	assert.NotNil(t, tc)

	err = tc.Send("@channel", feed.Item{Enclosure: feed.Enclosure{URL: ts.URL + "/download/some.mp3"}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can't send to telegram for")

	require.Equal(t, 1, len(snd.SendCalls()))
}

func TestSend(t *testing.T) {
	ts := mockTelegramServer(nil)
	defer ts.Close()

	dur := &mocks.DurationServiceMock{
		FileFunc: func(string) int {
			return 12345
		},
	}

	snd := &mocks.TelegramSenderMock{
		SendFunc: func(tb.Audio, *tb.Bot, tb.Recipient, *tb.SendOptions) (*tb.Message, error) {
			return &tb.Message{Text: "Some test message"}, nil
		},
	}

	tc, err := NewTelegramClient("test-token", ts.URL, 900*time.Millisecond, dur, snd)

	require.NoError(t, err)
	assert.NotNil(t, tc)

	err = tc.Send("@channel", feed.Item{Enclosure: feed.Enclosure{URL: ts.URL + "/download/some.mp3"}})
	assert.NoError(t, err)

	require.Equal(t, 1, len(snd.SendCalls()))
	assert.Equal(t, 12345, snd.SendCalls()[0].Audio.Duration)
	assert.Equal(t, "audio/mpeg", snd.SendCalls()[0].Audio.MIME)
	assert.Equal(t, "some.mp3", snd.SendCalls()[0].Audio.FileName)
	assert.Equal(t, "test-token", snd.SendCalls()[0].Bot.Token)
	assert.Equal(t, ts.URL, snd.SendCalls()[0].Bot.URL)
	assert.Equal(t, "@channel", snd.SendCalls()[0].Recipient.Recipient())
}

func TestSendTextIfAudioLarge(t *testing.T) {
	ts := mockTelegramServer(nil)
	defer ts.Close()

	dur := &mocks.DurationServiceMock{
		FileFunc: func(string) int {
			return 12345
		},
	}

	snd := &mocks.TelegramSenderMock{
		SendFunc: func(tb.Audio, *tb.Bot, tb.Recipient, *tb.SendOptions) (*tb.Message, error) {
			return nil, errors.New("Request Entity Too Large")
		},
	}

	tc, err := NewTelegramClient("test-token", ts.URL, 900*time.Millisecond, dur, snd)

	require.NoError(t, err)
	assert.NotNil(t, tc)

	err = tc.Send("@channel", feed.Item{Enclosure: feed.Enclosure{URL: ts.URL + "/download/some.mp3"}})
	assert.NoError(t, err)

	require.Equal(t, 1, len(snd.SendCalls()))
	assert.Equal(t, 12345, snd.SendCalls()[0].Audio.Duration)
	assert.Equal(t, "audio/mpeg", snd.SendCalls()[0].Audio.MIME)
	assert.Equal(t, "some.mp3", snd.SendCalls()[0].Audio.FileName)
	assert.Equal(t, "test-token", snd.SendCalls()[0].Bot.Token)
	assert.Equal(t, ts.URL, snd.SendCalls()[0].Bot.URL)
	assert.Equal(t, "@channel", snd.SendCalls()[0].Recipient.Recipient())
}

func TestTelegramSenderImpl_Send(t *testing.T) {
	ts := mockTelegramServer(nil)
	defer ts.Close()

	senderImpl := TelegramSenderImpl{}

	fName := "testdata/audio.mp3"
	fh, err := os.Open(fName)
	require.NoError(t, err)
	defer fh.Close() //nolint

	f := tb.File{FileLocal: fName}

	bot, err := tb.NewBot(tb.Settings{URL: ts.URL})
	require.NoError(t, err)

	_, err = senderImpl.Send(tb.Audio{File: f, FileName: fName}, bot, &tb.Chat{}, &tb.SendOptions{})
	assert.NoError(t, err)
}

const getMeResp = `{"ok": true,
				"result": {
					"first_name": "comments_test",
					"id": 707381019,
					"is_bot": true,
					"username": "feedMaster_test_bot"
				}}`

func mockTelegramServer(h http.HandlerFunc) *httptest.Server {
	if h != nil {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.String(), "getMe") {
				_, _ = w.Write([]byte(getMeResp))
				return
			}
			h(w, r)
		}))
	}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /bottest-token/getMe", func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(getMeResp))
	})

	mux.HandleFunc("GET /download/some.mp3", func(w http.ResponseWriter, _ *http.Request) {
		// taken from https://github.com/mathiasbynens/small/blob/master/mp3.mp3
		smallMP3File := []byte{54, 53, 53, 48, 55, 54, 51, 52, 48, 48, 51, 49, 56, 52, 51, 50, 48, 55, 54, 49, 54, 55, 49, 55, 49, 55, 55, 49, 53, 49, 49, 56, 51, 51, 49, 52, 51, 56, 50, 49, 50, 56, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48}

		_, _ = w.Write(smallMP3File)
	})

	mux.HandleFunc("POST /bottest-token/sendMessage", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok": true, "result": {"text": "Some test message"}}`))
	})

	mux.HandleFunc("POST /bot/sendAudio", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok": true, "result": {"id": 1}}`))
	})

	mux.HandleFunc("POST /bot/getMe", func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(getMeResp))
	})

	return httptest.NewServer(mux)
}
