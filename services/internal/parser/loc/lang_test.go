// lang_test.go covers language-id extraction from loc filenames.

package loc

import "testing"

func TestLanguageFromFilename(t *testing.T) {
	cases := map[string]string{
		"events_l_english.yml":   "english",
		"foo_l_french.yml":       "french",
		"bar_l_SimpChinese.yaml": "simpchinese",
		"descriptor.mod":         "",
		"noconvention.yml":       "",
	}
	for name, want := range cases {
		if got := LanguageFromFilename(name); got != want {
			t.Errorf("LanguageFromFilename(%q) = %q want %q", name, got, want)
		}
	}
	if got := LanguageOf("events.yml", "english"); got != "english" {
		t.Errorf("LanguageOf header fallback = %q", got)
	}
	if got := LanguageOf("foo_l_french.yml", "english"); got != "french" {
		t.Errorf("LanguageOf filename wins = %q", got)
	}
}
