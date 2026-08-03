// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

// Package guigui provides a GUI framework for Go built on top of Ebitengine.
//
// # Widget lifecycle
//
// The core of guigui is the [Widget] interface. All UI components implement this interface
// by embedding [DefaultWidget] in their structs.
//
// The framework guarantees the following about a few of the widget methods:
//
//   - [Widget.Build] constructs the child widget tree. When Build is called, neither the widget's
//     own bounds nor its parent's bounds are determined yet (so [WidgetBounds] is not passed as
//     an argument), and the child tree is not determined yet either.
//   - [Widget.Layout] positions and sizes children. When Layout is called, the widget's own bounds
//     and its parent's bounds are determined, but its children's bounds are not yet determined.
//   - [Widget.Tick] is invoked at the application's TPS (60 times per second by default,
//     or whatever TPS the user has configured).
//   - [Widget.HandlePointingInput] is invoked in post-order (children before their parent),
//     per layer from top to bottom. This lets an inner or higher-layer widget consume a pointing
//     event before its ancestors or lower layers see it.
//   - [Widget.HandleButtonInput] is invoked with the same post-order, top-to-bottom-layer
//     traversal, but only on a subset of widgets: roughly, a widget that is focused, has a focused
//     ancestor or descendant, or is itself button-input-receptive
//     (see [Context.SetButtonInputReceptive]). Disabled or hidden widgets are skipped.
//     See [Widget.HandleButtonInput] for the exact conditions.
//   - [Widget.Draw] is invoked in pre-order (parent before its children), per layer from bottom
//     to top, so children and higher layers are rendered on top of their parents and lower layers.
//
// All other aspects — such as when and how often each method is called — are implementation
// details that the framework may change.
//
// # Running an application
//
// Use [Run] to start an application with a root widget:
//
//	type Root struct {
//		guigui.DefaultWidget
//		// ...
//	}
//
//	func main() {
//		if err := guigui.Run(&Root{}, &guigui.RunOptions{
//			Title: "My App",
//		}); err != nil {
//			log.Fatal(err)
//		}
//	}
//
// # Environment variables
//
// The environment variable GUIGUI_COLOR_MODE specifies the preferred color mode. Its value is
// either light or dark. [Context.SetPreferredColorMode] overrides it.
//
// The environment variable GUIGUI_LOCALES specifies the locales. Its value is a comma-separated
// list of BCP 47 language tags. The effective locales are determined by the app locales,
// GUIGUI_LOCALES, and the system locales, in that priority order.
//
// # Debugging
//
// The environment variable GUIGUI_DEBUG enables debugging features. Its value is a
// comma-separated list of these options:
//
//   - showrenderingregions: visualizes the regions that are redrawn.
//   - showbuildlogs: logs why the widget tree is rebuilt.
//   - showinputlogs: logs which widget handled an input.
//   - devicescale=<float>: uses the given device scale factor instead of the monitor's.
//   - emulateclipboard: replaces the system clipboard with an in-process one, so that copying
//     and pasting leave the system clipboard untouched. The environment variable
//     GUIGUI_DEBUG_CLIPBOARD_TEXT gives the emulated clipboard its initial text.
//
// These options are for debugging and can change at any time.
package guigui
