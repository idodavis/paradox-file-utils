// Package steamugc talks to ISteamUGC via Steamworks SDK 1.65 flats (purego, no CGO).
package steamugc

import (
	"fmt"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
)

const (
	ugcCreateItemCB     = 3403
	ugcSubmitCB         = 3404
	eResultOK           = 1
	eResultFileNotFound = 9
	fileTypeFirst       = 0
	visPrivate          = 2
)

type lib struct {
	close           func()
	initFlat        func(*[1024]byte) int32
	shutdown        func()
	runCallbacks    func()
	ugc             func() uintptr
	utils           func() uintptr
	createItem      func(uintptr, uint32, int32) uint64
	startUpdate     func(uintptr, uint32, uint64) uint64
	setTitle        func(uintptr, uint64, *byte) bool
	setDesc         func(uintptr, uint64, *byte) bool
	setVis          func(uintptr, uint64, int32) bool
	setContent      func(uintptr, uint64, *byte) bool
	setPreview      func(uintptr, uint64, *byte) bool
	addPreviewFile  func(uintptr, uint64, *byte, int32) bool
	addPreviewVideo func(uintptr, uint64, *byte) bool
	removePreview   func(uintptr, uint64, uint32) bool
	submit          func(uintptr, uint64, *byte) uint64
	callDone        func(uintptr, uint64, *byte) bool
	callResult      func(uintptr, uint64, unsafe.Pointer, int32, int32, *byte) bool
}

func openLib(path string) (*lib, error) {
	h, closeFn, err := openLibHandle(path)
	if err != nil {
		return nil, fmt.Errorf("load steam api: %w", err)
	}
	l := &lib{close: closeFn}
	for _, b := range []struct {
		fn   any
		name string
	}{
		{&l.initFlat, "SteamAPI_InitFlat"},
		{&l.shutdown, "SteamAPI_Shutdown"},
		{&l.runCallbacks, "SteamAPI_RunCallbacks"},
		{&l.ugc, "SteamAPI_SteamUGC_v021"},
		{&l.utils, "SteamAPI_SteamUtils_v011"},
		{&l.createItem, "SteamAPI_ISteamUGC_CreateItem"},
		{&l.startUpdate, "SteamAPI_ISteamUGC_StartItemUpdate"},
		{&l.setTitle, "SteamAPI_ISteamUGC_SetItemTitle"},
		{&l.setDesc, "SteamAPI_ISteamUGC_SetItemDescription"},
		{&l.setVis, "SteamAPI_ISteamUGC_SetItemVisibility"},
		{&l.setContent, "SteamAPI_ISteamUGC_SetItemContent"},
		{&l.setPreview, "SteamAPI_ISteamUGC_SetItemPreview"},
		{&l.addPreviewFile, "SteamAPI_ISteamUGC_AddItemPreviewFile"},
		{&l.addPreviewVideo, "SteamAPI_ISteamUGC_AddItemPreviewVideo"},
		{&l.removePreview, "SteamAPI_ISteamUGC_RemoveItemPreview"},
		{&l.submit, "SteamAPI_ISteamUGC_SubmitItemUpdate"},
		{&l.callDone, "SteamAPI_ISteamUtils_IsAPICallCompleted"},
		{&l.callResult, "SteamAPI_ISteamUtils_GetAPICallResult"},
	} {
		if err := register(b.fn, h, b.name); err != nil {
			closeFn()
			return nil, err
		}
	}
	return l, nil
}

func register(fptr any, h uintptr, name string) error {
	addr, err := lookupSym(h, name)
	if err != nil || addr == 0 {
		return fmt.Errorf("missing %s", name)
	}
	purego.RegisterFunc(fptr, addr)
	return nil
}

func (l *lib) init() error {
	var msg [1024]byte
	if l.initFlat(&msg) != 0 {
		n := 0
		for n < len(msg) && msg[n] != 0 {
			n++
		}
		return fmt.Errorf("SteamAPI_InitFlat: %s", string(msg[:n]))
	}
	return nil
}

func (l *lib) waitCall(call uint64, dest unsafe.Pointer, size, cb int32) error {
	if call == 0 {
		return fmt.Errorf("steam api call failed to start")
	}
	utils := l.utils()
	if utils == 0 {
		return fmt.Errorf("SteamAPI_SteamUtils_v011 returned nil")
	}
	deadline := time.Now().Add(10 * time.Minute)
	var failed byte
	for time.Now().Before(deadline) {
		l.runCallbacks()
		if l.callDone(utils, call, &failed) {
			if failed != 0 {
				return fmt.Errorf("steam api call failed")
			}
			if !l.callResult(utils, call, dest, size, cb, &failed) || failed != 0 {
				return fmt.Errorf("steam api call result failed")
			}
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("steam api call timed out")
}

func cstr(s string) *byte {
	b := append([]byte(s), 0)
	return &b[0]
}
