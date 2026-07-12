package ebookmeta

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// OPFMetadata is the OPF subset zentag consumes.
type OPFMetadata struct {
	Title       string
	Authors     []string
	Series      string
	SeriesIndex float64
	ISBN        string
	ASIN        string
	Publisher   []string
	Language    string
	Description string
	Tags        []string
	Year        int
}

// encoding/xml matches local names regardless of dc: prefix; no namespace URLs needed.
type opfPackage struct {
	XMLName  xml.Name `xml:"package"`
	Metadata opfMeta  `xml:"metadata"`
}

type opfMeta struct {
	Title       string        `xml:"title"`
	Creators    []opfCreator  `xml:"creator"`
	Identifiers []opfIdent    `xml:"identifier"`
	Language    string        `xml:"language"`
	Date        string        `xml:"date"`
	Publisher   []string      `xml:"publisher"`
	Description string        `xml:"description"`
	Subjects    []string      `xml:"subject"`
	Metas       []opfNameMeta `xml:"meta"`
}

type opfCreator struct {
	Role  string `xml:"role,attr"`
	Value string `xml:",chardata"`
}

type opfIdent struct {
	Scheme string `xml:"scheme,attr"`
	Value  string `xml:",chardata"`
}

type opfNameMeta struct {
	Name    string `xml:"name,attr"`
	Content string `xml:"content,attr"`
}

var yearRE = regexp.MustCompile(`\d{4}`)

func parseOPF(data []byte) (*OPFMetadata, error) {
	var pkg opfPackage
	if err := xml.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse OPF metadata: %w", err)
	}
	md := pkg.Metadata
	out := &OPFMetadata{
		Title:       strings.TrimSpace(md.Title),
		Language:    strings.TrimSpace(md.Language),
		Description: strings.TrimSpace(md.Description),
	}
	for _, c := range md.Creators {
		v := strings.TrimSpace(c.Value)
		if v == "" {
			continue
		}
		if c.Role == "" || strings.EqualFold(c.Role, "aut") {
			out.Authors = append(out.Authors, v)
		}
	}
	for _, id := range md.Identifiers {
		v := strings.TrimSpace(id.Value)
		switch strings.ToUpper(id.Scheme) {
		case "ISBN":
			if out.ISBN == "" {
				out.ISBN = v
			}
		case "ASIN", "MOBI-ASIN", "AMAZON":
			if out.ASIN == "" {
				out.ASIN = v
			}
		}
	}
	for _, p := range md.Publisher {
		if s := strings.TrimSpace(p); s != "" {
			out.Publisher = append(out.Publisher, s)
		}
	}
	for _, s := range md.Subjects {
		if t := strings.TrimSpace(s); t != "" {
			out.Tags = append(out.Tags, t)
		}
	}
	for _, m := range md.Metas {
		switch m.Name {
		case "calibre:series":
			out.Series = strings.TrimSpace(m.Content)
		case "calibre:series_index":
			if f, err := strconv.ParseFloat(strings.TrimSpace(m.Content), 64); err == nil {
				out.SeriesIndex = f
			}
		}
	}
	if md.Date != "" {
		if m := yearRE.FindString(md.Date); m != "" {
			if y, err := strconv.Atoi(m); err == nil {
				out.Year = y
			}
		}
	}
	return out, nil
}
