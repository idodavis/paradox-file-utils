// newmod.go writes a starter mod folder: descriptor + loc stub + empty common/events.

package game

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// DefaultModParent is Documents/Paradox Interactive/<DocsFolderName>/mod.
func DefaultModParent(gameID string) string {
	info := Get(gameID)
	if info == nil {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(
		home, "Documents", "Paradox Interactive", info.DocsFolderName, "mod",
	)
}

// ModSlug is a filesystem-safe folder name from a display name.
func ModSlug(name string) string {
	var b strings.Builder
	prevUS := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevUS = false
			continue
		}
		if !prevUS {
			b.WriteByte('_')
			prevUS = true
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "mod"
	}
	return out
}

func pinVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || strings.EqualFold(v, "latest") {
		return ""
	}
	return v
}

func locLangOrEnglish(lang string) string {
	lang = strings.TrimSpace(lang)
	if lang == "" {
		return "english"
	}
	return lang
}

func descOrName(description, name string) string {
	description = strings.TrimSpace(description)
	if description == "" {
		return name
	}
	return description
}

// WriteNewMod creates root with a descriptor, empty common/ and events/, loc stub,
// readmes, and an optional thumbnail copied in with picture= on the descriptor.
func WriteNewMod(
	gameID, root, name, supportedVersion, locLang, description, thumbnailSrc string,
) error {
	info := Get(gameID)
	if info == nil {
		return fmt.Errorf("unknown game %s", gameID)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	desc := DescriptorPath(gameID, root)
	if _, err := os.Stat(desc); err == nil {
		return fmt.Errorf("mod already exists: %s", desc)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	picture, err := copyThumbnail(root, thumbnailSrc)
	if err != nil {
		return err
	}
	if err := writeDescriptor(info, root, name, pinVersion(supportedVersion), picture); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, "common"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, "events"), 0o755); err != nil {
		return err
	}
	body := descOrName(description, name)
	if err := writeReadmes(root, name, body); err != nil {
		return err
	}
	lang := locLangOrEnglish(locLang)
	locDir := filepath.Join(root, "localization", lang)
	if err := os.MkdirAll(locDir, 0o755); err != nil {
		return err
	}
	locName := ModSlug(name) + "_l_" + lang + ".yml"
	locBody := append(append([]byte{}, utf8BOM...), []byte("l_"+lang+":\n")...)
	return os.WriteFile(filepath.Join(locDir, locName), locBody, 0o644)
}

func writeReadmes(root, name, description string) error {
	md := "# " + name + "\n\n" + description + "\n"
	bb := "[b]" + name + "[/b]\n\n" + description + "\n"
	if err := os.WriteFile(filepath.Join(root, "readme.md"), []byte(md), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "readme.bbcode"), []byte(bb), 0o644)
}

func copyThumbnail(root, src string) (string, error) {
	src = strings.TrimSpace(src)
	if src == "" {
		return "", nil
	}
	ext := strings.ToLower(filepath.Ext(src))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".svg":
	default:
		return "", fmt.Errorf("thumbnail must be png, jpg, or svg")
	}
	name := "thumbnail" + ext
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.Create(filepath.Join(root, name))
	if err != nil {
		return "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return "", err
	}
	return name, nil
}

func writeDescriptor(info *GameInfo, root, name, supported, picture string) error {
	if info.Descriptor == "metadata" {
		return writeMetadataJSON(root, name, supported, picture)
	}
	return writeDescriptorMod(root, name, supported, picture)
}

func writeDescriptorMod(root, name, supported, picture string) error {
	var b strings.Builder
	b.WriteString("version=\"1.0\"\n")
	fmt.Fprintf(&b, "name=\"%s\"\n", strings.ReplaceAll(name, `"`, `'`))
	if supported != "" {
		fmt.Fprintf(&b, "supported_version=\"%s\"\n", supported)
	}
	if picture != "" {
		fmt.Fprintf(&b, "picture=\"%s\"\n", picture)
	}
	return os.WriteFile(filepath.Join(root, "descriptor.mod"), []byte(b.String()), 0o644)
}

func writeMetadataJSON(root, name, supported, picture string) error {
	meta := map[string]string{
		"name":    name,
		"id":      ModSlug(name),
		"version": "1.0.0",
	}
	if supported != "" {
		meta["supported_game_version"] = supported
	}
	if picture != "" {
		meta["picture"] = picture
	}
	raw, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Join(root, ".metadata")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "metadata.json"), append(raw, '\n'), 0o644)
}
