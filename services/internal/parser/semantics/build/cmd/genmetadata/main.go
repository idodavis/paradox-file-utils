package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"paradox-modding-tools/services/internal/parser/semantics/build"
)

func main() {
	game := flag.String("game", "", "game key (e.g. ck3, eu5)")
	root := flag.String("root", "", "absolute path to game install root")
	flag.Parse()

	if strings.TrimSpace(*game) == "" || strings.TrimSpace(*root) == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./services/internal/parser/semantics/build/cmd/genmetadata -game <game> -root <game-root>")
		os.Exit(2)
	}

	start := time.Now()
	if err := build.Generate(*root, *game); err != nil {
		fmt.Fprintf(os.Stderr, "generate failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("semantic metadata generated for %s in %s\n", strings.ToUpper(strings.TrimSpace(*game)), time.Since(start).Round(time.Millisecond))
}
