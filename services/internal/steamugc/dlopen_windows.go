//go:build windows

// dlopen_windows.go loads steam_api64.dll via LoadLibrary.

package steamugc

import "syscall"

func openLibHandle(path string) (uintptr, func(), error) {
	h, err := syscall.LoadLibrary(path)
	if err != nil {
		return 0, nil, err
	}
	return uintptr(h), func() { _ = syscall.FreeLibrary(h) }, nil
}

func lookupSym(h uintptr, name string) (uintptr, error) {
	return syscall.GetProcAddress(syscall.Handle(h), name)
}
