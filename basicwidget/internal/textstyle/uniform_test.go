// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textstyle_test

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
	"github.com/guigui-gui/guigui/basicwidget/internal/textstyle"
)

func TestRunsUniformUnderline(t *testing.T) {
	tests := []struct {
		name        string
		ops         func(runs *textstyle.Runs)
		start, end  int
		fallback    bool
		wantValue   bool
		wantUniform bool
	}{
		{
			name:        "no overrides",
			ops:         func(runs *textstyle.Runs) {},
			start:       0,
			end:         10,
			wantValue:   false,
			wantUniform: true,
		},
		{
			name:        "no overrides with a true fallback",
			ops:         func(runs *textstyle.Runs) {},
			start:       0,
			end:         10,
			fallback:    true,
			wantValue:   true,
			wantUniform: true,
		},
		{
			name: "range inside one run",
			ops: func(runs *textstyle.Runs) {
				runs.SetUnderline(0, 10, true)
			},
			start:       2,
			end:         8,
			wantValue:   true,
			wantUniform: true,
		},
		{
			name: "gap before the run",
			ops: func(runs *textstyle.Runs) {
				runs.SetUnderline(5, 10, true)
			},
			start:       0,
			end:         10,
			wantUniform: false,
		},
		{
			name: "gap after the run",
			ops: func(runs *textstyle.Runs) {
				runs.SetUnderline(0, 5, true)
			},
			start:       0,
			end:         10,
			wantUniform: false,
		},
		{
			name: "gap between runs",
			ops: func(runs *textstyle.Runs) {
				runs.SetUnderline(0, 3, true)
				runs.SetUnderline(7, 10, true)
			},
			start:       0,
			end:         10,
			wantUniform: false,
		},
		{
			name: "gap matching the fallback",
			ops: func(runs *textstyle.Runs) {
				runs.SetUnderline(0, 3, true)
				runs.SetUnderline(7, 10, true)
			},
			start:       0,
			end:         10,
			fallback:    true,
			wantValue:   true,
			wantUniform: true,
		},
		{
			name: "adjacent runs with equal values",
			ops: func(runs *textstyle.Runs) {
				runs.SetUnderline(0, 5, true)
				runs.SetColor(0, 5, color.RGBA{R: 0xff, A: 0xff})
				runs.SetUnderline(5, 10, true)
			},
			start:       0,
			end:         10,
			wantValue:   true,
			wantUniform: true,
		},
		{
			name: "adjacent runs with differing values",
			ops: func(runs *textstyle.Runs) {
				runs.SetUnderline(0, 5, true)
				runs.SetUnderline(5, 10, false)
			},
			start:       0,
			end:         10,
			wantUniform: false,
		},
		{
			name: "run without the property counts as fallback",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(0, 5, color.RGBA{R: 0xff, A: 0xff})
				runs.SetUnderline(5, 10, true)
			},
			start:       0,
			end:         10,
			wantUniform: false,
		},
		{
			name: "explicit false override matches a false fallback",
			ops: func(runs *textstyle.Runs) {
				runs.SetUnderline(3, 6, false)
			},
			start:       0,
			end:         10,
			wantValue:   false,
			wantUniform: true,
		},
		{
			name: "empty range",
			ops: func(runs *textstyle.Runs) {
				runs.SetUnderline(0, 10, true)
			},
			start:       5,
			end:         5,
			wantValue:   false,
			wantUniform: true,
		},
		{
			name: "negative start is clamped",
			ops: func(runs *textstyle.Runs) {
				runs.SetUnderline(0, 10, true)
			},
			start:       -3,
			end:         10,
			wantValue:   true,
			wantUniform: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var runs textstyle.Runs
			tt.ops(&runs)
			value, uniform := runs.UniformUnderline(tt.start, tt.end, tt.fallback)
			if uniform != tt.wantUniform {
				t.Fatalf("uniform: got: %t, want: %t", uniform, tt.wantUniform)
			}
			if uniform && value != tt.wantValue {
				t.Errorf("value: got: %t, want: %t", value, tt.wantValue)
			}
		})
	}
}

func TestRunsUniformColor(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}
	blue := color.RGBA{B: 0xff, A: 0xff}

	var runs textstyle.Runs
	runs.SetColor(0, 5, red)

	if clr, uniform := runs.UniformColor(0, 5, nil); !uniform || clr != color.Color(red) {
		t.Errorf("got: %v, %t, want: %v, true", clr, uniform, red)
	}
	if _, uniform := runs.UniformColor(0, 8, nil); uniform {
		t.Error("a partially covered range must not be uniform")
	}
	if clr, uniform := runs.UniformColor(5, 8, nil); !uniform || clr != nil {
		t.Errorf("got: %v, %t, want: nil, true", clr, uniform)
	}

	runs.SetColor(5, 8, blue)
	if _, uniform := runs.UniformColor(0, 8, nil); uniform {
		t.Error("differing colors must not be uniform")
	}
}

func TestRunsUniformVariation(t *testing.T) {
	bold := float32(text.WeightBold)
	medium := float32(text.WeightMedium)

	var runs textstyle.Runs
	runs.SetVariation(0, 5, font.TagWght, bold)
	runs.SetItalic(5, 10, true)

	if v, uniform := runs.UniformVariation(0, 5, font.TagWght, medium); !uniform || v != bold {
		t.Errorf("got: %v, %t, want: %v, true", v, uniform, bold)
	}
	if _, uniform := runs.UniformVariation(0, 10, font.TagWght, medium); uniform {
		t.Error("a partially overridden range with a differing fallback must not be uniform")
	}
	if v, uniform := runs.UniformVariation(0, 10, font.TagWght, bold); !uniform || v != bold {
		t.Errorf("got: %v, %t, want: %v, true", v, uniform, bold)
	}
	if v, uniform := runs.UniformVariation(5, 10, font.TagWght, medium); !uniform || v != medium {
		t.Errorf("got: %v, %t, want: %v, true", v, uniform, medium)
	}
}

func TestRunsUniformScale(t *testing.T) {
	var runs textstyle.Runs
	runs.SetScale(0, 5, 2)

	if scale, uniform := runs.UniformScale(0, 5, 1); !uniform || scale != 2 {
		t.Errorf("got: %v, %t, want: 2, true", scale, uniform)
	}
	if _, uniform := runs.UniformScale(0, 8, 1); uniform {
		t.Error("a partially covered range must not be uniform")
	}
	if scale, uniform := runs.UniformScale(5, 8, 1); !uniform || scale != 1 {
		t.Errorf("got: %v, %t, want: 1, true", scale, uniform)
	}
}

func TestRunsUniformItalic(t *testing.T) {
	var runs textstyle.Runs
	runs.SetItalic(0, 5, true)

	if italic, uniform := runs.UniformItalic(0, 5, false); !uniform || !italic {
		t.Errorf("got: %t, %t, want: true, true", italic, uniform)
	}
	if _, uniform := runs.UniformItalic(0, 8, false); uniform {
		t.Error("a partially covered range must not be uniform")
	}
	if italic, uniform := runs.UniformItalic(0, 8, true); !uniform || !italic {
		t.Errorf("got: %t, %t, want: true, true", italic, uniform)
	}
}

func TestRunsUniformStrikethrough(t *testing.T) {
	var runs textstyle.Runs
	runs.SetStrikethrough(2, 6, true)

	if strikethrough, uniform := runs.UniformStrikethrough(2, 6, false); !uniform || !strikethrough {
		t.Errorf("got: %t, %t, want: true, true", strikethrough, uniform)
	}
	if _, uniform := runs.UniformStrikethrough(0, 6, false); uniform {
		t.Error("a partially covered range must not be uniform")
	}
}
