// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/guigui-gui/guigui/basicwidget"
)

type bigIntValueWidget interface {
	SetMinimumValue(int)
	SetMaximumValue(int)
	SetValueBigInt(*big.Int)
	ValueBigInt() *big.Int
}

func TestNumberWidgetsSetValueBigIntPreservesArgument(t *testing.T) {
	for _, widget := range []struct {
		name      string
		newWidget func() bigIntValueWidget
		force     bool
	}{
		{
			name: "NumberInput/SetValueBigInt",
			newWidget: func() bigIntValueWidget {
				return &basicwidget.NumberInput{}
			},
			force: false,
		},
		{
			name: "NumberInput/ForceSetValueBigInt",
			newWidget: func() bigIntValueWidget {
				return &basicwidget.NumberInput{}
			},
			force: true,
		},
		{
			name: "Slider/SetValueBigInt",
			newWidget: func() bigIntValueWidget {
				return &basicwidget.Slider{}
			},
			force: false,
		},
		{
			name: "ProgressBar/SetValueBigInt",
			newWidget: func() bigIntValueWidget {
				return &basicwidget.ProgressBar{}
			},
			force: false,
		},
	} {
		for _, tc := range []struct {
			name  string
			input int64
			want  int64
		}{
			{
				name:  "upper",
				input: 20,
				want:  10,
			},
			{
				name:  "lower",
				input: -20,
				want:  -10,
			},
			{
				name:  "within",
				input: 5,
				want:  5,
			},
		} {
			t.Run(widget.name+"/"+tc.name, func(t *testing.T) {
				w := widget.newWidget()
				w.SetMinimumValue(-10)
				w.SetMaximumValue(10)
				set := w.SetValueBigInt
				if widget.force {
					set = w.(*basicwidget.NumberInput).ForceSetValueBigInt
				}
				input := big.NewInt(tc.input)
				for range 2 {
					set(input)
					if input.Cmp(big.NewInt(tc.input)) != 0 {
						t.Errorf("argument = %v, want %d", input, tc.input)
					}
					if got := w.ValueBigInt(); got.Cmp(big.NewInt(tc.want)) != 0 {
						t.Errorf("widget value = %v, want %d", got, tc.want)
					}
				}
				input.SetInt64(7)
				if got := w.ValueBigInt(); got.Cmp(big.NewInt(tc.want)) != 0 {
					t.Errorf("widget value after argument mutation = %v, want %d", got, tc.want)
				}
			})
		}
	}
}

func TestAbstractNumberInputBigIntEvents(t *testing.T) {
	for _, force := range []bool{false, true} {
		for _, committed := range []bool{false, true} {
			for _, bound := range []int64{-10, 10} {
				t.Run(fmt.Sprintf("force=%t/committed=%t/bound=%d", force, committed, bound), func(t *testing.T) {
					var a basicwidget.AbstractNumberInput
					a.SetMinimumValue(-10)
					a.SetMaximumValue(10)
					counts := make(map[string]int)
					check := func(kind string, value *big.Int, gotCommitted bool) {
						t.Helper()
						counts[kind]++
						want := big.NewInt(bound)
						if kind == "uint64" && bound < 0 {
							want.SetInt64(0)
						}
						if value.Cmp(want) != 0 || gotCommitted != committed {
							t.Errorf("%s event = (%v, %t), want (%v, %t)", kind, value, gotCommitted, want, committed)
						}
					}
					a.OnValueChanged(func(value int, c bool) { check("int", big.NewInt(int64(value)), c) })
					a.OnValueChangedBigInt(func(value *big.Int, c bool) { check("bigInt", value, c) })
					a.OnValueChangedInt64(func(value int64, c bool) { check("int64", big.NewInt(value), c) })
					a.OnValueChangedUint64(func(value uint64, c bool) { check("uint64", new(big.Int).SetUint64(value), c) })
					a.OnValueChangedString(func(value string, gotForce bool) {
						counts["string"]++
						if value != big.NewInt(bound).String() || gotForce != force {
							t.Errorf("string event = (%s, %t), want (%d, %t)", value, gotForce, bound, force)
						}
					})
					set := a.SetValueBigInt
					if force {
						set = a.ForceSetValueBigInt
					}
					set(big.NewInt(bound*2), committed)
					set(big.NewInt(bound*3), committed)
					for _, kind := range []string{"int", "bigInt", "int64", "uint64", "string"} {
						if got := counts[kind]; got != 1 {
							t.Errorf("%s event count = %d, want 1", kind, got)
						}
					}
				})
			}
		}
	}
}
