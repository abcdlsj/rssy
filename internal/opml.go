package internal

import (
	"encoding/xml"
	"time"
)

type OPML struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    Head     `xml:"head"`
	Body    Body     `xml:"body"`
}

type Head struct {
	Title        string `xml:"title"`
	DateCreated  string `xml:"dateCreated"`
	DateModified string `xml:"dateModified"`
}

type Body struct {
	Outlines []Outline `xml:"outline"`
}

type Outline struct {
	Text     string    `xml:"text,attr"`
	Type     string    `xml:"type,attr"`
	XMLURL   string    `xml:"xmlUrl,attr"`
	HTMLURL  string    `xml:"htmlUrl,attr,omitempty"`
	Language string    `xml:"language,attr,omitempty"`
	Title    string    `xml:"title,attr,omitempty"`
	Outlines []Outline `xml:"outline"`
}

func parseOPML(data []byte) (*OPML, error) {
	var opml OPML
	if err := xml.Unmarshal(data, &opml); err != nil {
		return nil, err
	}

	return &opml, nil
}

func exportOPML(feeds []Feed) ([]byte, error) {
	outlines := make([]Outline, len(feeds))
	for i, feed := range feeds {
		outlines[i] = Outline{
			Title:  feed.Title,
			Text:   feed.Title,
			Type:   "rss",
			XMLURL: feed.URL,
		}
	}

	opml := OPML{
		Version: "2.0",
		Head: Head{
			Title:        "My Feeds",
			DateCreated:  time.Now().Format(time.RFC1123Z),
			DateModified: time.Now().Format(time.RFC1123Z),
		},
		Body: Body{
			Outlines: outlines,
		},
	}

	bytes, err := xml.MarshalIndent(opml, "", "  ")
	if err != nil {
		return nil, err
	}

	return bytes, nil
}
