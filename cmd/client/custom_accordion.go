package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// TappableContainer é um container que aceita eventos de tap
type TappableContainer struct {
	widget.BaseWidget
	content        fyne.CanvasObject
	onTap          func()
	onTapSecondary func(pe *fyne.PointEvent)
}

// NewTappableContainer cria um novo container clicável
func NewTappableContainer(content fyne.CanvasObject, onTap func(), onTapSecondary func(pe *fyne.PointEvent)) *TappableContainer {
	tc := &TappableContainer{
		content:        content,
		onTap:          onTap,
		onTapSecondary: onTapSecondary,
	}
	tc.ExtendBaseWidget(tc)
	return tc
}

// CreateRenderer implementa o WidgetRenderer
func (tc *TappableContainer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(tc.content)
}

// Tapped implementa o evento de clique
func (tc *TappableContainer) Tapped(pe *fyne.PointEvent) {
	if tc.onTap != nil {
		tc.onTap()
	}
}

// TappedSecondary implementa o evento de clique secundário (direito)
func (tc *TappableContainer) TappedSecondary(pe *fyne.PointEvent) {
	if tc.onTapSecondary != nil {
		tc.onTapSecondary(pe)
	}
}

// CustomAccordionItem representa um item do accordion personalizado que aceita qualquer widget como título
type CustomAccordionItem struct {
	Title          fyne.CanvasObject // Pode ser qualquer widget
	Content        fyne.CanvasObject
	EndContent     fyne.CanvasObject // Conteúdo que fica no final da linha (ex: contador de usuários)
	IsOpen         bool
	container      *fyne.Container
	expandLabel    *widget.Label
	OnTap          func()                       // Callback opcional para clique no título
	OnTapSecondary func(pe *fyne.PointEvent) // Callback para clique secundário
}

// NewCustomAccordionItem cria um novo item do accordion personalizado
func NewCustomAccordionItem(title, content fyne.CanvasObject) *CustomAccordionItem {
	return NewCustomAccordionItemWithEndContent(title, content, nil)
}

// NewCustomAccordionItemWithEndContent cria um novo item do accordion personalizado com conteúdo no final
func NewCustomAccordionItemWithEndContent(title, content, endContent fyne.CanvasObject) *CustomAccordionItem {
	item := &CustomAccordionItem{
		Title:      title,
		Content:    content,
		EndContent: endContent,
		IsOpen:     false,
	}

	// Criar label para o indicador de expansão
	item.expandLabel = widget.NewLabel("▶")
	item.expandLabel.TextStyle = fyne.TextStyle{Monospace: true}

	item.updateContainer()
	return item
}

// NewCustomAccordionItemWithCallbacks cria um novo item do accordion personalizado com callbacks
func NewCustomAccordionItemWithCallbacks(title, content fyne.CanvasObject, onTap func(), onTapSecondary func(pe *fyne.PointEvent)) *CustomAccordionItem {
	return NewCustomAccordionItemWithEndContentAndCallbacks(title, content, nil, onTap, onTapSecondary)
}

// NewCustomAccordionItemWithEndContentAndCallbacks cria um novo item do accordion personalizado com conteúdo no final e callbacks
func NewCustomAccordionItemWithEndContentAndCallbacks(title, content, endContent fyne.CanvasObject, onTap func(), onTapSecondary func(pe *fyne.PointEvent)) *CustomAccordionItem {
	item := NewCustomAccordionItemWithEndContent(title, content, endContent)
	item.OnTap = onTap
	item.OnTapSecondary = onTapSecondary
	return item
}

// Toggle alterna o estado de expansão do item
func (item *CustomAccordionItem) Toggle() {
	item.IsOpen = !item.IsOpen
	item.updateContainer()
	item.container.Refresh()
}

// toggleState alterna apenas o estado sem chamar updateContainer (para evitar recursão)
func (item *CustomAccordionItem) toggleState() {
	item.IsOpen = !item.IsOpen
}

// Open abre o item
func (item *CustomAccordionItem) Open() {
	if !item.IsOpen {
		item.IsOpen = true
		item.updateContainer()
		item.container.Refresh()
	}
}

// Close fecha o item
func (item *CustomAccordionItem) Close() {
	if item.IsOpen {
		item.IsOpen = false
		item.updateContainer()
		item.container.Refresh()
	}
}

// updateContainer atualiza o container baseado no estado atual
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

// GetContainer retorna o container do item
func (item *CustomAccordionItem) GetContainer() *fyne.Container {
	return item.container
}

// CustomAccordion representa um accordion personalizado que aceita widgets como título
type CustomAccordion struct {
	widget.BaseWidget
	Items     []*CustomAccordionItem
	container *fyne.Container
}

// NewCustomAccordion cria um novo accordion personalizado
func NewCustomAccordion() *CustomAccordion {
	accordion := &CustomAccordion{
		Items:     make([]*CustomAccordionItem, 0),
		container: container.NewVBox(),
	}
	accordion.ExtendBaseWidget(accordion)
	return accordion
}

// AddItem adiciona um item ao accordion
func (accordion *CustomAccordion) AddItem(item *CustomAccordionItem) {
	accordion.Items = append(accordion.Items, item)
	accordion.updateContainer()
}

// RemoveItem remove um item do accordion
func (accordion *CustomAccordion) RemoveItem(item *CustomAccordionItem) {
	for i, existingItem := range accordion.Items {
		if existingItem == item {
			accordion.Items = append(accordion.Items[:i], accordion.Items[i+1:]...)
			break
		}
	}
	accordion.updateContainer()
}

// RemoveAll remove todos os itens do accordion
func (accordion *CustomAccordion) RemoveAll() {
	accordion.Items = make([]*CustomAccordionItem, 0)
	accordion.updateContainer()
}

// OpenAll abre todos os itens
func (accordion *CustomAccordion) OpenAll() {
	for _, item := range accordion.Items {
		item.Open()
	}
}

// CloseAll fecha todos os itens
func (accordion *CustomAccordion) CloseAll() {
	for _, item := range accordion.Items {
		item.Close()
	}
}

// updateContainer atualiza o container principal do accordion
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

// CreateRenderer implementa o WidgetRenderer
func (accordion *CustomAccordion) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(accordion.container)
}

// GetContainer retorna o container principal
func (accordion *CustomAccordion) GetContainer() *fyne.Container {
	return accordion.container
}