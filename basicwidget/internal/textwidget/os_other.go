// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

//go:build !darwin && !js

package textwidget

// IsDarwin reports whether the app runs on a Darwin-based platform, where
// text-editing key bindings follow the macOS conventions.
func IsDarwin() bool {
	return false
}
