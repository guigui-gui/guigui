// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

// Package ebitenginewidget provides a Guigui widget that hosts a separate Ebitengine application.
//
// The [Ebitengine] widget runs a prebuilt Ebitengine binary as a virtualization guest of Ebitengine's
// exp/vmhost package: the guest runs in its own process, and the widget forwards the window's input to
// it, composites its rendered frames into the widget's area, and plays its audio.
//
// The guest binary must be built with the "ebitenginevm" build tag and against the same Ebitengine
// version as the host, since the two speak a version-locked protocol. This package depends on the
// experimental exp/vmhost, whose API may change.
package ebitenginewidget
