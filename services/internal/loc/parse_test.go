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
}

func TestParseNoHeader(t *testing.T) {
	r := Parse("key: \"value\"\n")
	found := false
	for _, e := range r.Errors {
		if e.Code == ErrNoHeader {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected no-header error, got %+v", r.Errors)
	}
}

func TestParseContentBeforeHeader(t *testing.T) {
	r := Parse("stray content\nl_english:\n key: \"v\"\n")
	found := false
	for _, e := range r.Errors {
		if e.Code == ErrContentBeforeHeader {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected content-before-header, got %+v", r.Errors)
	}
}

func TestParseTabIndent(t *testing.T) {
	r := Parse("l_english:\n\tkey: \"v\"\n")
	found := false
	for _, e := range r.Errors {
		if e.Code == ErrTabIndent {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected tab-indent, got %+v", r.Errors)
	}
}

func TestParseUnterminatedValue(t *testing.T) {
	r := Parse("l_english:\n key: \"unterminated\n")
	found := false
	for _, e := range r.Errors {
		if e.Code == ErrUnterminatedValue {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected unterminated-value, got %+v", r.Errors)
	}
}

func TestParseLastQuoteWins(t *testing.T) {
	// Inner quotes are literal; value runs to the LAST quote on the line.
	src := "l_english:\n k: \"say \"\"hi\"\" now\"\n"
	r := Parse(src)
	if len(r.Entries) != 1 || r.Entries[0].Value != "say \"\"hi\"\" now" {
		t.Fatalf("entry = %+v", r.Entries)
	}
}

func TestValueRangeByteOffsets(t *testing.T) {
	src := "l_english:\n k: \"Hi\"\n"
	r := Parse(src)
	e := r.Entries[0]
	if src[e.ValueRange.Start:e.ValueRange.End] != "Hi" {
		t.Fatalf("value range slice = %q", src[e.ValueRange.Start:e.ValueRange.End])
	}
	if src[e.KeyRange.Start:e.KeyRange.End] != "k" {
		t.Fatalf("key range slice = %q", src[e.KeyRange.Start:e.KeyRange.End])
	}
}
