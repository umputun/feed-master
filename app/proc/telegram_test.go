package proc

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	tg "github.com/xelaj/mtproto/telegram"

	"github.com/umputun/feed-master/app/feed"
)

func TestSendIfBotIsNil(t *testing.T) {
	client := TelegramClient{}
	err := client.Send("@channel", feed.Item{})
	assert.NoError(t, err)
}

func TestSendIfChannelIDEmpty(t *testing.T) {
	client := TelegramClient{}

	err := client.Send("", feed.Item{})
	assert.NoError(t, err)
}

func TestSendIfLockIsNil(t *testing.T) {
	client := TelegramClient{Token: "good_token"}
	err := client.Send("@channel", feed.Item{})
	assert.EqualError(t, err, "lock is not defined")
}

func TestSendIfContentLengthZero(t *testing.T) {
	mockClient := &MockTelegramAPIClient{t: t}
	client := TelegramClient{
		AppID:   12345,
		AppHash: "test_api_hash",
		Token:   "test:api_token",
		Lock:    &sync.Mutex{},
		client:  mockClient,
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Length", "0")
	}))
	defer ts.Close()

	err := client.Send("100500", feed.Item{Enclosure: feed.Enclosure{URL: ts.URL}})

	assert.EqualError(t, err, fmt.Sprintf("can't get length for %s: non-200 status, 500", ts.URL))
}

func TestSendClientCreationErrors(t *testing.T) {
	client := TelegramClient{
		Token:          "good_token",
		Lock:           &sync.Mutex{},
		PublicKeysFile: "bad_file",
		Server:         "bad_server",
	}

	err := client.Send("100500", feed.Item{})
	assert.EqualError(t, err, "error creating telegram client: file 'bad_file' not found")

	client.PublicKeysFile = "../../_example/etc/tg_public_keys.pem"
	err = client.Send("100500", feed.Item{})
	assert.EqualError(t, err, "error creating telegram client: creating connection: resolving tcp: address bad_server: missing port in address")
}

func TestSend(t *testing.T) {
	mockClient := &MockTelegramAPIClient{
		t:                   t,
		failAuth:            true,
		failUploadFile:      true,
		failSendMedia:       true,
		failSendText:        true,
		failResolveUsername: true,
	}
	client := TelegramClient{
		Token:       "test:api_token",
		AppHash:     "test_api_hash",
		Version:     "",
		AppID:       12345,
		OnlyMessage: false,
		Lock:        &sync.Mutex{},
		client:      mockClient,
	}
	// taken from https://github.com/mathiasbynens/small/blob/master/mp3.mp3
	smallMP3File := []byte{54, 53, 53, 48, 55, 54, 51, 52, 48, 48, 51, 49, 56, 52, 51, 50, 48, 55, 54, 49, 54, 55, 49, 55, 49, 55, 55, 49, 53, 49, 49, 56, 51, 51, 49, 52, 51, 56, 50, 49, 50, 56, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write(smallMP3File)
		assert.NoError(t, err)
		w.Header().Set("Content-Length", strconv.Itoa(len(smallMP3File)))
	}))
	defer ts.Close()

	// auth error
	err := client.Send("100500", feed.Item{Enclosure: feed.Enclosure{URL: ts.URL}})
	assert.EqualError(t, err, "error authorizing with telegram bot: test")

	// channel metadata error
	mockClient.failAuth = false
	err = client.Send("100500", feed.Item{Enclosure: feed.Enclosure{URL: ts.URL}})
	assert.EqualError(t, err, "error retrieving channel metadata: test")

	// error uploading the file
	mockClient.failResolveUsername = false
	err = client.Send("100500", feed.Item{Enclosure: feed.Enclosure{URL: ts.URL}})
	assert.EqualError(t, err, "error uploading the file: error uploading the file using telegram API: test")

	// error sending the message with media
	mockClient.failUploadFile = false
	err = client.Send("100500", feed.Item{Enclosure: feed.Enclosure{URL: ts.URL}})
	assert.EqualError(t, err, "error uploading message to channel: test")

	// successful send of media message
	mockClient.failSendMedia = false
	err = client.Send("100500", feed.Item{Enclosure: feed.Enclosure{URL: ts.URL}, Link: ts.URL, Title: "test_title", Description: "test description"})
	assert.NoError(t, err)

	// error sending the message without media
	client.OnlyMessage = true
	err = client.Send("100500", feed.Item{Enclosure: feed.Enclosure{URL: ts.URL}})
	assert.EqualError(t, err, "error sending the telegram message: test")

	// successful send of text
	mockClient.failSendText = false
	err = client.Send("100500", feed.Item{Enclosure: feed.Enclosure{URL: ts.URL}, Title: "test_title", Description: "test description"})
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
		item             feed.Item
		expectedPlain    string
		expectedEntities []tg.MessageEntity
	}{
		{item: feed.Item{}, expectedPlain: "", expectedEntities: nil},
		{item: feed.Item{Description: "plain text"}, expectedPlain: "plain text", expectedEntities: nil},
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
			expectedPlain: "Дела рабочие.\nЦепная реакция в деле \"умного дома\".\nСнегопад помешал собачьему дню.\nСудорожные поиски аккумулятора и капризность хонды.\nВопросы и ответы\n\nаудио",
			expectedEntities: []tg.MessageEntity{&tg.MessageEntityTextURL{
				Offset: 153,
				Length: 5,
				URL:    "https://podcast.umputun.com/media/ump_podcast463.mp3",
			}},
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
			expectedPlain: "Код доступа : Юлия Латынина\n\nВедущие:\n    \n    Юлия Латынина\n  \nЕсть резиновая кукла под названием Явлинский, которую надувает Кириенко. И есть системная кампания Кремля. Вот заказчик — Путин, организатор — Кириенко, а исполнителей, имя им — легион. Это компания по уничтожению сентябрьских выборов и «Умного голосования»...",
			expectedEntities: []tg.MessageEntity{
				&tg.MessageEntityTextURL{
					Offset: 0,
					Length: 27,
					URL:    "https://echo.msk.ru/programs/code/2868346-echo/",
				},
				&tg.MessageEntityTextURL{
					Offset: 47,
					Length: 13,
					URL:    "https://echo.msk.ru/contributors/324/",
				},
			},
		},
	}

	for i, tc := range cases {
		i := i
		tc := tc
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			htmlMessage := client.getMessageHTML(tc.item)
			plainMessage := client.getPlainMessage(htmlMessage)
			assert.Equal(t, tc.expectedPlain, plainMessage)

			entities := client.getMessageFormatting(htmlMessage, plainMessage)
			assert.Equal(t, tc.expectedEntities, entities)
		})
	}
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

	expected := "<a href=\"https://example.com/xyz\">Podcast</a>\n\nNews <a href=\"/test\">Podcast Link</a>"

	client := TelegramClient{}
	msg := client.getMessageHTML(item)
	assert.Equal(t, expected, msg)
}

func TestGetContentLengthNotFound(t *testing.T) {
	client := TelegramClient{}
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
					_, err := fmt.Fprint(w, "abcd")
					assert.NoError(t, err)
				}
			}))

			defer ts.Close()

			length, err := client.getContentLength(ts.URL)

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

	client := TelegramClient{}

	length, err := client.getContentLength(ts.URL)

	assert.Equal(t, length, 0)
	assert.EqualError(t, err, fmt.Sprintf("can't HEAD %s: Head %q: EOF", ts.URL, ts.URL))
}

func TestUploadFileBadReader(t *testing.T) {
	client := TelegramClient{}
	r := iotest.ErrReader(errors.New("test error"))
	duration, err := client.uploadFileToTelegram(&MockTelegramAPIClient{}, r, 0, 0)
	assert.EqualError(t, err, "error reading the file chunk for upload: test error")
	assert.Zero(t, duration)
}

func TestDurationDetectBadReader(t *testing.T) {
	client := TelegramClient{}
	r := iotest.ErrReader(errors.New("test error"))
	duration := client.duration(r)
	assert.Zero(t, duration)
}

type MockTelegramAPIClient struct {
	t *testing.T

	failAuth, failUploadFile, failSendMedia, failSendText, failResolveUsername bool
}

func (m MockTelegramAPIClient) AuthImportBotAuthorization(flags, apiID int32, apiHash, botAuthToken string) (tg.AuthAuthorization, error) {
	if m.failAuth {
		return nil, errors.New("test")
	}
	assert.Equal(m.t, int32(0), flags)
	assert.Equal(m.t, int32(12345), apiID)
	assert.Equal(m.t, "test_api_hash", apiHash)
	assert.Equal(m.t, "test:api_token", botAuthToken)
	return nil, nil
}

//nolint:dupl // duplicating tests is OK
func (m MockTelegramAPIClient) MessagesSendMessage(params *tg.MessagesSendMessageParams) (tg.Updates, error) {
	if m.failSendText {
		return nil, errors.New("test")
	}
	assert.True(m.t, params.NoWebpage)
	assert.NotZero(m.t, params.RandomID)
	assert.Equal(m.t, &tg.InputPeerChannel{ChannelID: 321, AccessHash: 123321}, params.Peer)
	assert.Equal(m.t, "test_title\n\ntest description", params.Message)
	assert.Zero(m.t, params.Entities)
	return nil, nil
}

//nolint:dupl // duplicating tests is OK
func (m MockTelegramAPIClient) MessagesSendMedia(params *tg.MessagesSendMediaParams) (tg.Updates, error) {
	if m.failSendMedia {
		return nil, errors.New("test")
	}
	assert.NotEmpty(m.t, params.Media)
	assert.NotZero(m.t, params.RandomID)
	assert.Equal(m.t, &tg.InputPeerChannel{ChannelID: 321, AccessHash: 123321}, params.Peer)
	assert.Equal(m.t, "test_title\n\ntest description", params.Message)
	if len(params.Entities) > 0 {
		assert.Equal(m.t, 1, len(params.Entities))
		assert.Equal(m.t, 10, int(params.Entities[0].(*tg.MessageEntityTextURL).Length))
		assert.Contains(m.t, params.Entities[0].(*tg.MessageEntityTextURL).URL, "http://127.0.0.1:")
	}
	return nil, nil
}

func (m MockTelegramAPIClient) ContactsResolveUsername(usernames string) (*tg.ContactsResolvedPeer, error) {
	if m.failResolveUsername {
		return nil, errors.New("test")
	}
	assert.Equal(m.t, "100500", usernames)
	return &tg.ContactsResolvedPeer{Chats: []tg.Chat{&tg.Channel{
		ID:         321,
		AccessHash: 123321,
	}}}, nil
}

func (m MockTelegramAPIClient) UploadSaveBigFilePart(fileID int64, filePart, fileTotalParts int32, bytes []byte) (bool, error) {
	if m.failUploadFile {
		return false, errors.New("test")
	}
	// taken from https://github.com/mathiasbynens/small/blob/master/mp3.mp3
	smallMP3File := []byte{54, 53, 53, 48, 55, 54, 51, 52, 48, 48, 51, 49, 56, 52, 51, 50, 48, 55, 54, 49, 54, 55, 49, 55, 49, 55, 55, 49, 53, 49, 49, 56, 51, 51, 49, 52, 51, 56, 50, 49, 50, 56, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48}

	assert.NotEmpty(m.t, fileID)
	assert.Equal(m.t, int32(0), filePart)
	assert.Equal(m.t, int32(1), fileTotalParts)
	assert.Equal(m.t, smallMP3File, bytes)
	return false, nil
}

func (m MockTelegramAPIClient) Stop() error {
	return nil
}
