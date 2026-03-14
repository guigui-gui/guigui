// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package calc_test

import (
	"testing"

	"github.com/guigui-gui/guigui/example/calc/internal/calc"
)

func pressButtons(c *calc.Calc, labels ...string) {
	for _, label := range labels {
		idx := calc.ButtonIndex(label)
		c.PressButton(idx)
	}
}

func TestCalc(t *testing.T) {
	tests := []struct {
		name    string
		buttons []string
		want    string
	}{
		{
			name: "initial display",
			want: "0",
		},
		{
			name:    "simple addition",
			buttons: []string{"1", "+", "2", "="},
			want:    "3",
		},
		{
			name:    "simple subtraction",
			buttons: []string{"5", "−", "3", "="},
			want:    "2",
		},
		{
			name:    "simple multiplication",
			buttons: []string{"3", "×", "4", "="},
			want:    "12",
		},
		{
			name:    "simple division",
			buttons: []string{"8", "÷", "2", "="},
			want:    "4",
		},
		{
			name:    "repeated equals",
			buttons: []string{"1", "+", "=", "=", "="},
			want:    "4",
		},
		{
			name:    "repeated equals then new number",
			buttons: []string{"1", "+", "=", "=", "2", "="},
			want:    "2",
		},
		{
			name:    "double operator no double eval",
			buttons: []string{"9", "+", "+"},
			want:    "9",
		},
		{
			name:    "percent then add",
			buttons: []string{"1", "%", "+", "1", "="},
			want:    "1.01",
		},
		{
			name:    "percent then digit starts fresh",
			buttons: []string{"1", "%", "%", "%", "%", "%", "2"},
			want:    "2",
		},
		{
			name:    "division by zero",
			buttons: []string{"1", "÷", "0", "="},
			want:    "Error",
		},
		{
			name:    "clear resets",
			buttons: []string{"5", "+", "3", "=", "C"},
			want:    "0",
		},
		{
			name:    "negate",
			buttons: []string{"5", "±"},
			want:    "-5",
		},
		{
			name:    "negate negative",
			buttons: []string{"5", "±", "±"},
			want:    "5",
		},
		{
			name:    "negate zero",
			buttons: []string{"0", "±"},
			want:    "0",
		},
		{
			name:    "decimal point",
			buttons: []string{"1", ".", "5", "+", "1", ".", "5", "="},
			want:    "3",
		},
		{
			name:    "chained operations",
			buttons: []string{"2", "+", "3", "+", "4", "="},
			want:    "9",
		},
		{
			name:    "change operator",
			buttons: []string{"5", "+", "−", "3", "="},
			want:    "2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &calc.Calc{}
			pressButtons(c, tt.buttons...)
			if c.Display() != tt.want {
				t.Errorf("buttons %v: got %q, want %q", tt.buttons, c.Display(), tt.want)
			}
		})
	}
}
