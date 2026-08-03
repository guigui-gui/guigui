// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package debugmode

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type debugMode struct {
	showRenderingRegions bool
	showBuildLogs        bool
	showInputLogs        bool
	deviceScale          float64

	// emulateClipboard replaces the system clipboard with an in-process one.
	emulateClipboard bool

	// emulatedClipboardText is the initial text of the emulated clipboard.
	emulatedClipboardText string
}

var theDebugMode debugMode

func init() {
	for token := range strings.SplitSeq(os.Getenv("GUIGUI_DEBUG"), ",") {
		switch {
		case token == "showrenderingregions":
			theDebugMode.showRenderingRegions = true
		case token == "showbuildlogs":
			theDebugMode.showBuildLogs = true
		case token == "showinputlogs":
			theDebugMode.showInputLogs = true
		case token == "emulateclipboard":
			theDebugMode.emulateClipboard = true
		case strings.HasPrefix(token, "devicescale="):
			f, err := strconv.ParseFloat(token[len("devicescale="):], 64)
			if err != nil {
				slog.Error(err.Error())
			}
			theDebugMode.deviceScale = f
		case token == "":
		default:
			slog.Warn("unknown debug option", "option", token)
		}
	}

	if theDebugMode.emulateClipboard {
		theDebugMode.emulatedClipboardText = os.Getenv("GUIGUI_DEBUG_CLIPBOARD_TEXT")
	}
}

// ShowRenderingRegions reports whether redrawn regions should be visualized.
func ShowRenderingRegions() bool {
	return theDebugMode.showRenderingRegions
}

// ShowBuildLogs reports whether widget tree rebuilds should be logged.
func ShowBuildLogs() bool {
	return theDebugMode.showBuildLogs
}

// ShowInputLogs reports whether input handling should be logged.
func ShowInputLogs() bool {
	return theDebugMode.showInputLogs
}

// DeviceScale returns the device scale factor to use instead of the monitor's,
// or 0 when the monitor's factor should be used.
func DeviceScale() float64 {
	return theDebugMode.deviceScale
}

// EmulatesClipboard reports whether the system clipboard is replaced with an
// in-process one.
func EmulatesClipboard() bool {
	return theDebugMode.emulateClipboard
}

// EmulatedClipboardText returns the initial text of the emulated clipboard.
// The result is meaningful only when [EmulatesClipboard] reports true.
func EmulatedClipboardText() string {
	return theDebugMode.emulatedClipboardText
}
