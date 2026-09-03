//go:build !windows

// dlopen_unix.go loads steam_api via dlopen on non-Windows hosts.

package steamugc

import "github.com/ebitengine/purego"

func openLibHandle(path string) (uintptr, func(), error) {
	h, err := purego.Dlopen(path, purego.RTLD_NOW)
	if err != nil {
		return 0, nil, err
	}
	return h, func() { _ = purego.Dlclose(h) }, nil
}

func lookupSym(h uintptr, name string) (uintptr, error) {
	return purego.Dlsym(h, name)
}
