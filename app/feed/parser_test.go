package feed

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFeedParse(t *testing.T) {
	r, err := Parse("http://feeds.rucast.net/radio-t")
	assert.Nil(t, err)
	log.Printf("%+v", r.ItemList[0])
}

func TestFeedParseHttpError(t *testing.T) {
	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts.CloseClientConnections()
	}))

	_, err := Parse(ts.URL)

	assert.NotNil(t, err)
}

func TestNormalizeDate(t *testing.T) {

	tbl := []struct {
		inp string
		err error
		out string
	}{
		{"", fmt.Errorf("can't normalize empty pubDate"), time.Now().Format(time.RFC822Z)},
		{"05 Mar 14 22:08 +0400", nil, "05 Mar 14 22:08 +0400"},           // RFC822Z
		{"05 Mar 14 22:08 MST", nil, "05 Mar 14 22:08 +0000"},             // RFC822
		{"Mon, 02 Jan 2006 15:04:05 -0700", nil, "02 Jan 06 15:04 -0700"}, // RFC1123Z
		{"Mon, 02 Jan 2006 15:04:05 MST", nil, "02 Jan 06 15:04 +0000"},   // RFC1123
		{"2006-01-02 15:04:05 -0700", nil, "02 Jan 06 15:04 -0700"},
		{"100500", fmt.Errorf("can't normalize 100500"), time.Now().Format(time.RFC822Z)},
	}

	rss := Rss2{}
	for _, tb := range tbl {
		ts, err := rss.normalizeDate(tb.inp)
		assert.Equal(t, tb.err, err)
		assert.Equal(t, tb.out, ts.Format(time.RFC822Z))
	}
}

func TestParseAtomInvalidContent(t *testing.T) {
	invalidContent := []byte(`<?xml version="1.0" encoding="UTF-8"?> <rss`)

	_, err := parseAtom(invalidContent)

	assert.Equal(t, err.Error(), "can't parse atom1: XML syntax error on line 1: unexpected EOF")
}

func TestParseAtom(t *testing.T) {
	atom1 := `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">

  <title>Example Feed</title>
  <link href="http://example.org/"/>
  <updated>2003-12-13T18:30:02Z</updated>
  <author>
    <name>John Doe</name>
  </author>
  <id>urn:uuid:60a76c80-d399-11d9-b93C-0003939e0af6</id>

  <entry>
    <title>Atom-Powered Robots Run Amok</title>
    <link href="http://example.org/2003/12/13/atom03"/>
    <id>urn:uuid:1225c695-cfb8-4ebb-aaaa-80da344efa6a</id>
    <updated>2003-12-13T18:30:02Z</updated>
    <summary>Some text.</summary>
  </entry>

  <entry>
    <title>Atom-Powered Robots Run Amok</title>
    <link href="http://example.org/2003/12/13/atom03"/>
    <id>urn:uuid:1225c695-cfb8-4ebb-aaaa-80da344efa6a</id>
    <updated>2003-12-13T18:30:02Z</updated>
    <summary>Some text.</summary>
	<content>Example content</content>
  </entry>

</feed>`

	got, err := parseAtom([]byte(atom1))

	assert.Nil(t, err)
	assert.Equal(t, got.Title, "Example Feed")
	assert.Equal(t, got.Description, "")

	assert.Len(t, got.ItemList, 2)
	assert.Equal(t, got.ItemList[0].Title, "Atom-Powered Robots Run Amok")
	assert.Equal(t, got.ItemList[0].Description, template.HTML("Some text."))

	assert.Equal(t, got.ItemList[1].Description, template.HTML("Example content"))
}
