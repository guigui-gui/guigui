// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func TestBoolPropState(t *testing.T) {
	testCases := []struct {
		name         string
		state        boolPropState
		uniformlyOn  bool
		willToggleOn bool
	}{
		{
			name:         "uniformly on",
			state:        boolPropState{On: true, Uniform: true},
			uniformlyOn:  true,
			willToggleOn: false,
		},
		{
			name:         "uniformly off",
			state:        boolPropState{On: false, Uniform: true},
			uniformlyOn:  false,
			willToggleOn: true,
		},
		{
			name:         "mixed",
			state:        boolPropState{On: true, Uniform: false},
			uniformlyOn:  false,
			willToggleOn: true,
		},
		{
			name:         "zero value",
			state:        boolPropState{},
			uniformlyOn:  false,
			willToggleOn: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.state.UniformlyOn(); got != tc.uniformlyOn {
				t.Errorf("UniformlyOn() = %t; want %t", got, tc.uniformlyOn)
			}
			if got := tc.state.WillToggleOn(); got != tc.willToggleOn {
				t.Errorf("WillToggleOn() = %t; want %t", got, tc.willToggleOn)
			}
		})
	}
}

func TestIsBoldWeight(t *testing.T) {
	testCases := []struct {
		weight text.Weight
		want   bool
	}{
		{weight: text.WeightThin, want: false},
		{weight: text.WeightLight, want: false},
		{weight: text.WeightNormal, want: false},
		{weight: text.WeightMedium, want: false},
		{weight: text.WeightSemibold, want: true},
		{weight: text.WeightBold, want: true},
		{weight: text.WeightBlack, want: true},
	}
	for _, tc := range testCases {
		if got := isBoldWeight(tc.weight); got != tc.want {
			t.Errorf("isBoldWeight(%v) = %t; want %t", tc.weight, got, tc.want)
		}
	}
}

func TestScaleUp(t *testing.T) {
	testCases := []struct {
		scale   float64
		uniform bool
		want    float64
	}{
		{scale: 0.5, uniform: true, want: 0.75},
		{scale: 0.75, uniform: true, want: 1},
		{scale: 1, uniform: true, want: 1.25},
		{scale: 1.1, uniform: true, want: 1.25},
		{scale: 1.25, uniform: true, want: 1.5},
		{scale: 1.5, uniform: true, want: 2},
		{scale: 2, uniform: true, want: 2},
		{scale: 3, uniform: true, want: 2},
		{scale: 2, uniform: false, want: 1.25},
		{scale: 1, uniform: false, want: 1.25},
	}
	for _, tc := range testCases {
		if got := scaleUp(tc.scale, tc.uniform); got != tc.want {
			t.Errorf("scaleUp(%v, %t) = %v; want %v", tc.scale, tc.uniform, got, tc.want)
		}
	}
}

func TestScaleDown(t *testing.T) {
	testCases := []struct {
		scale   float64
		uniform bool
		want    float64
	}{
		{scale: 3, uniform: true, want: 2},
		{scale: 2, uniform: true, want: 1.5},
		{scale: 1.5, uniform: true, want: 1.25},
		{scale: 1.25, uniform: true, want: 1},
		{scale: 1.1, uniform: true, want: 1},
		{scale: 1, uniform: true, want: 0.75},
		{scale: 0.75, uniform: true, want: 0.75},
		{scale: 0.5, uniform: true, want: 0.75},
		{scale: 0.75, uniform: false, want: 0.75},
		{scale: 1, uniform: false, want: 0.75},
	}
	for _, tc := range testCases {
		if got := scaleDown(tc.scale, tc.uniform); got != tc.want {
			t.Errorf("scaleDown(%v, %t) = %v; want %v", tc.scale, tc.uniform, got, tc.want)
		}
	}
}
