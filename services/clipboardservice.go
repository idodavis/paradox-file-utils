package services

import "github.com/wailsapp/wails/v3/pkg/application"

// ClipboardService exposes the native clipboard to the frontend.
type ClipboardService struct{}

// CopyToClipboard writes text to the system clipboard.
func (c *ClipboardService) CopyToClipboard(text string) bool {
	return application.Get().Clipboard.SetText(text)
}

// ReadFromClipboard reads text from the system clipboard.
func (c *ClipboardService) ReadFromClipboard() string {
	text, _ := application.Get().Clipboard.Text()
	return text
}
