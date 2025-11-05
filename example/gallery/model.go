// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"fmt"
	"iter"
	"math/big"
	"slices"

	"github.com/guigui-gui/guigui/basicwidget"
)

type Model struct {
	mode string

	buttons      ButtonsModel
	texts        TextsModel
	textInputs   TextInputsModel
	numberInputs NumberInputsModel
	lists        ListsModel
	tables       TablesModel
}

func (m *Model) Mode() string {
	if m.mode == "" {
		return "settings"
	}
	return m.mode
}

func (m *Model) SetMode(mode string) {
	m.mode = mode
}

func (m *Model) Buttons() *ButtonsModel {
	return &m.buttons
}

func (m *Model) Texts() *TextsModel {
	return &m.texts
}

func (m *Model) TextInputs() *TextInputsModel {
	return &m.textInputs
}

func (m *Model) NumberInputs() *NumberInputsModel {
	return &m.numberInputs
}

func (m *Model) Lists() *ListsModel {
	return &m.lists
}

func (m *Model) Tables() *TablesModel {
	return &m.tables
}

type ButtonsModel struct {
	disabled bool
}

func (b *ButtonsModel) Enabled() bool {
	return !b.disabled
}

func (b *ButtonsModel) SetEnabled(enabled bool) {
	b.disabled = !enabled
}

type TextsModel struct {
	text    string
	textSet bool

	horizontalAlign basicwidget.HorizontalAlign
	verticalAlign   basicwidget.VerticalAlign
	noWrap          bool
	bold            bool
	selectable      bool
	editable        bool
}

func (t *TextsModel) Text() string {
	if !t.textSet {
		return `Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.
隴西の李徴は博学才穎、天宝の末年、若くして名を虎榜に連ね、ついで江南尉に補せられたが、性、狷介、自ら恃むところ頗る厚く、賤吏に甘んずるを潔しとしなかった。`
	}
	return t.text
}

func (t *TextsModel) SetText(text string) {
	t.text = text
	t.textSet = true
}

func (t *TextsModel) HorizontalAlign() basicwidget.HorizontalAlign {
	return t.horizontalAlign
}

func (t *TextsModel) SetHorizontalAlign(align basicwidget.HorizontalAlign) {
	t.horizontalAlign = align
}

func (t *TextsModel) VerticalAlign() basicwidget.VerticalAlign {
	return t.verticalAlign
}

func (t *TextsModel) SetVerticalAlign(align basicwidget.VerticalAlign) {
	t.verticalAlign = align
}

func (t *TextsModel) AutoWrap() bool {
	return !t.noWrap
}

func (t *TextsModel) SetAutoWrap(autoWrap bool) {
	t.noWrap = !autoWrap
}

func (t *TextsModel) Bold() bool {
	return t.bold
}

func (t *TextsModel) SetBold(bold bool) {
	t.bold = bold
}

func (t *TextsModel) Selectable() bool {
	return t.selectable
}

func (t *TextsModel) SetSelectable(selectable bool) {
	t.selectable = selectable
	if !selectable {
		t.editable = false
	}
}

func (t *TextsModel) Editable() bool {
	return t.editable
}

func (t *TextsModel) SetEditable(editable bool) {
	t.editable = editable
	if editable {
		t.selectable = true
	}
}

type TextInputsModel struct {
	singleLineText     string
	singleLinetTextSet bool
	multilineText      string
	multilineTextSet   bool

	horizontalAlign basicwidget.HorizontalAlign
	verticalAlign   basicwidget.VerticalAlign
	noWrap          bool
	uneditable      bool
	disabled        bool
}

func (t *TextInputsModel) SingleLineText() string {
	if !t.singleLinetTextSet {
		return "Hello, Guigui!"
	}
	return t.singleLineText
}

func (t *TextInputsModel) SetSingleLineText(text string) {
	t.singleLineText = text
	t.singleLinetTextSet = true
}

func (t *TextInputsModel) MultilineText() string {
	if !t.multilineTextSet {
		return "Hello, Guigui!\nThis is a multiline text field."
	}
	return t.multilineText
}

func (t *TextInputsModel) SetMultilineText(text string) {
	t.multilineText = text
	t.multilineTextSet = true
}

func (t *TextInputsModel) HorizontalAlign() basicwidget.HorizontalAlign {
	return t.horizontalAlign
}

func (t *TextInputsModel) SetHorizontalAlign(align basicwidget.HorizontalAlign) {
	t.horizontalAlign = align
}

func (t *TextInputsModel) VerticalAlign() basicwidget.VerticalAlign {
	return t.verticalAlign
}

func (t *TextInputsModel) SetVerticalAlign(align basicwidget.VerticalAlign) {
	t.verticalAlign = align
}

func (t *TextInputsModel) AutoWrap() bool {
	return !t.noWrap
}

func (t *TextInputsModel) SetAutoWrap(autoWrap bool) {
	t.noWrap = !autoWrap
}

func (t *TextInputsModel) Editable() bool {
	return !t.uneditable
}

func (t *TextInputsModel) SetEditable(editable bool) {
	t.uneditable = !editable
	if editable {
		t.disabled = false
	}
}

func (t *TextInputsModel) Enabled() bool {
	return !t.disabled
}

func (t *TextInputsModel) SetEnabled(enabled bool) {
	t.disabled = !enabled
	if !enabled {
		t.uneditable = true
	}
}

type NumberInputsModel struct {
	numberInputValue1 big.Int
	numberInputValue2 uint64
	numberInputValue3 int

	uneditable bool
	disabled   bool
}

func (n *NumberInputsModel) Editable() bool {
	return !n.uneditable
}

func (n *NumberInputsModel) SetEditable(editable bool) {
	n.uneditable = !editable
	if editable {
		n.disabled = false
	}
}

func (n *NumberInputsModel) Enabled() bool {
	return !n.disabled
}

func (n *NumberInputsModel) SetEnabled(enabled bool) {
	n.disabled = !enabled
	if !enabled {
		n.uneditable = true
	}
}

func (n *NumberInputsModel) NumberInputValue1() *big.Int {
	var v big.Int
	v.Set(&n.numberInputValue1)
	return &v
}

func (n *NumberInputsModel) SetNumberInputValue1(value *big.Int) {
	n.numberInputValue1.Set(value)
}

func (n *NumberInputsModel) NumberInputValue2() uint64 {
	return n.numberInputValue2
}

func (n *NumberInputsModel) SetNumberInputValue2(value uint64) {
	n.numberInputValue2 = value
}

func (n *NumberInputsModel) NumberInputValue3() int {
	return n.numberInputValue3
}

func (n *NumberInputsModel) SetNumberInputValue3(value int) {
	n.numberInputValue3 = value
}

type ListsModel struct {
	listItems         []basicwidget.ListItem[int]
	treeItems         []basicwidget.ListItem[int]
	dropdownListItems []basicwidget.DropdownListItem[int]

	stripeVisible bool
	headerVisible bool
	footerVisible bool
	unmovable     bool
	disabled      bool
}

func (l *ListsModel) ensureListItems() {
	if l.listItems != nil {
		return
	}
	for i := range 99 {
		l.listItems = append(l.listItems, basicwidget.ListItem[int]{
			Text: fmt.Sprintf("Item %d", i+1),
		})
	}
}

func (l *ListsModel) AppendListItems(items []basicwidget.ListItem[int]) []basicwidget.ListItem[int] {
	l.ensureListItems()
	for i := range l.listItems {
		l.listItems[i].Movable = !l.unmovable
	}
	return append(items, l.listItems...)
}

func (l *ListsModel) ListItemCount() int {
	l.ensureListItems()
	return len(l.listItems)
}

func (l *ListsModel) ensureTreeItems() {
	if l.treeItems != nil {
		return
	}
	l.treeItems = []basicwidget.ListItem[int]{
		{Text: "Item 1", Value: 1, IndentLevel: 1},
		{Text: "Item 2", Value: 2, IndentLevel: 1},
		{Text: "Item 3", Value: 3, IndentLevel: 2},
		{Text: "Item 4", Value: 4, IndentLevel: 2},
		{Text: "Item 5", Value: 5, IndentLevel: 3},
		{Text: "Item 6", Value: 6, IndentLevel: 3},
		{Text: "Item 7", Value: 7, IndentLevel: 1},
		{Text: "Item 8", Value: 8, IndentLevel: 2},
		{Text: "Item 9", Value: 9, IndentLevel: 2},
		{Text: "Item 10", Value: 10, IndentLevel: 3},
		{Text: "Item 11", Value: 11, IndentLevel: 3},
		{Text: "Item 12", Value: 12, IndentLevel: 1},
	}
}

func (l *ListsModel) AppendTreeItems(items []basicwidget.ListItem[int]) []basicwidget.ListItem[int] {
	l.ensureTreeItems()
	// TODO: Enable to move items.
	return append(items, l.treeItems...)
}

func (l *ListsModel) AppendDropdownListItems(items []basicwidget.DropdownListItem[int]) []basicwidget.DropdownListItem[int] {
	if l.dropdownListItems == nil {
		for i := range 9 {
			l.dropdownListItems = append(l.dropdownListItems, basicwidget.DropdownListItem[int]{
				Text:  fmt.Sprintf("Item %d", i+1),
				Value: i + 1,
			})
		}
	}
	return append(items, l.dropdownListItems...)
}

func (l *ListsModel) MoveListItems(from int, count int, to int) int {
	return basicwidget.MoveItemsInSlice(l.listItems, from, count, to)
}

func (l *ListsModel) SetTreeItemExpanded(index int, expanded bool) {
	l.ensureTreeItems()
	if index < 0 || index >= len(l.treeItems) {
		return
	}
	l.treeItems[index].Collapsed = !expanded
}

func (l *ListsModel) IsStripeVisible() bool {
	return l.stripeVisible
}

func (l *ListsModel) SetStripeVisible(visible bool) {
	l.stripeVisible = visible
}

func (l *ListsModel) IsHeaderVisible() bool {
	return l.headerVisible
}

func (l *ListsModel) SetHeaderVisible(hasHeader bool) {
	l.headerVisible = hasHeader
}

func (l *ListsModel) IsFooterVisible() bool {
	return l.footerVisible
}

func (l *ListsModel) SetFooterVisible(hasFooter bool) {
	l.footerVisible = hasFooter
}

func (l *ListsModel) Movable() bool {
	return !l.unmovable
}

func (l *ListsModel) SetMovable(movable bool) {
	l.unmovable = !movable
}

func (l *ListsModel) Enabled() bool {
	return !l.disabled
}

func (l *ListsModel) SetEnabled(enabled bool) {
	l.disabled = !enabled
}

type TableItem struct {
	ID     int
	Name   string
	Amount int
	Cost   int
}

type TablesModel struct {
	tableItems []TableItem

	footerVisible bool
	unmovable     bool
	disabled      bool
}

func (t *TablesModel) ensureTableItems() {
	if t.tableItems != nil {
		return
	}
	t.tableItems = []TableItem{
		{ID: 1, Name: "Apple", Amount: 3, Cost: 120},
		{ID: 2, Name: "Banana", Amount: 6, Cost: 50},
		{ID: 3, Name: "Cherry", Amount: 15, Cost: 200},
		{ID: 4, Name: "Grape", Amount: 10, Cost: 175},
		{ID: 5, Name: "Mango", Amount: 2, Cost: 250},
		{ID: 6, Name: "Orange", Amount: 4, Cost: 110},
		{ID: 7, Name: "Kiwi", Amount: 5, Cost: 160},
		{ID: 8, Name: "Peach", Amount: 3, Cost: 180},
		{ID: 9, Name: "Lemon", Amount: 7, Cost: 90},
		{ID: 10, Name: "Pineapple", Amount: 1, Cost: 300},
	}
}

func (t *TablesModel) TableItemCount() int {
	t.ensureTableItems()
	return len(t.tableItems)
}

func (t *TablesModel) TableItems() iter.Seq2[int, TableItem] {
	t.ensureTableItems()
	return slices.All(t.tableItems)
}

func (t *TablesModel) MoveTableItems(from int, count int, to int) int {
	t.ensureTableItems()
	return basicwidget.MoveItemsInSlice(t.tableItems, from, count, to)
}

func (t *TablesModel) IsFooterVisible() bool {
	return t.footerVisible
}

func (t *TablesModel) SetFooterVisible(hasFooter bool) {
	t.footerVisible = hasFooter
}

func (t *TablesModel) Movable() bool {
	return !t.unmovable
}

func (t *TablesModel) SetMovable(movable bool) {
	t.unmovable = !movable
}

func (t *TablesModel) Enabled() bool {
	return !t.disabled
}

func (t *TablesModel) SetEnabled(enabled bool) {
	t.disabled = !enabled
}
