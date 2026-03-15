// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package calc

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ButtonLabel represents a calculator button label.
type ButtonLabel string

const (
	ButtonLabel0        ButtonLabel = "0"
	ButtonLabel1        ButtonLabel = "1"
	ButtonLabel2        ButtonLabel = "2"
	ButtonLabel3        ButtonLabel = "3"
	ButtonLabel4        ButtonLabel = "4"
	ButtonLabel5        ButtonLabel = "5"
	ButtonLabel6        ButtonLabel = "6"
	ButtonLabel7        ButtonLabel = "7"
	ButtonLabel8        ButtonLabel = "8"
	ButtonLabel9        ButtonLabel = "9"
	ButtonLabelDot      ButtonLabel = "."
	ButtonLabelAdd      ButtonLabel = "+"
	ButtonLabelSubtract ButtonLabel = "−"
	ButtonLabelMultiply ButtonLabel = "×"
	ButtonLabelDivide   ButtonLabel = "÷"
	ButtonLabelPercent  ButtonLabel = "%"
	ButtonLabelEquals   ButtonLabel = "="
	ButtonLabelNegate   ButtonLabel = "±"
	ButtonLabelClear    ButtonLabel = "C"
)

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

// PressButton processes a button press with the given label.
func (c *Calc) PressButton(label ButtonLabel) {
	if c.display == "" {
		c.display = "0"
	}
	switch label {
	case ButtonLabel0, ButtonLabel1, ButtonLabel2, ButtonLabel3, ButtonLabel4,
		ButtonLabel5, ButtonLabel6, ButtonLabel7, ButtonLabel8, ButtonLabel9:
		c.lastOperator = operatorNone
		if c.newOperand {
			c.display = string(label)
			c.newOperand = false
		} else if c.display == "0" {
			c.display = string(label)
		} else {
			c.display += string(label)
		}
	case ButtonLabelDot:
		if c.newOperand {
			c.display = "0."
			c.newOperand = false
		} else if !strings.Contains(c.display, ".") {
			c.display += "."
		}
	case ButtonLabelAdd:
		c.applyOperator(operatorAdd)
	case ButtonLabelSubtract:
		c.applyOperator(operatorSubtract)
	case ButtonLabelMultiply:
		c.applyOperator(operatorMultiply)
	case ButtonLabelDivide:
		c.applyOperator(operatorDivide)
	case ButtonLabelEquals:
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
	case ButtonLabelClear:
		c.display = "0"
		c.operand = 0
		c.operator = operatorNone
		c.lastOperand = 0
		c.lastOperator = operatorNone
		c.newOperand = false
	case ButtonLabelNegate:
		if c.display != "0" {
			if strings.HasPrefix(c.display, "-") {
				c.display = c.display[1:]
			} else {
				c.display = "-" + c.display
			}
		}
	case ButtonLabelPercent:
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
