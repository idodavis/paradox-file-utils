// parse_test.go covers the loc dialect: entries, header/tab/unterminated errors,
// last-quote-wins values, and byte-offset ranges.

package loc

import "testing"

func TestParseBasicEntries(t *testing.T) {
	src := "l_english:\n key1:0 \"Hello\"\n key2: \"World\" # comment\n"
	r := Parse(src)
	if r.Language != "english" {
		t.Fatalf("language = %q", r.Language)
	}
	if len(r.Entries) != 2 {
		t.Fatalf("entries = %d %+v", len(r.Entries), r.Entries)
	}
	if r.Entries[0].Key != "key1" || r.Entries[0].Version != 0 || r.Entries[0].Value != "Hello" {
		t.Fatalf("entry0 = %+v", r.Entries[0])
	}
	if r.Entries[1].Key != "key2" || r.Entries[1].Version != -1 || r.Entries[1].Value != "World" {
		t.Fatalf("entry1 = %+v", r.Entries[1])
	}

	// Inner quotes are literal; value runs to the LAST quote on the line.
	q := Parse("l_english:\n k: \"say \"\"hi\"\" now\"\n")
	if len(q.Entries) != 1 || q.Entries[0].Value != "say \"\"hi\"\" now" {
		t.Fatalf("last quote = %+v", q.Entries)
	}
	rng := "l_english:\n k: \"Hi\"\n"
	e := Parse(rng).Entries[0]
	if rng[e.ValueRange.Start:e.ValueRange.End] != "Hi" ||
		rng[e.KeyRange.Start:e.KeyRange.End] != "k" {
		t.Fatalf("ranges key=%q val=%q",
			rng[e.KeyRange.Start:e.KeyRange.End],
			rng[e.ValueRange.Start:e.ValueRange.End])
	}
}

func TestParseBOM(t *testing.T) {
	src := "\ufeffl_english:\n k:0 \"Hi\"\n"
	r := Parse(src)
	if !r.HadBOM {
		t.Fatal("HadBOM")
	}
	if r.Language != "english" || len(r.Entries) != 1 {
		t.Fatalf("parsed %+v", r)
	}
	e := r.Entries[0]
	if src[e.KeyRange.Start:e.KeyRange.End] != "k" ||
		src[e.ValueRange.Start:e.ValueRange.End] != "Hi" {
		t.Fatalf("BOM ranges key=%q val=%q",
			src[e.KeyRange.Start:e.KeyRange.End], src[e.ValueRange.Start:e.ValueRange.End])
	}
}

func TestInterps(t *testing.T) {
	src := "l_english:\n a:0 \"see $used$ and $used|U$\"\n"
	e := Parse(src).Entries[0]
	got := Interps(e.Value, e.ValueRange.Start)
	if len(got) != 2 || got[0].Key != "used" || got[1].Key != "used" {
		t.Fatalf("interps=%+v", got)
	}
	if src[got[0].KeyRange.Start:got[0].KeyRange.End] != "used" ||
		src[got[0].WrapRange.Start:got[0].WrapRange.End] != "$used$" ||
		src[got[1].WrapRange.Start:got[1].WrapRange.End] != "$used|U$" {
		t.Fatalf("interp ranges %+v", got)
	}
	if len(Interps("plain $notclosed", 0)) != 0 {
		t.Fatal("unterminated interpolations are not keys")
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name, src string
		code      ErrorCode
	}{
		{"no header", "key: \"value\"\n", ErrNoHeader},
		{"content before header", "stray content\nl_english:\n key: \"v\"\n", ErrContentBeforeHeader},
		{"tab indent", "l_english:\n\tkey: \"v\"\n", ErrTabIndent},
		{"unterminated value", "l_english:\n key: \"unterminated\n", ErrUnterminatedValue},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Parse(tt.src)
			found := false
			for _, e := range r.Errors {
				if e.Code == tt.code {
					found = true
				}
			}
			if !found {
				t.Fatalf("want %s, got %+v", tt.code, r.Errors)
			}
		})
	}
}
