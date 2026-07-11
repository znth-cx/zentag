package mediainfo

import (
	"context"
	"errors"
	"testing"
)

type fakeRunner struct {
	gotBinPath string
	gotArgs    []string
	out        []byte
	err        error
}

func (f *fakeRunner) Run(_ context.Context, binPath string, args []string) ([]byte, error) {
	f.gotBinPath = binPath
	f.gotArgs = args
	return f.out, f.err
}

const happyPathJSON = `{
  "media": {
    "track": [
      {"@type": "General", "Format": "MPEG-4"},
      {"@type": "Audio", "Format": "AAC", "BitRate": "128000"}
    ]
  }
}`

func TestProbe_HappyPath(t *testing.T) {
	fr := &fakeRunner{out: []byte(happyPathJSON)}
	w := &Wrapper{BinPath: "mediainfo", Runner: fr}

	got, err := w.Probe(context.Background(), "book.m4b")
	if err != nil {
		t.Fatalf("Probe() error = %v", err)
	}

	want := TechnicalInfo{Container: "MPEG-4", Codec: "AAC", Bitrate: 128}
	if got != want {
		t.Errorf("Probe() = %+v, want %+v", got, want)
	}
}

func TestProbe_RunsWithJSONOutputFlagAndPath(t *testing.T) {
	fr := &fakeRunner{out: []byte(happyPathJSON)}
	w := &Wrapper{BinPath: "mediainfo", Runner: fr}

	if _, err := w.Probe(context.Background(), "book.m4b"); err != nil {
		t.Fatalf("Probe() error = %v", err)
	}

	if fr.gotBinPath != "mediainfo" {
		t.Errorf("binPath = %q, want %q", fr.gotBinPath, "mediainfo")
	}
	want := []string{"--Output=JSON", "book.m4b"}
	if len(fr.gotArgs) != len(want) {
		t.Fatalf("args = %q, want %q", fr.gotArgs, want)
	}
	for i := range want {
		if fr.gotArgs[i] != want[i] {
			t.Errorf("args[%d] = %q, want %q", i, fr.gotArgs[i], want[i])
		}
	}
}

func TestDump_HappyPath(t *testing.T) {
	fr := &fakeRunner{out: []byte("General\nComplete name : book.m4b\n")}
	w := &Wrapper{BinPath: "mediainfo", Runner: fr}

	got, err := w.Dump(context.Background(), "book.m4b")
	if err != nil {
		t.Fatalf("Dump() error = %v", err)
	}
	if got != "General\nComplete name : book.m4b\n" {
		t.Errorf("Dump() = %q", got)
	}

	if fr.gotBinPath != "mediainfo" {
		t.Errorf("binPath = %q, want %q", fr.gotBinPath, "mediainfo")
	}
	want := []string{"book.m4b"}
	if len(fr.gotArgs) != len(want) || fr.gotArgs[0] != want[0] {
		t.Errorf("args = %q, want %q", fr.gotArgs, want)
	}
}

func TestDump_RunnerError(t *testing.T) {
	fr := &fakeRunner{err: errors.New("exit status 1")}
	w := &Wrapper{BinPath: "mediainfo", Runner: fr}

	if _, err := w.Dump(context.Background(), "book.m4b"); err == nil {
		t.Fatal("Dump() error = nil, want error")
	}
}

func TestProbe_BitRateVariants(t *testing.T) {
	cases := []struct {
		name    string
		bitrate string
		want    int
		wantErr bool
	}{
		{name: "plain digits", bitrate: "128000", want: 128},
		{name: "whitespace", bitrate: " 128000 ", want: 128},
		{name: "float", bitrate: "128000.0", want: 128},
		{name: "garbage", bitrate: "garbage", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := `{"media":{"track":[
				{"@type":"General","Format":"MPEG-4"},
				{"@type":"Audio","Format":"AAC","BitRate":"` + tc.bitrate + `"}
			]}}`
			fr := &fakeRunner{out: []byte(out)}
			w := &Wrapper{BinPath: "mediainfo", Runner: fr}

			got, err := w.Probe(context.Background(), "book.m4b")
			if tc.wantErr {
				if err == nil {
					t.Fatal("Probe() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Probe() error = %v", err)
			}
			if got.Bitrate != tc.want {
				t.Errorf("Bitrate = %d, want %d", got.Bitrate, tc.want)
			}
		})
	}
}

func TestProbe_ErrorCases(t *testing.T) {
	cases := []struct {
		name string
		out  string
		err  error
	}{
		{
			name: "runner error",
			out:  "",
			err:  errors.New("exit status 1"),
		},
		{
			name: "malformed json",
			out:  `{not valid json`,
		},
		{
			name: "missing audio track",
			out: `{"media":{"track":[
				{"@type":"General","Format":"MPEG-4"}
			]}}`,
		},
		{
			name: "missing general track",
			out: `{"media":{"track":[
				{"@type":"Audio","Format":"AAC","BitRate":"128000"}
			]}}`,
		},
		{
			name: "non-numeric bitrate",
			out: `{"media":{"track":[
				{"@type":"General","Format":"MPEG-4"},
				{"@type":"Audio","Format":"AAC","BitRate":"not-a-number"}
			]}}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fr := &fakeRunner{out: []byte(tc.out), err: tc.err}
			w := &Wrapper{BinPath: "mediainfo", Runner: fr}

			_, err := w.Probe(context.Background(), "book.m4b")
			if err == nil {
				t.Fatal("Probe() error = nil, want error")
			}
		})
	}
}
