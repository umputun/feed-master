package proc

import (
	"bytes"
	"fmt"
	"hash/maphash"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf16"

	log "github.com/go-pkgz/lgr"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pkg/errors"
	"github.com/tcolgate/mp3"
	tg "github.com/xelaj/mtproto/telegram"
	"golang.org/x/net/html"

	"github.com/umputun/feed-master/app/feed"
)

// TelegramClient is a Telegram API client
type TelegramClient struct {
	Token          string // obtained from https://core.telegram.org/bots#3-how-do-i-create-a-bot
	Server         string // taken from https://my.telegram.org/apps
	PublicKeysFile string // with content taken from https://my.telegram.org/apps
	SessionFile    string // sessions will be stored, doesn't have to exist in advance
	AppID          int    // taken from https://my.telegram.org/apps
	AppHash        string // taken from https://my.telegram.org/apps
	Version        string // app version
	OnlyMessage    bool   // instead of sending media files, just send the text message

	Lock *sync.Mutex

	client telegramAPIClient // used only in tests
}

// telegramAPIClient is a subset of API client functions used by this module,
// using which allows to mock it in tests
type telegramAPIClient interface {
	MessagesSendMessage(*tg.MessagesSendMessageParams) (tg.Updates, error)
	MessagesSendMedia(*tg.MessagesSendMediaParams) (tg.Updates, error)
	ContactsResolveUsername(string) (*tg.ContactsResolvedPeer, error)
	UploadSaveBigFilePart(int64, int32, int32, []byte) (bool, error)
	AuthImportBotAuthorization(int32, int32, string, string) (tg.AuthAuthorization, error)
	Stop() error
}

// Send message, skip if telegram token or channelID are empty
func (client TelegramClient) Send(channelID string, item feed.Item) error {
	if client.Token == "" || channelID == "" {
		return nil
	}

	if client.Lock == nil {
		return errors.New("lock is not defined")
	}

	// run only one at any given moment to prevent bombarding the API
	client.Lock.Lock()
	defer client.Lock.Unlock()

	var err error
	var newClient telegramAPIClient
	newClient = client.client
	if newClient == nil {
		newClient, err = tg.NewClient(tg.ClientConfig{
			PublicKeysFile:  client.PublicKeysFile,
			SessionFile:     client.SessionFile,
			ServerHost:      client.Server,
			AppID:           client.AppID,
			AppHash:         client.AppHash,
			InitWarnChannel: true,
			AppVersion:      client.Version,
		})
		if err != nil {
			return errors.Wrapf(err, "error creating telegram client")
		}
		newClient.(*tg.Client).RecoverFunc = func(r interface{}) {
			log.Printf("[ERROR] recovered panic from Telegram API: %v", r)
		}
		// close the newClient after the sending
		finished := make(chan bool)
		defer close(finished)
		go func(err chan error, finished chan bool) {
			for {
				select {
				case tgErr := <-err:
					log.Printf("[DEBUG] warning from Telegram API: %v", tgErr)
				case <-finished:
					return
				}
			}
		}(newClient.(*tg.Client).Warnings, finished)
	}
	defer newClient.Stop()

	_, err = newClient.AuthImportBotAuthorization(0, int32(client.AppID), client.AppHash, client.Token)
	if err != nil {
		return errors.Wrapf(err, "error authorizing with telegram bot")
	}

	chanRef, err := client.getChannelReference(newClient, channelID)
	if err != nil {
		return errors.Wrapf(err, "error retrieving channel metadata")
	}

	htmlMessage := client.getMessageHTML(item)
	plainMessage := client.getPlainMessage(htmlMessage)

	entities := client.getMessageFormatting(htmlMessage, plainMessage)

	if client.OnlyMessage {
		return client.sendTextMessage(newClient, item, chanRef, entities, plainMessage)
	}

	return client.sendMessageWithFile(newClient, item, chanRef, entities, plainMessage)
}

func (client TelegramClient) sendTextMessage(c telegramAPIClient, item feed.Item, chanRef tg.InputPeer, entities []tg.MessageEntity, msg string) error {
	log.Printf("[DEBUG] sending the text message for %s", item.Enclosure.URL)
	_, err := c.MessagesSendMessage(&tg.MessagesSendMessageParams{
		NoWebpage: true,
		Peer:      chanRef,
		Message:   msg,
		Entities:  entities,
		RandomID:  client.getInt64Hash(msg),
	})
	if err != nil {
		return errors.Wrapf(err, "error sending the telegram message")
	}
	return nil
}

func (client TelegramClient) sendMessageWithFile(c telegramAPIClient, item feed.Item, chanRef tg.InputPeer, entities []tg.MessageEntity, msg string) error {
	contentLength, err := client.getContentLength(item.Enclosure.URL)
	if err != nil {
		return errors.Wrapf(err, "can't get length for %s", item.Enclosure.URL)
	}

	log.Printf("[DEBUG] start uploading audio %s (%dMb)", item.Enclosure.URL, contentLength/1024/1024)
	httpBody, err := item.DownloadAudio(time.Minute)
	if err != nil {
		return errors.Wrapf(err, "error retrieving audio")
	}
	defer httpBody.Close()

	var httpBodyCopy bytes.Buffer
	tee := io.TeeReader(httpBody, &httpBodyCopy)

	fileID := client.getInt64Hash(item.Enclosure.URL)
	fileChunks, err := client.uploadFileToTelegram(c, tee, fileID, contentLength)
	if err != nil {
		return errors.Wrapf(err, "error uploading the file")
	}

	mimeType := item.Enclosure.Type
	if mimeType == "" {
		mimeType = "audio/mpeg"
	}

	trackDuration := client.duration(&httpBodyCopy)

	_, err = c.MessagesSendMedia(&tg.MessagesSendMediaParams{
		Peer: chanRef,
		Media: &tg.InputMediaUploadedDocument{
			MimeType: mimeType,
			Attributes: []tg.DocumentAttribute{
				&tg.DocumentAttributeAudio{Title: item.Title, Duration: trackDuration},
				&tg.DocumentAttributeFilename{FileName: item.GetFilename()},
			},
			File: &tg.InputFileBig{
				ID:    fileID,
				Parts: fileChunks,
				Name:  item.GetFilename(),
			},
		},
		RandomID: fileID,
		Message:  msg,
		Entities: entities,
	})
	if err != nil {
		return errors.Wrapf(err, "error uploading message to channel")
	}

	return nil
}

// getChannelReference returns telegram channel metadata reference which
// is enough to send messages to that channel using the telegram API
func (client TelegramClient) getChannelReference(c telegramAPIClient, channelID string) (tg.InputPeer, error) {
	channel, err := c.ContactsResolveUsername(channelID)
	if err != nil {
		return nil, err
	}
	return &tg.InputPeerChannel{
		ChannelID:  channel.Chats[0].(*tg.Channel).ID,
		AccessHash: channel.Chats[0].(*tg.Channel).AccessHash,
	}, nil
}

// uploadFileToTelegram uploads file to telegram API returns number of file parts it uploaded
func (client TelegramClient) uploadFileToTelegram(c telegramAPIClient, r io.Reader, fileID int64, fileLength int) (int32, error) {
	var fileParts []int32
	// 512kb is magic number from https://core.telegram.org/api/files, you can't set bigger chunks
	chunkSize := 1024 * 512
	buf := make([]byte, chunkSize)
	approximateChunks := int32(fileLength/chunkSize + 1)
	var err error
	var copyBytes int
	for err != io.EOF && err != io.ErrUnexpectedEOF {
		copyBytes, err = io.ReadFull(r, buf)
		if err != io.EOF && err != io.ErrUnexpectedEOF && err != nil {
			return 0, errors.Wrapf(err, "error reading the file chunk for upload")
		}
		// don't send zero-filled buffer part in case that's the last chunk of file
		if err == io.ErrUnexpectedEOF {
			buf = buf[:copyBytes]
		}

		filePartID := int32(len(fileParts))
		_, uploadErr := c.UploadSaveBigFilePart(fileID, filePartID, approximateChunks, buf)
		if uploadErr != nil {
			return 0, errors.Wrapf(uploadErr, "error uploading the file using telegram API")
		}
		fileParts = append(fileParts, filePartID)
	}
	return int32(len(fileParts)), nil
}

// getInt64Hash generates int64 hash from the provided string, returns 0 in case of error
func (client TelegramClient) getInt64Hash(s string) int64 {
	hash := maphash.Hash{}
	_, _ = hash.Write([]byte(s))
	return int64(hash.Sum64())
}

// getContentLength uses HEAD request and called as a fallback in case of item.Enclosure.Length not populated
func (client TelegramClient) getContentLength(url string) (int, error) {
	resp, err := http.Head(url) // nolint:gosec // URL considered safe
	if err != nil {
		return 0, errors.Wrapf(err, "can't HEAD %s", url)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, errors.Errorf("non-200 status, %d", resp.StatusCode)
	}

	return int(resp.ContentLength), err
}

// https://core.telegram.org/api/entities
// currently only links are supported, but it's possible to parse all listed entities
func (client TelegramClient) tagLinkOnlySupport(htmlText string) string {
	p := bluemonday.NewPolicy()
	p.AllowAttrs("href").OnElements("a")
	return html.UnescapeString(p.Sanitize(htmlText))
}

// getPlainMessage strips provided HTML to the bare text
func (client TelegramClient) getPlainMessage(htmlText string) string {
	p := bluemonday.NewPolicy()
	return html.UnescapeString(p.Sanitize(htmlText))
}

// getMessageHTML generates HTML message from provided feed.Item
func (client TelegramClient) getMessageHTML(item feed.Item) string {
	description := string(item.Description)

	description = strings.TrimPrefix(description, "<![CDATA[")
	description = strings.TrimSuffix(description, "]]>")

	// apparently bluemonday doesn't remove escaped HTML tags
	description = client.tagLinkOnlySupport(html.UnescapeString(description))
	description = strings.TrimSpace(description)

	messageHTML := description

	title := strings.TrimSpace(item.Title)
	if title != "" {
		switch {
		case item.Link == "":
			messageHTML = fmt.Sprintf("%s\n\n", title) + messageHTML
		case item.Link != "":
			messageHTML = fmt.Sprintf("<a href=\"%s\">%s</a>\n\n", item.Link, title) + messageHTML
		}
	}

	return messageHTML
}

// duration scans MP3 file from provided io.Reader and returns it's duration in seconds, ignoring possible errors
func (client TelegramClient) duration(r io.Reader) int32 {
	d := mp3.NewDecoder(r)
	var f mp3.Frame
	var skipped int
	var duration float64
	var err error

	for err == nil {
		if err = d.Decode(&f, &skipped); err != nil && err != io.EOF {
			return 0
		}
		duration += f.Duration().Seconds()
	}
	return int32(duration)
}

// getMessageFormatting gets links from HTML text and maps them to same text in plain format using MessageEntity
func (client TelegramClient) getMessageFormatting(htmlMessage, plainMessage string) []tg.MessageEntity {
	doc, err := html.Parse(bytes.NewBufferString(htmlMessage))
	if err != nil {
		log.Printf("[WARN] can't parse HTML message: %v", err)
		return nil
	}

	b, err := client.getBody(doc)
	if err != nil {
		log.Printf("[WARN] problem finding HTML message body: %v", err)
		return nil
	}

	// this parser doesn't work recursively, only for the first level,
	// which is OK as we strip everything but <a> and they can't be nested
	n := b.FirstChild
	var entities []tg.MessageEntity
	var offsetIndexUTF8 int // this variable is necessary to track the link position
	for n != nil {
		if n.Data != "a" {
			offsetIndexUTF8 += len(n.Data)
		}
		if n.Data == "a" {
			url := ""
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					url = attr.Val
				}
			}
			if n.FirstChild == nil || n.FirstChild != n.LastChild {
				log.Printf("[WARN] problem parsing a href=%s, can't retrieve link text", url)
				n = n.NextSibling
				continue
			}
			aText := n.FirstChild.Data
			offsetIndexUTF16 := len(utf16.Encode([]rune(plainMessage[:offsetIndexUTF8])))
			lengthUTF16 := len(utf16.Encode([]rune(aText)))
			entities = append(entities, &tg.MessageEntityTextURL{
				Offset: int32(offsetIndexUTF16),
				Length: int32(lengthUTF16),
				URL:    url,
			})
			// for <a> link, rendered text is located in the first child data
			offsetIndexUTF8 += len(aText)
		}
		n = n.NextSibling
	}

	return entities
}

// getBody returns provided document <body> node if found
func (client TelegramClient) getBody(doc *html.Node) (*html.Node, error) {
	var body *html.Node
	var crawler func(*html.Node)
	crawler = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "body" {
			body = node
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			crawler(child)
		}
	}
	crawler(doc)
	if body != nil {
		return body, nil
	}
	return nil, errors.New("missing <body> in the node tree")
}
