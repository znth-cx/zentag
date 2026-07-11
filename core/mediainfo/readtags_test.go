package mediainfo

import (
	"context"
	"reflect"
	"testing"

	"codeberg.org/Ether/zentag/core/metadata"
)

const tagsJSON = `{
  "media": {
    "track": [
      {
        "@type": "General",
        "Title": "The Way of Kings",
        "Genre": "Fantasy;Epic",
        "extra": {
          "author": "Brandon Sanderson",
          "narrator": "Michael Kramer;Kate Reading",
          "publisher": "Macmillan",
          "subtitle": "Stormlight Archive 1",
          "description": "A war-torn world.",
          "series": "Stormlight Archive",
          "series-part": "1",
          "language": "en",
          "isbn": "9780765326355",
          "asin": "B0041OW6EG",
          "year": "2010"
        }
      },
      {"@type": "Audio", "Format": "AAC", "BitRate": "128000"}
    ]
  }
}`

func TestReadTags_HappyPath(t *testing.T) {
	fr := &fakeRunner{out: []byte(tagsJSON)}
	w := &Wrapper{BinPath: "mediainfo", Runner: fr}

	got, err := w.ReadTags(context.Background(), "book.m4b")
	if err != nil {
		t.Fatalf("ReadTags() error = %v", err)
	}

	want := TagSet{
		Author:      []string{"Brandon Sanderson"},
		Narrator:    []string{"Michael Kramer", "Kate Reading"},
		Publisher:   []string{"Macmillan"},
		Genre:       []string{"Fantasy", "Epic"},
		Title:       "The Way of Kings",
		Subtitle:    "Stormlight Archive 1",
		Description: "A war-torn world.",
		Year:        2010,
		Series: []metadata.SeriesEntry{
			{Name: "Stormlight Archive", Part: "1"},
		},
		Language: "en",
		ISBN:     "9780765326355",
		ASIN:     "B0041OW6EG",
		Extra: map[string]string{
			"author":      "Brandon Sanderson",
			"narrator":    "Michael Kramer;Kate Reading",
			"publisher":   "Macmillan",
			"subtitle":    "Stormlight Archive 1",
			"description": "A war-torn world.",
			"series":      "Stormlight Archive",
			"series-part": "1",
			"language":    "en",
			"isbn":        "9780765326355",
			"asin":        "B0041OW6EG",
			"year":        "2010",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("ReadTags() =\n%+v\nwant\n%+v", got, want)
	}
}

func TestReadTags_MissingGeneralTrack(t *testing.T) {
	fr := &fakeRunner{out: []byte(`{"media":{"track":[{"@type":"Audio","Format":"AAC","BitRate":"128000"}]}}`)}
	w := &Wrapper{BinPath: "mediainfo", Runner: fr}

	_, err := w.ReadTags(context.Background(), "book.m4b")
	if err == nil {
		t.Fatal("ReadTags() error = nil, want error")
	}
}

func TestReadTags_MalformedJSON(t *testing.T) {
	fr := &fakeRunner{out: []byte(`{not valid json`)}
	w := &Wrapper{BinPath: "mediainfo", Runner: fr}

	_, err := w.ReadTags(context.Background(), "book.m4b")
	if err == nil {
		t.Fatal("ReadTags() error = nil, want error")
	}
}

func TestReadTags_MissingYearDefaultsToZero(t *testing.T) {
	fr := &fakeRunner{out: []byte(`{"media":{"track":[
		{"@type":"General","Title":"Some Book"},
		{"@type":"Audio","Format":"AAC","BitRate":"128000"}
	]}}`)}
	w := &Wrapper{BinPath: "mediainfo", Runner: fr}

	got, err := w.ReadTags(context.Background(), "book.m4b")
	if err != nil {
		t.Fatalf("ReadTags() error = %v", err)
	}
	if got.Year != 0 {
		t.Errorf("Year = %d, want 0", got.Year)
	}
}

func TestReadTags_IgnoresCDEKButUsesAudioLanguage(t *testing.T) {
	fr := &fakeRunner{out: []byte(`{"media":{"track":[
		{"@type":"General","Title":"Some Book","extra":{"CDEK":"B0FALLBACKASIN"}},
		{"@type":"Audio","Format":"AAC","BitRate":"128000","Language":"fr"}
	]}}`)}
	w := &Wrapper{BinPath: "mediainfo", Runner: fr}

	got, err := w.ReadTags(context.Background(), "book.m4b")
	if err != nil {
		t.Fatalf("ReadTags() error = %v", err)
	}
	if got.ASIN != "" {
		t.Errorf("ASIN = %q, want empty — ReadTags must not fall back to CDEK", got.ASIN)
	}
	// Audio.Language is the only place mediainfo exposes MP3/FLAC language tags. Strict ReadTags must use it.
	if got.Language != "fr" {
		t.Errorf("Language = %q, want %q from the Audio track", got.Language, "fr")
	}
}

func TestReadTagsGreedy_HappyPath(t *testing.T) {
	fr := &fakeRunner{out: []byte(tagsJSON)}
	w := &Wrapper{BinPath: "mediainfo", Runner: fr}

	got, err := w.ReadTagsGreedy(context.Background(), "book.m4b")
	if err != nil {
		t.Fatalf("ReadTagsGreedy() error = %v", err)
	}

	want := TagSet{
		Author:      []string{"Brandon Sanderson"},
		Narrator:    []string{"Michael Kramer", "Kate Reading"},
		Publisher:   []string{"Macmillan"},
		Genre:       []string{"Fantasy", "Epic"},
		Title:       "The Way of Kings",
		Subtitle:    "Stormlight Archive 1",
		Description: "A war-torn world.",
		Year:        2010,
		Series: []metadata.SeriesEntry{
			{Name: "Stormlight Archive", Part: "1"},
		},
		Language: "en",
		ISBN:     "9780765326355",
		ASIN:     "B0041OW6EG",
		Extra: map[string]string{
			"author":      "Brandon Sanderson",
			"narrator":    "Michael Kramer;Kate Reading",
			"publisher":   "Macmillan",
			"subtitle":    "Stormlight Archive 1",
			"description": "A war-torn world.",
			"series":      "Stormlight Archive",
			"series-part": "1",
			"language":    "en",
			"isbn":        "9780765326355",
			"asin":        "B0041OW6EG",
			"year":        "2010",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("ReadTagsGreedy() =\n%+v\nwant\n%+v", got, want)
	}
}

func TestReadTagsGreedy_ASINFallsBackToCDEKTag(t *testing.T) {
	fr := &fakeRunner{out: []byte(`{"media":{"track":[
		{"@type":"General","Title":"Some Book","extra":{"CDEK":"B0FALLBACKASIN"}},
		{"@type":"Audio","Format":"AAC","BitRate":"128000"}
	]}}`)}
	w := &Wrapper{BinPath: "mediainfo", Runner: fr}

	got, err := w.ReadTagsGreedy(context.Background(), "book.m4b")
	if err != nil {
		t.Fatalf("ReadTagsGreedy() error = %v", err)
	}
	if got.ASIN != "B0FALLBACKASIN" {
		t.Errorf("ASIN = %q, want fallback from CDEK tag", got.ASIN)
	}
}

func TestReadTagsGreedy_NamedASINTagWinsOverCDEKFallback(t *testing.T) {
	fr := &fakeRunner{out: []byte(`{"media":{"track":[
		{"@type":"General","Title":"Some Book","extra":{"asin":"B0NAMEDASIN","CDEK":"B0FALLBACKASIN"}},
		{"@type":"Audio","Format":"AAC","BitRate":"128000"}
	]}}`)}
	w := &Wrapper{BinPath: "mediainfo", Runner: fr}

	got, err := w.ReadTagsGreedy(context.Background(), "book.m4b")
	if err != nil {
		t.Fatalf("ReadTagsGreedy() error = %v", err)
	}
	if got.ASIN != "B0NAMEDASIN" {
		t.Errorf("ASIN = %q, want named asin tag to win over CDEK fallback", got.ASIN)
	}
}

func TestReadTagsGreedy_LanguageFallsBackToAudioTrack(t *testing.T) {
	fr := &fakeRunner{out: []byte(`{"media":{"track":[
		{"@type":"General","Title":"Some Book"},
		{"@type":"Audio","Format":"AAC","BitRate":"128000","Language":"fr"}
	]}}`)}
	w := &Wrapper{BinPath: "mediainfo", Runner: fr}

	got, err := w.ReadTagsGreedy(context.Background(), "book.m4b")
	if err != nil {
		t.Fatalf("ReadTagsGreedy() error = %v", err)
	}
	if got.Language != "fr" {
		t.Errorf("Language = %q, want fallback from Audio track", got.Language)
	}
}

func TestReadTagsGreedy_NamedLanguageTagWinsOverAudioFallback(t *testing.T) {
	fr := &fakeRunner{out: []byte(`{"media":{"track":[
		{"@type":"General","Title":"Some Book","extra":{"language":"en"}},
		{"@type":"Audio","Format":"AAC","BitRate":"128000","Language":"fr"}
	]}}`)}
	w := &Wrapper{BinPath: "mediainfo", Runner: fr}

	got, err := w.ReadTagsGreedy(context.Background(), "book.m4b")
	if err != nil {
		t.Fatalf("ReadTagsGreedy() error = %v", err)
	}
	if got.Language != "en" {
		t.Errorf("Language = %q, want named language tag to win over Audio fallback", got.Language)
	}
}

func TestReadTags_DescriptionFromNamedComment(t *testing.T) {
	// mp4tag writes description to ©cmt (General.Comment), not extra. Must recover it there.
	fr := &fakeRunner{out: []byte(`{"media":{"track":[
		{"@type":"General","Title":"Flag in Exile","Comment":"A war-torn world."},
		{"@type":"Audio","Format":"AAC","BitRate":"128000"}
	]}}`)}
	w := &Wrapper{BinPath: "mediainfo", Runner: fr}

	got, err := w.ReadTags(context.Background(), "book.m4b")
	if err != nil {
		t.Fatalf("ReadTags() error = %v", err)
	}
	if got.Description != "A war-torn world." {
		t.Errorf("Description = %q, want it recovered from General.Comment", got.Description)
	}
}

func TestReadTags_ExtraDescriptionWinsOverComment(t *testing.T) {
	fr := &fakeRunner{out: []byte(`{"media":{"track":[
		{"@type":"General","Title":"B","Comment":"named","extra":{"description":"freeform"}},
		{"@type":"Audio","Format":"AAC","BitRate":"128000"}
	]}}`)}
	w := &Wrapper{BinPath: "mediainfo", Runner: fr}

	got, err := w.ReadTags(context.Background(), "book.m4b")
	if err != nil {
		t.Fatalf("ReadTags() error = %v", err)
	}
	if got.Description != "freeform" {
		t.Errorf("Description = %q, want extra 'description' to win over General.Comment", got.Description)
	}
}
