// Package rgbin embeds the microsoft/ripgrep-prebuilt payload fetched by
// `task common:fetch:rg` (gitignored `bin/rg.bin`).
package rgbin

import "embed"

// Version is the microsoft/ripgrep-prebuilt release tag the fetch script pins.
const Version = "v15.0.1"

//go:embed all:bin
var binFS embed.FS

// Binary returns the embedded ripgrep payload, or nil until fetch:rg has run.
func Binary() []byte {
	b, err := binFS.ReadFile("bin/rg.bin")
	if err != nil {
		return nil
	}
	return b
}
