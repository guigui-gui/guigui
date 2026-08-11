// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget_test

import (
	"image/color"
	"testing"

	"github.com/guigui-gui/guigui/basicwidget/internal/textstyle"
	"github.com/guigui-gui/guigui/basicwidget/internal/textwidget"
)

// TestTextEnterAtLineEndKeepsTheBreakStyle asserts that splitting a line at
// its end moves the line's own break, and its style, onto the new empty line,
// whose head the caret then reads.
func TestTextEnterAtLineEndKeepsTheBreakStyle(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetEditable(true)
	txt.SetMultiline(true)
	txt.ForceSetValue("Foo\nBar")
	var runs textstyle.Runs
	runs.SetUnderline(0, 3, true)
	runs.SetColor(3, 4, red)
	txt.CopyOverrideStyleRunsFrom(&runs, false)
	txt.SetSelection(3, 3)

	txt.ReplaceTextAt("\n", 3, 3, nil)

	if got := txt.Value(); got != "Foo\n\nBar" {
		t.Fatalf("Value(): got: %q, want: %q", got, "Foo\n\nBar")
	}
	var wantRuns textstyle.Runs
	wantRuns.SetUnderline(0, 4, true)
	wantRuns.SetColor(4, 5, red)
	if got := txt.OverrideStyleRuns(); !equalStyleRuns(got, runsSlice(&wantRuns)) {
		t.Errorf("got: %+v, want: %+v", got, runsSlice(&wantRuns))
	}

	// The caret sits at the head of the new empty line, so it reads that
	// line's own break rather than the byte before it.
	if clr, ok := txt.EffectiveStyleAt(4).Color(); !ok || clr != color.Color(red) {
		t.Errorf("EffectiveStyleAt(4).Color(): got: %v, %t, want: %v, true", clr, ok, red)
	}
	if underline, _ := txt.EffectiveStyleAt(4).Underline(); underline {
		t.Error("EffectiveStyleAt(4).Underline(): got: true, want: false")
	}
}

// TestTextInsertionAtLineHeadAdoptsTheFollowingStyle asserts that text
// inserted at the head of a logical line takes the style of the byte the
// caret was at, not of the break before it.
func TestTextInsertionAtLineHeadAdoptsTheFollowingStyle(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	for _, tc := range []struct {
		name   string
		insert func(txt *textwidget.Text)
	}{
		{
			name: "line break",
			insert: func(txt *textwidget.Text) {
				txt.ReplaceTextAt("\n", 4, 4, nil)
			},
		},
		{
			name: "character",
			insert: func(txt *textwidget.Text) {
				txt.ReplaceTextAt("x", 4, 4, nil)
			},
		},
		{
			name: "IME commit",
			insert: func(txt *textwidget.Text) {
				txt.CommitTextByIME("x")
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// The preceding line and its break are red; the following line is
			// not.
			var txt textwidget.Text
			txt.SetEditable(true)
			txt.SetMultiline(true)
			txt.ForceSetValue("Foo\nBar")
			var runs textstyle.Runs
			runs.SetColor(0, 4, red)
			txt.CopyOverrideStyleRunsFrom(&runs, false)
			txt.SetSelection(4, 4)

			tc.insert(&txt)

			var wantRuns textstyle.Runs
			wantRuns.SetColor(0, 4, red)
			if got := txt.OverrideStyleRuns(); !equalStyleRuns(got, runsSlice(&wantRuns)) {
				t.Errorf("inserting before unstyled text: got: %+v, want: %+v", got, runsSlice(&wantRuns))
			}

			// The following line is red; the preceding one is not.
			var txt2 textwidget.Text
			txt2.SetEditable(true)
			txt2.SetMultiline(true)
			txt2.ForceSetValue("Foo\nBar")
			var runs2 textstyle.Runs
			runs2.SetColor(4, 7, red)
			txt2.CopyOverrideStyleRunsFrom(&runs2, false)
			txt2.SetSelection(4, 4)

			tc.insert(&txt2)

			var wantRuns2 textstyle.Runs
			wantRuns2.SetColor(4, 8, red)
			if got := txt2.OverrideStyleRuns(); !equalStyleRuns(got, runsSlice(&wantRuns2)) {
				t.Errorf("inserting before styled text: got: %+v, want: %+v", got, runsSlice(&wantRuns2))
			}
		})
	}
}

// TestTextCompositionAtLineHeadRendersTheFollowingStyle asserts that an IME
// composition at the head of a logical line renders with the style its text
// carries once committed.
func TestTextCompositionAtLineHeadRendersTheFollowingStyle(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetEditable(true)
	txt.SetMultiline(true)
	txt.ForceSetValue("Foo\nBar")
	var runs textstyle.Runs
	// The preceding line and its break are red; the following line is not.
	runs.SetColor(0, 4, red)
	txt.CopyOverrideStyleRunsFrom(&runs, false)
	txt.HandleFocusChanged(true)
	txt.SetSelection(4, 4)

	const composition = "xy"
	txt.SetCompositionByIME(composition, 0, len(composition))

	var wantRuns textstyle.Runs
	wantRuns.SetColor(0, 4, red)
	if got := txt.RenderingStyleRuns(); !equalStyleRuns(got, runsSlice(&wantRuns)) {
		t.Errorf("while composing: got: %+v, want: %+v", got, runsSlice(&wantRuns))
	}

	// Committing carries the same styles the composition rendered with.
	txt.CommitTextByIME(composition)
	if got := txt.OverrideStyleRuns(); !equalStyleRuns(got, runsSlice(&wantRuns)) {
		t.Errorf("after committing: got: %+v, want: %+v", got, runsSlice(&wantRuns))
	}

	// The following line is red; the preceding one is not.
	var txt2 textwidget.Text
	txt2.SetEditable(true)
	txt2.SetMultiline(true)
	txt2.ForceSetValue("Foo\nBar")
	var runs2 textstyle.Runs
	runs2.SetColor(4, 7, red)
	txt2.CopyOverrideStyleRunsFrom(&runs2, false)
	txt2.HandleFocusChanged(true)
	txt2.SetSelection(4, 4)

	txt2.SetCompositionByIME(composition, 0, len(composition))

	var wantRuns2 textstyle.Runs
	wantRuns2.SetColor(4, 9, red)
	if got := txt2.RenderingStyleRuns(); !equalStyleRuns(got, runsSlice(&wantRuns2)) {
		t.Errorf("composing before styled text: got: %+v, want: %+v", got, runsSlice(&wantRuns2))
	}

	txt2.CommitTextByIME(composition)
	if got := txt2.OverrideStyleRuns(); !equalStyleRuns(got, runsSlice(&wantRuns2)) {
		t.Errorf("after committing before styled text: got: %+v, want: %+v", got, runsSlice(&wantRuns2))
	}
}

// TestTextInsertionAtTheTextEndAdoptsTheStyleBefore asserts that the head of
// the empty last line of a text ending in a line break, which has no byte at
// the caret, falls back to the byte before it.
func TestTextInsertionAtTheTextEndAdoptsTheStyleBefore(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetEditable(true)
	txt.SetMultiline(true)
	txt.ForceSetValue("Heading")
	var runs textstyle.Runs
	runs.SetColor(0, 7, red)
	txt.CopyOverrideStyleRunsFrom(&runs, false)
	txt.SetSelection(7, 7)

	// Enter at the end of the styled line carries the style onto the break.
	txt.ReplaceTextAt("\n", 7, 7, nil)
	var wantRuns textstyle.Runs
	wantRuns.SetColor(0, 8, red)
	if got := txt.OverrideStyleRuns(); !equalStyleRuns(got, runsSlice(&wantRuns)) {
		t.Fatalf("after the break: got: %+v, want: %+v", got, runsSlice(&wantRuns))
	}
	if clr, ok := txt.EffectiveStyleAt(8).Color(); !ok || clr != color.Color(red) {
		t.Errorf("EffectiveStyleAt(8).Color(): got: %v, %t, want: %v, true", clr, ok, red)
	}

	// Typing on that empty last line keeps the style.
	txt.SetSelection(8, 8)
	txt.ReplaceTextAt("x", 8, 8, nil)
	wantRuns.Clear()
	wantRuns.SetColor(0, 9, red)
	if got := txt.OverrideStyleRuns(); !equalStyleRuns(got, runsSlice(&wantRuns)) {
		t.Errorf("after typing: got: %+v, want: %+v", got, runsSlice(&wantRuns))
	}
}

// TestTextReplacementAtLineHeadAdoptsTheFirstReplacedByte asserts that
// replacing a selection that starts at a line head keeps taking the style of
// the first replaced byte.
func TestTextReplacementAtLineHeadAdoptsTheFirstReplacedByte(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetEditable(true)
	txt.SetMultiline(true)
	txt.ForceSetValue("Foo\nBar")
	var runs textstyle.Runs
	runs.SetColor(4, 7, red)
	txt.CopyOverrideStyleRunsFrom(&runs, false)
	txt.SetSelection(4, 7)

	txt.ReplaceTextAt("x", 4, 7, nil)

	if got := txt.Value(); got != "Foo\nx" {
		t.Fatalf("Value(): got: %q, want: %q", got, "Foo\nx")
	}
	var wantRuns textstyle.Runs
	wantRuns.SetColor(4, 5, red)
	if got := txt.OverrideStyleRuns(); !equalStyleRuns(got, runsSlice(&wantRuns)) {
		t.Errorf("got: %+v, want: %+v", got, runsSlice(&wantRuns))
	}
}
