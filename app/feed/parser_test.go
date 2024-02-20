package feed

import (
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeedParse(t *testing.T) {
	const testFeed = `
<?xml version="1.0" encoding="UTF-8"?>
<rss xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd" xmlns:media="http://search.yahoo.com/mrss/" version="2.0">
  <channel>
    <title>Радио-Т</title>
    <link>https://radio-t.com</link>
    <language>ru</language>
    <copyright>Creative Commons - Attribution, Noncommercial, No Derivative Works 3.0 License.</copyright>
    <itunes:author>Umputun, Bobuk, Gray, Ksenks, Alek.sys</itunes:author>
    <itunes:subtitle>Еженедельные импровизации на хай–тек темы</itunes:subtitle>
    <description>Разговоры на темы хайтек, высоких компьютерных технологий, гаджетов, облаков, программирования и прочего интересного из мира ИТ.</description>
    <itunes:explicit>no</itunes:explicit>
    <itunes:image href="https://radio-t.com/images/cover.jpg" />
    <itunes:keywords>hitech,russian,radiot,tech,news,радио</itunes:keywords>
    <atom10:link xmlns:atom10="http://www.w3.org/2005/Atom" rel="self" type="application/rss+xml" href="http://feeds.rucast.net/radio-t" /><feedburner:info xmlns:feedburner="http://rssnamespace.org/feedburner/ext/1.0" uri="radio-t" /><atom10:link xmlns:atom10="http://www.w3.org/2005/Atom" rel="hub" href="http://pubsubhubbub.appspot.com/" /><media:copyright>Creative Commons - Attribution, Noncommercial, No Derivative Works 3.0 License.</media:copyright><media:thumbnail url="https://radio-t.com/images/cover.jpg" /><media:keywords>hitech,russian,radiot,tech,news,радио</media:keywords><media:category scheme="http://www.itunes.com/dtds/podcast-1.0.dtd">Technology/Tech News</media:category><media:category scheme="http://www.itunes.com/dtds/podcast-1.0.dtd">Technology/Gadgets</media:category><itunes:owner><itunes:email>podcast@radio-t.com</itunes:email><itunes:name>Umputun, Bobuk, Gray, Ksenks, Alek.sys</itunes:name></itunes:owner><itunes:summary>Еженедельные импровизации на хай–тек темы</itunes:summary><itunes:category text="Technology"><itunes:category text="Tech News" /></itunes:category><itunes:category text="Technology"><itunes:category text="Gadgets" /></itunes:category>
    <item>
      <title>Радио-Т 762</title>
      <description><![CDATA[<p><img src="https://radio-t.com/images/radio-t/rt762.jpg" alt=""></p>
<p><em>Темы</em><ul>
<li>Официальный кат №1 от Алексея - <em>00:14:35</em></li>
<li>Официальный кат №2 от Алексея - <em>00:16:10</em></li>
<li><a href="https://www.theguardian.com/commentisfree/2021/jul/05/amazon-worker-fired-app-dystopia">Злые роботы увольняют из Amazon</a> - <em>00:17:23</em>.</li>
<li>Шутка от Алексея - <em>00:33:01</em></li>
<li><a href="https://donjon.ledger.com/kaspersky-password-manager/">Пароли от Касперского не очень</a> - <em>00:33:16</em>.</li>
<li>Шутка от Алексея - <em>00:44:43</em></li>
<li><a href="https://copilot.github.com/">GitHub Copilot и проблема GPL</a> - <em>00:45:06</em>.</li>
<li><a href="https://habr.com/ru/company/selectel/blog/565644/">Четырехдневная рабочая неделя</a> - <em>00:56:31</em>.</li>
<li>Появились Умпутун и Ксюша - <em>01:04:54</em></li>
<li><a href="https://techraptor.net/gaming/news/amazon-games-personal-game-policy">Невероятная политика Amazon про личные проекты</a> - <em>01:09:37</em>.</li>
<li><a href="https://www.opennet.ru/opennews/art.shtml?num=55444">Созданы форки Audacity, избавленные от телеметрии</a> - <em>01:31:16</em>.</li>
<li><a href="https://www.opennet.ru/opennews/art.shtml?num=55452">Создатель форка Audacity покинул проект</a> - <em>01:33:04</em>.</li>
<li><a href="https://habr.com/ru/company/macloud/blog/566092/">Обзор самых неоднозначных проектов на Kickstarter</a> - <em>02:04:09</em>.</li>
<li><a href="https://radio-t.com/p/2021/07/06/prep-762/">Темы слушателей</a> - <em>02:29:22</em>.</li>
</ul></p>
<p><em>Спонсор этого выпуска <a href="http://do.co/radiot-mongo">DigitalOcean</a></em></p>
<p><a href="https://cdn.radio-t.com/rt_podcast762.mp3">аудио</a> • <a href="https://chat.radio-t.com/logs/radio-t-762.html">лог чата</a></p>
<audio src="https://cdn.radio-t.com/rt_podcast762.mp3" preload="none"></audio>]]></description>
      <link>https://radio-t.com/p/2021/07/10/podcast-762/</link>
      <guid>https://radio-t.com/p/2021/07/10//podcast-762/</guid>
      <pubDate>Sat, 10 Jul 2021 18:31:09 EST</pubDate>
      <itunes:author>Umputun, Bobuk, Gray, Ksenks, Alek.sys</itunes:author>
      <itunes:summary><![CDATA[<p><img src="https://radio-t.com/images/radio-t/rt762.jpg" alt=""></p>
<ul>
<li>Официальный кат №1 от Алексея - <em>00:14:35</em></li>
<li>Официальный кат №2 от Алексея - <em>00:16:10</em></li>
<li><a href="https://www.theguardian.com/commentisfree/2021/jul/05/amazon-worker-fired-app-dystopia">Злые роботы увольняют из Amazon</a> - <em>00:17:23</em>.</li>
</ul>
<p><em>Спонсор этого выпуска <a href="http://do.co/radiot-mongo">DigitalOcean</a></em></p>
<p><a href="https://cdn.radio-t.com/rt_podcast762.mp3">аудио</a> • <a href="https://chat.radio-t.com/logs/radio-t-762.html">лог чата</a></p>
<audio src="https://cdn.radio-t.com/rt_podcast762.mp3" preload="none"></audio>]]></itunes:summary>
      <itunes:image href="https://radio-t.com/images/radio-t/rt762.jpg" />
      <enclosure url="http://cdn.radio-t.com/rt_podcast762.mp3" type="audio/mp3" length="166608723" />
    <author>podcast@radio-t.com (Umputun, Bobuk, Gray, Ksenks, Alek.sys)</author><media:content url="http://cdn.radio-t.com/rt_podcast762.mp3" fileSize="166608723" type="audio/mp3" /><itunes:explicit>no</itunes:explicit><itunes:subtitle>Подкаст выходного дня - импровизации на темы высоких технологий</itunes:subtitle><itunes:keywords>hitech,russian,radiot,tech,news,радио</itunes:keywords>
  </item>
  <media:credit role="author">Umputun, Bobuk, Gray, Ksenks, Alek.sys</media:credit><media:rating>nonadult</media:rating><media:description type="plain">Еженедельные импровизации на хай–тек темы</media:description></channel>
</rss>
`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(testFeed))
		assert.NoError(t, err)
	}))
	r, err := Parse(ts.URL)
	assert.NoError(t, err)
	assert.Equal(t, "Еженедельные импровизации на хай–тек темы", r.Description)
	require.Equal(t, 1, len(r.ItemList))
	assert.Equal(t, "podcast@radio-t.com (Umputun, Bobuk, Gray, Ksenks, Alek.sys)", r.ItemList[0].Author)
}

func TestFeedParseBadBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("bad"))
		assert.NoError(t, err)
	}))
	r, err := Parse(ts.URL)
	assert.Error(t, err)
	assert.Empty(t, r)
}

func TestFeedParseHttpError(t *testing.T) {
	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		ts.CloseClientConnections()
	}))

	_, err := Parse(ts.URL)

	assert.Error(t, err)
}

func TestParseDateTime(t *testing.T) {
	tbl := []struct {
		inp string
		err error
		out string
	}{
		{"", fmt.Errorf("can't parse empty date-time"), time.Now().Format(time.RFC822Z)},
		{"05 Mar 14 22:08 +0400", nil, "05 Mar 14 22:08 +0400"},           // RFC822Z
		{"05 Mar 14 22:08 MST", nil, "05 Mar 14 22:08 +0000"},             // RFC822
		{"Mon, 02 Jan 2006 15:04:05 -0700", nil, "02 Jan 06 15:04 -0700"}, // RFC1123Z
		{"Mon, 02 Jan 2006 15:04:05 MST", nil, "02 Jan 06 15:04 +0000"},   // RFC1123
		{"2006-01-02 15:04:05 -0700", nil, "02 Jan 06 15:04 -0700"},
		{"2017-09-30T14:11:48-0500", nil, "30 Sep 17 14:11 -0500"},
		{"100500", fmt.Errorf("can't parse timestamp 100500"), time.Now().Format(time.RFC822Z)},
	}

	rss := Rss2{}
	for _, tb := range tbl {
		ts, err := rss.parseDateTime(tb.inp)
		assert.Equal(t, tb.err, err)
		assert.Equal(t, tb.out, ts.Format(time.RFC822Z))
	}
}

func TestNormalizeIfLastBuildDateAndPubDateInvalidFormat(t *testing.T) {
	cases := []struct {
		lastBuildDate string
		pubDate       string
	}{
		{"invalidFormat", "02 Jan 06 15:04 MST"},
		{"02 Jan 06 15:04 MST", "invalidFormat"},
	}

	for i, tc := range cases {
		i := i
		tc := tc
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			rss := Rss2{
				LastBuildDate: tc.lastBuildDate,
				PubDate:       tc.pubDate,
			}

			got, err := rss.Normalize()

			assert.NoError(t, err)
			assert.Equal(t, got.PubDate, "Mon, 02 Jan 2006 15:04:00 +0000")
		})
	}
}

func TestParseAtomInvalidContent(t *testing.T) {
	invalidContent := []byte(`<?xml version="1.0" encoding="UTF-8"?> <rss`)

	_, err := parseAtom(invalidContent)

	assert.EqualError(t, err, "can't parse atom1: XML syntax error on line 1: unexpected EOF")
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

	assert.NoError(t, err)
	assert.Equal(t, got.Title, "Example Feed")
	assert.Equal(t, got.Description, "")

	assert.Len(t, got.ItemList, 2)
	assert.Equal(t, got.ItemList[0].Title, "Atom-Powered Robots Run Amok")
	assert.Equal(t, got.ItemList[0].Description, template.HTML("Some text."))

	assert.Equal(t, got.ItemList[1].Description, template.HTML("Example content"))
}

func TestParseFeedContentIfRSSVersionNot2_0(t *testing.T) {
	rss := `<?xml version="1.0" encoding="UTF-8"?>
<rss xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd" xmlns:media="http://search.yahoo.com/mrss/" version="3.0">
  <channel>
    <title>Радио-Т</title>
    <link>https://radio-t.com</link>
    <language>ru</language>
  </channel>
</rss>`

	_, err := parseFeedContent([]byte(rss))

	assert.EqualError(t, err, "not RSS 2.0")
}

func TestParseFeedContentIfAtom1_0(t *testing.T) {
	atom1 := `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Example Feed</title>
  <link href="http://example.org/"/>
  <updated>2003-12-13T18:30:02Z</updated>
</feed>`

	got, err := parseFeedContent([]byte(atom1))

	assert.NoError(t, err)
	assert.Equal(t, got.Title, "Example Feed")
}

func TestParseFeedContentIfNotAtom1_0(t *testing.T) {
	atom1 := `<?xml version="2.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Example Feed</title>
  <link href="http://example.org/"/>
  <updated>2003-12-13T18:30:02Z</updated>
</feed>`

	_, err := parseFeedContent([]byte(atom1))

	assert.EqualError(t, err, "can't parse feed content: xml: unsupported version \"2.0\"; only version 1.0 is supported")
}

func TestParseFeedContentIfRSSVersionEmptyContent(t *testing.T) {
	rss := `<?xml version="1.0" encoding="UTF-8"?>
<rss xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd" xmlns:media="http://search.yahoo.com/mrss/" version="2.0">
  <channel>
    <title>Радио-Т</title>
    <link>https://radio-t.com</link>
    <language>ru</language>

	<item>
	  <title>Example</title>
	  <description>Description</description>
	  <encoded>Content</encoded>
	</item>
  </channel>
</rss>`

	got, err := parseFeedContent([]byte(rss))

	assert.NoError(t, err)
	assert.Equal(t, got.ItemList[0].Content, template.HTML("Content"))
	assert.Equal(t, got.ItemList[0].Description, template.HTML("Content"))
}
