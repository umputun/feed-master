package proc

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tb "gopkg.in/tucnak/telebot.v2"

	"github.com/umputun/feed-master/app/feed"
)

func TestNewTelegramClientIfTokenEmpty(t *testing.T) {
	client, err := NewTelegramClient("", 0)
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
			client, err := NewTelegramClient("", tt.timeout)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, client.Timeout)
		})
	}
}

func TestSendIfBotIsNil(t *testing.T) {
	client, err := NewTelegramClient("", 0)
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

func TestSendIfContentLengthZero(t *testing.T) {
	client := TelegramClient{
		Bot: &tb.Bot{},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Length", "0")
	}))
	defer ts.Close()

	err := client.Send("100500", feed.Item{
		Enclosure: feed.Enclosure{
			URL:    ts.URL,
			Length: 0,
		},
	})

	assert.Error(t, err)
	assert.EqualError(t, err, fmt.Sprintf("can't get length for %s: non-200 status, 500", ts.URL))
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

func TestGetMessageHTML(t *testing.T) {
	item := feed.Item{
		Title:       "\tPodcast\n\t",
		Description: "<p>News <a href='#'>Podcast Link</a></p>\n",
		Enclosure: feed.Enclosure{
			URL: "https://example.com",
		},
		Link: "https://example.com/xyz",
	}

	expected := "<a href=\"https://example.com/xyz\">Podcast</a>\n\nNews <a href=\"#\">Podcast Link</a>\n\nhttps://example.com"

	client := TelegramClient{}
	msg := client.getMessageHTML(item, true)
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

func TestGetFilenameByURL(t *testing.T) {
	tbl := []struct {
		url, expected string
	}{
		{"https://example.com/100500/song.mp3", "song.mp3"},
		{"https://example.com//song.mp3", "song.mp3"},
		{"https://example.com/song.mp3", "song.mp3"},
		{"https://example.com/song.mp3/", ""},
		{"https://example.com/", ""},
	}

	for i, tt := range tbl {
		i := i
		tt := tt
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			client := TelegramClient{}
			fname := client.getFilenameByURL(tt.url)
			assert.Equal(t, tt.expected, fname)
		})
	}
}

func TestGetContentLengthNotFound(t *testing.T) {
	cases := []struct {
		statusCode     int
		contentLength  int
		expectedLength int
		expectedError  string
	}{
		{http.StatusInternalServerError, 100500, 0, "non-200 status, 500"},
		{http.StatusMethodNotAllowed, 100500, 0, "non-200 status, 405"},
		{http.StatusOK, 4, 4, ""},
	}

	for i, tc := range cases {
		i := i
		tc := tc
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				w.Header().Set("Content-Length", strconv.Itoa(tc.contentLength))
				if tc.contentLength > 0 {
					fmt.Fprint(w, "abcd")
				}
			}))

			defer ts.Close()

			length, err := getContentLength(ts.URL)

			assert.Equal(t, tc.expectedLength, length)
			if err != nil {
				assert.EqualError(t, err, tc.expectedError)
			}
		})
	}
}

func TestGetContentLengthIfErrorConnect(t *testing.T) {
	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts.CloseClientConnections()
	}))

	defer ts.Close()

	length, err := getContentLength(ts.URL)

	assert.Equal(t, length, 0)
	assert.EqualError(t, err, fmt.Sprintf("can't HEAD %s: Head %s: EOF", ts.URL, ts.URL))
}

func TestDownloadAudioIfRequestError(t *testing.T) {
	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts.CloseClientConnections()
	}))

	defer ts.Close()

	client := TelegramClient{}
	got, err := client.downloadAudio(ts.URL)

	assert.Nil(t, got)
	assert.EqualError(t, err, fmt.Sprintf("Get %s: EOF", ts.URL))
}

func TestDownloadAudio(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Length", fmt.Sprint(4))
		fmt.Fprint(w, "abcd")
	}))
	defer ts.Close()

	client := TelegramClient{}
	got, err := client.downloadAudio(ts.URL)

	assert.NotNil(t, got)
	assert.Nil(t, err)
}
