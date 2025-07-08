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
	content   fyne.CanvasObject
	onTap     func()
	container *fyne.Container
}

// NewTappableContainer cria um novo container clicável
func NewTappableContainer(content fyne.CanvasObject, onTap func()) *TappableContainer {
	tc := &TappableContainer{
		content: content,
		onTap:   onTap,
	}
	tc.ExtendBaseWidget(tc)
	tc.container = container.NewStack(content)
	return tc
}

// CreateRenderer implementa o WidgetRenderer
func (tc *TappableContainer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(tc.container)
}

// Tapped implementa o evento de clique
func (tc *TappableContainer) Tapped(pe *fyne.PointEvent) {
	if tc.onTap != nil {
		tc.onTap()
	}
}

// CustomAccordionItem representa um item do accordion personalizado que aceita qualquer widget como título
type CustomAccordionItem struct {
	Title       fyne.CanvasObject // Pode ser qualquer widget
	Content     fyne.CanvasObject
	EndContent  fyne.CanvasObject // Conteúdo que fica no final da linha (ex: contador de usuários)
	IsOpen      bool
	container   *fyne.Container
	expandLabel *widget.Label
	titleButton *CustomButton
	OnTap       func() // Callback opcional para clique no título
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
func NewCustomAccordionItemWithCallbacks(title, content fyne.CanvasObject, onTap func()) *CustomAccordionItem {
	return NewCustomAccordionItemWithEndContentAndCallbacks(title, content, nil, onTap)
}

// NewCustomAccordionItemWithEndContentAndCallbacks cria um novo item do accordion personalizado com conteúdo no final e callbacks
func NewCustomAccordionItemWithEndContentAndCallbacks(title, content, endContent fyne.CanvasObject, onTap func()) *CustomAccordionItem {
	item := NewCustomAccordionItemWithEndContent(title, content, endContent)
	item.OnTap = onTap
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

	// Create the content for the button (title + end content + expand indicator)
	var endItems []fyne.CanvasObject
	if item.EndContent != nil {
		endItems = []fyne.CanvasObject{item.EndContent, item.expandLabel}
	} else {
		endItems = []fyne.CanvasObject{item.expandLabel}
	}

	buttonContent := container.NewHBox(
		item.Title,
		layout.NewSpacer(),
	)

	// Add end items to the button content
	for _, endItem := range endItems {
		buttonContent.Add(endItem)
	}

	// Create a button that spans the entire title row
	if item.titleButton == nil {
		item.titleButton = NewCustomButton("", func() {
			// Toggle state
			item.toggleState()
			// Update the container with new state
			item.updateContainer()
			item.container.Refresh()
			// Call custom callback if set
			if item.OnTap != nil {
				item.OnTap()
			}
		})
		// Make button flat and remove importance
		item.titleButton.SetImportance(widget.LowImportance)
	}

	// Clear and rebuild container
	if item.container != nil {
		item.container.RemoveAll()
	} else {
		item.container = container.NewVBox()
	}

	// Create a container that overlays the button content on the button
	titleContainer := container.NewStack(
		item.titleButton,
		container.NewPadded(buttonContent),
	)

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

// CustomButton é um botão customizado
type CustomButton struct {
	widget.BaseWidget
	button *widget.Button
	onTap  func()
}

// NewCustomButton cria um novo botão customizado
func NewCustomButton(text string, onTap func()) *CustomButton {
	cb := &CustomButton{
		onTap: onTap,
	}
	cb.button = widget.NewButton(text, onTap)
	cb.ExtendBaseWidget(cb)
	return cb
}

// CreateRenderer implementa o WidgetRenderer para CustomButton
func (cb *CustomButton) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(cb.button)
}

// Tapped implementa o evento de clique
func (cb *CustomButton) Tapped(pe *fyne.PointEvent) {
	if cb.onTap != nil {
		cb.onTap()
	}
}

// SetText define o texto do botão
func (cb *CustomButton) SetText(text string) {
	cb.button.SetText(text)
}

// SetImportance define a importância do botão
func (cb *CustomButton) SetImportance(importance widget.Importance) {
	cb.button.Importance = importance
}
