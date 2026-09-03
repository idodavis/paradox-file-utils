// Command pmt-steamugc publishes a Workshop item via ISteamUGC (CGO-free helper).
package main

import (
	"encoding/json"
	"os"

	"paradox-modding-tools/services/internal/steamugc"
)

func main() {
	lib := os.Getenv("PMT_STEAM_API")
	if lib == "" {
		steamugc.EncodeResponse(os.Stdout, steamugc.Response{
			Error: "PMT_STEAM_API is not set",
		})
		os.Exit(1)
	}
	var req steamugc.Request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		steamugc.EncodeResponse(os.Stdout, steamugc.Response{Error: err.Error()})
		os.Exit(1)
	}
	resp := steamugc.Run(lib, req)
	steamugc.EncodeResponse(os.Stdout, resp)
	if resp.Error != "" {
		os.Exit(1)
	}
}
