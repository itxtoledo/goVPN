package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// TappableContainer is a container that accepts tap events
type TappableContainer struct {
	widget.BaseWidget
	content        fyne.CanvasObject
	onTap          func()
	onTapSecondary func(pe *fyne.PointEvent)
}

// NewTappableContainer creates a new tappable container
func NewTappableContainer(content fyne.CanvasObject, onTap func(), onTapSecondary func(pe *fyne.PointEvent)) *TappableContainer {
	tc := &TappableContainer{
		content:        content,
		onTap:          onTap,
		onTapSecondary: onTapSecondary,
	}
	tc.ExtendBaseWidget(tc)
	return tc
}

// CreateRenderer implements the WidgetRenderer
func (tc *TappableContainer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(tc.content)
}

// Tapped implements the tap event
func (tc *TappableContainer) Tapped(pe *fyne.PointEvent) {
	if tc.onTap != nil {
		tc.onTap()
	}
}

// TappedSecondary implements the secondary tap event (right click)
func (tc *TappableContainer) TappedSecondary(pe *fyne.PointEvent) {
	if tc.onTapSecondary != nil {
		tc.onTapSecondary(pe)
	}
}

// CustomAccordionItem represents a custom accordion item that accepts any widget as title
type CustomAccordionItem struct {
	Title          fyne.CanvasObject // Can be any widget
	Content        fyne.CanvasObject
	EndContent     fyne.CanvasObject // Content that stays at the end of the line (e.g., user counter)
	IsOpen         bool
	container      *fyne.Container
	expandLabel    *widget.Label
	OnTap          func()                    // Optional callback for title tap
	OnTapSecondary func(pe *fyne.PointEvent) // Callback for secondary tap
}

// NewCustomAccordionItem creates a new custom accordion item
func NewCustomAccordionItem(title, content fyne.CanvasObject) *CustomAccordionItem {
	return NewCustomAccordionItemWithEndContent(title, content, nil)
}

// NewCustomAccordionItemWithEndContent creates a new custom accordion item with end content
func NewCustomAccordionItemWithEndContent(title, content, endContent fyne.CanvasObject) *CustomAccordionItem {
	item := &CustomAccordionItem{
		Title:      title,
		Content:    content,
		EndContent: endContent,
		IsOpen:     false,
	}

	// Create label for expansion indicator
	item.expandLabel = widget.NewLabel("▶")
	item.expandLabel.TextStyle = fyne.TextStyle{Monospace: true}

	item.updateContainer()
	return item
}

// NewCustomAccordionItemWithCallbacks creates a new custom accordion item with callbacks
func NewCustomAccordionItemWithCallbacks(title, content fyne.CanvasObject, onTap func(), onTapSecondary func(pe *fyne.PointEvent)) *CustomAccordionItem {
	return NewCustomAccordionItemWithEndContentAndCallbacks(title, content, nil, onTap, onTapSecondary)
}

// NewCustomAccordionItemWithEndContentAndCallbacks creates a new custom accordion item with end content and callbacks
func NewCustomAccordionItemWithEndContentAndCallbacks(title, content, endContent fyne.CanvasObject, onTap func(), onTapSecondary func(pe *fyne.PointEvent)) *CustomAccordionItem {
	item := NewCustomAccordionItemWithEndContent(title, content, endContent)
	item.OnTap = onTap
	item.OnTapSecondary = onTapSecondary
	return item
}

// Toggle toggles the expansion state of the item
func (item *CustomAccordionItem) Toggle() {
	item.IsOpen = !item.IsOpen
	item.updateContainer()
	item.container.Refresh()
}

// toggleState toggles only the state without calling updateContainer (to avoid recursion)
func (item *CustomAccordionItem) toggleState() {
	item.IsOpen = !item.IsOpen
}

// Open opens the item
func (item *CustomAccordionItem) Open() {
	if !item.IsOpen {
		item.IsOpen = true
		item.updateContainer()
		item.container.Refresh()
	}
}

// Close closes the item
func (item *CustomAccordionItem) Close() {
	if item.IsOpen {
		item.IsOpen = false
		item.updateContainer()
		item.container.Refresh()
	}
}

// updateContainer updates the container based on current state
func (item *CustomAccordionItem) updateContainer() {
	// Update expand label icon
	if item.IsOpen {
		item.expandLabel.SetText("▼")
	} else {
		item.expandLabel.SetText("▶")
	}

	// Create the content for the title row
	var endItems []fyne.CanvasObject
	if item.EndContent != nil {
		endItems = []fyne.CanvasObject{item.EndContent, item.expandLabel}
	} else {
		endItems = []fyne.CanvasObject{item.expandLabel}
	}

	titleContent := container.NewHBox(
		item.Title,
		layout.NewSpacer(),
	)

	// Add end items to the title content
	for _, endItem := range endItems {
		titleContent.Add(endItem)
	}

	// Create a tappable container for the title row
	titleContainer := NewTappableContainer(container.NewPadded(titleContent), func() {
		// Toggle state
		item.toggleState()
		// Update the container with new state
		item.updateContainer()
		item.container.Refresh()
		// Call custom callback if set
		if item.OnTap != nil {
			item.OnTap()
		}
	}, func(pe *fyne.PointEvent) {
		// Call secondary tap callback if set
		if item.OnTapSecondary != nil {
			item.OnTapSecondary(pe)
		}
	})

	// Clear and rebuild container
	if item.container != nil {
		item.container.RemoveAll()
	} else {
		item.container = container.NewVBox()
	}

	// Add title container
	item.container.Add(titleContainer)

	// Add content if open
	if item.IsOpen && item.Content != nil {
		item.container.Add(item.Content)
	}
}

// GetContainer returns the item's container
func (item *CustomAccordionItem) GetContainer() *fyne.Container {
	return item.container
}

// CustomAccordion represents a custom accordion that accepts widgets as titles
type CustomAccordion struct {
	widget.BaseWidget
	Items     []*CustomAccordionItem
	container *fyne.Container
}

// NewCustomAccordion creates a new custom accordion
func NewCustomAccordion() *CustomAccordion {
	accordion := &CustomAccordion{
		Items:     make([]*CustomAccordionItem, 0),
		container: container.NewVBox(),
	}
	accordion.ExtendBaseWidget(accordion)
	return accordion
}

// AddItem adds an item to the accordion
func (accordion *CustomAccordion) AddItem(item *CustomAccordionItem) {
	accordion.Items = append(accordion.Items, item)
	accordion.updateContainer()
}

// RemoveItem removes an item from the accordion
func (accordion *CustomAccordion) RemoveItem(item *CustomAccordionItem) {
	for i, existingItem := range accordion.Items {
		if existingItem == item {
			accordion.Items = append(accordion.Items[:i], accordion.Items[i+1:]...)
			break
		}
	}
	accordion.updateContainer()
}

// RemoveAll removes all items from the accordion
func (accordion *CustomAccordion) RemoveAll() {
	accordion.Items = make([]*CustomAccordionItem, 0)
	accordion.updateContainer()
}

// OpenAll opens all items
func (accordion *CustomAccordion) OpenAll() {
	for _, item := range accordion.Items {
		item.Open()
	}
}

// CloseAll closes all items
func (accordion *CustomAccordion) CloseAll() {
	for _, item := range accordion.Items {
		item.Close()
	}
}

// updateContainer updates the main accordion container
func (accordion *CustomAccordion) updateContainer() {
	accordion.container.RemoveAll()

	for i, item := range accordion.Items {
		accordion.container.Add(item.GetContainer())

		// Add separator between items (except for the last one)
		if i < len(accordion.Items)-1 {
			accordion.container.Add(widget.NewSeparator())
		}
	}

	accordion.container.Refresh()
}

// CreateRenderer implements the WidgetRenderer
func (accordion *CustomAccordion) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(accordion.container)
}

// GetContainer returns the main container
func (accordion *CustomAccordion) GetContainer() *fyne.Container {
	return accordion.container
}
