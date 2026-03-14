// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package calc

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
)

type buttonLabel string

const (
	buttonLabel0        buttonLabel = "0"
	buttonLabel1        buttonLabel = "1"
	buttonLabel2        buttonLabel = "2"
	buttonLabel3        buttonLabel = "3"
	buttonLabel4        buttonLabel = "4"
	buttonLabel5        buttonLabel = "5"
	buttonLabel6        buttonLabel = "6"
	buttonLabel7        buttonLabel = "7"
	buttonLabel8        buttonLabel = "8"
	buttonLabel9        buttonLabel = "9"
	buttonLabelDot      buttonLabel = "."
	buttonLabelAdd      buttonLabel = "+"
	buttonLabelSubtract buttonLabel = "−"
	buttonLabelMultiply buttonLabel = "×"
	buttonLabelDivide   buttonLabel = "÷"
	buttonLabelPercent  buttonLabel = "%"
	buttonLabelEquals   buttonLabel = "="
	buttonLabelNegate   buttonLabel = "±"
	buttonLabelClear    buttonLabel = "C"
)

var buttonLabels = [...]buttonLabel{
	buttonLabelClear, buttonLabelNegate, buttonLabelPercent, buttonLabelDivide,
	buttonLabel7, buttonLabel8, buttonLabel9, buttonLabelMultiply,
	buttonLabel4, buttonLabel5, buttonLabel6, buttonLabelSubtract,
	buttonLabel1, buttonLabel2, buttonLabel3, buttonLabelAdd,
	buttonLabel0, buttonLabelDot, buttonLabelEquals,
}

// ButtonCount is the number of buttons.
const ButtonCount = len(buttonLabels)

// ButtonLabelAt returns the label string for the button at the given index.
func ButtonLabelAt(idx int) string {
	return string(buttonLabels[idx])
}

// ButtonIndex returns the index of the button with the given label string.
func ButtonIndex(label string) int {
	return slices.IndexFunc(buttonLabels[:], func(l buttonLabel) bool {
		return string(l) == label
	})
}

type operator int

const (
	operatorNone operator = iota
	operatorAdd
	operatorSubtract
	operatorMultiply
	operatorDivide
)

type Calc struct {
	display      string
	operand      float64
	operator     operator
	lastOperand  float64
	lastOperator operator
	newOperand   bool
}

func (c *Calc) Display() string {
	if c.display == "" {
		return "0"
	}
	return c.display
}

func (c *Calc) PressButton(idx int) {
	if c.display == "" {
		c.display = "0"
	}
	label := buttonLabels[idx]
	switch label {
	case buttonLabel0, buttonLabel1, buttonLabel2, buttonLabel3, buttonLabel4,
		buttonLabel5, buttonLabel6, buttonLabel7, buttonLabel8, buttonLabel9:
		c.lastOperator = operatorNone
		if c.newOperand {
			c.display = string(label)
			c.newOperand = false
		} else if c.display == "0" {
			c.display = string(label)
		} else {
			c.display += string(label)
		}
	case buttonLabelDot:
		if c.newOperand {
			c.display = "0."
			c.newOperand = false
		} else if !strings.Contains(c.display, ".") {
			c.display += "."
		}
	case buttonLabelAdd:
		c.applyOperator(operatorAdd)
	case buttonLabelSubtract:
		c.applyOperator(operatorSubtract)
	case buttonLabelMultiply:
		c.applyOperator(operatorMultiply)
	case buttonLabelDivide:
		c.applyOperator(operatorDivide)
	case buttonLabelEquals:
		if c.operator != operatorNone {
			current, err := strconv.ParseFloat(c.display, 64)
			if err == nil {
				c.lastOperand = current
				c.lastOperator = c.operator
			}
			c.evaluate()
			c.operator = operatorNone
		} else if c.lastOperator != operatorNone {
			c.operand, _ = strconv.ParseFloat(c.display, 64)
			c.operator = c.lastOperator
			c.display = formatNumber(c.lastOperand)
			c.evaluate()
			c.operator = operatorNone
		}
		c.newOperand = true
	case buttonLabelClear:
		c.display = "0"
		c.operand = 0
		c.operator = operatorNone
		c.lastOperand = 0
		c.lastOperator = operatorNone
		c.newOperand = false
	case buttonLabelNegate:
		if c.display != "0" {
			if strings.HasPrefix(c.display, "-") {
				c.display = c.display[1:]
			} else {
				c.display = "-" + c.display
			}
		}
	case buttonLabelPercent:
		val, err := strconv.ParseFloat(c.display, 64)
		if err == nil {
			c.display = formatNumber(val / 100)
			c.newOperand = true
		}
	}
}

func (c *Calc) applyOperator(op operator) {
	if !c.newOperand {
		c.evaluate()
	} else {
		// Capture the current display as the operand (e.g. after %).
		val, err := strconv.ParseFloat(c.display, 64)
		if err == nil {
			c.operand = val
		}
	}
	c.operator = op
	c.newOperand = true
}

func (c *Calc) evaluate() {
	current, err := strconv.ParseFloat(c.display, 64)
	if err != nil {
		return
	}
	if c.operator == operatorNone {
		c.operand = current
		return
	}

	var result float64
	switch c.operator {
	case operatorAdd:
		result = c.operand + current
	case operatorSubtract:
		result = c.operand - current
	case operatorMultiply:
		result = c.operand * current
	case operatorDivide:
		result = c.operand / current
	}
	if math.IsInf(result, 0) || math.IsNaN(result) {
		c.display = "Error"
		c.operand = 0
		c.operator = operatorNone
		c.newOperand = true
		return
	}
	c.display = formatNumber(result)
	c.operand = result
}

func formatNumber(f float64) string {
	if f == float64(int64(f)) {
		return fmt.Sprintf("%d", int64(f))
	}
	s := fmt.Sprintf("%.10g", f)
	return s
}
