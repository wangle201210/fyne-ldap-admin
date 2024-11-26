package app

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/go-ldap/ldap/v3"
	"github.com/wangle201210/fyne-ldap-admin/app/ext"
)

// ResultShow displays search results in an improved UI
func (x *LdapAdmin) ResultShow() {
	if x.resultWindow != nil {
		x.updateResultContent()
		x.resultWindow.Show()
		return
	}

	size := fyne.Size{Width: 1000, Height: 800}

	// Create a toolbar with actions
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.DocumentCreateIcon(), func() {
			if x.selectData != nil {
				x.showEditDialog(x.selectData)
			}
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.ViewRefreshIcon(), func() {
			x.selectData = nil // Clear selection before refresh
			if x.currentList != nil {
				x.currentList.UnselectAll()
			}
			x.Search()
		}),
	)

	// Create status bar
	x.statusLabel = widget.NewLabel(fmt.Sprintf("Found %d entries", len(x.data)))
	statusBar := container.NewHBox(x.statusLabel)

	// Create main content area
	x.resultContent = x.createResultList()
	mainContent := container.NewBorder(toolbar, statusBar, nil, nil, x.resultContent)

	// Create and show the window
	x.resultWindow = x.App.NewWindow("LDAP Search Results")
	x.resultWindow.Resize(size)
	x.resultWindow.SetContent(mainContent)

	// Handle window close event
	x.resultWindow.SetOnClosed(func() {
		x.resultWindow = nil
		x.resultContent = nil
		x.currentList = nil
		x.selectData = nil
	})

	x.resultWindow.Show()
}

func (x *LdapAdmin) updateResultContent() {
	// Clear selection and list reference
	x.selectData = nil
	if x.currentList != nil {
		x.currentList.UnselectAll()
	}
	x.currentList = nil

	if x.resultContent != nil {
		x.resultContent.Hide()
		x.resultContent = x.createResultList()

		// Get existing toolbar and status bar
		existingContent := x.resultWindow.Content().(*fyne.Container)
		toolbar := existingContent.Objects[0]
		statusBar := existingContent.Objects[1]

		// Create new content with existing toolbar and status bar
		newContent := container.NewBorder(toolbar, statusBar, nil, nil, x.resultContent)
		x.resultWindow.SetContent(newContent)
		x.resultContent.Show()
	}

	if x.statusLabel != nil {
		x.statusLabel.SetText(fmt.Sprintf("Found %d entries", len(x.data)))
	}
}

// createResultList creates an enhanced list view for LDAP entries
func (x *LdapAdmin) createResultList() fyne.CanvasObject {
	if len(x.data) == 0 {
		return widget.NewLabel("No results found")
	}

	// Create split container
	split := container.NewHSplit(
		x.createEntryList(),
		x.createDetailView(),
	)
	split.SetOffset(0.3)

	return split
}

// createEntryList creates the list of LDAP entries
func (x *LdapAdmin) createEntryList() fyne.CanvasObject {
	list := ext.NewList(
		func() int { return len(x.data) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(theme.AccountIcon()),
				widget.NewLabel("Template"),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			box := item.(*fyne.Container)
			label := box.Objects[1].(*widget.Label)
			entry := x.data[id]

			// Get CN from DN if possible
			displayName := entry.DN
			if cn := entry.GetAttributeValue("cn"); cn != "" {
				displayName = cn
			}
			label.SetText(displayName)
		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		x.selectData = x.data[id]
		x.refreshDetailView()

		// Refresh the entire list to update background colors
		if x.currentList != nil {
			x.currentList.Refresh()
		}
	}

	list.OnUnselected = func(id widget.ListItemID) {
		x.selectData = nil
		x.refreshDetailView()

		// Refresh the entire list to update background colors
		if x.currentList != nil {
			x.currentList.Refresh()
		}
	}

	// Store the list reference for later use
	x.currentList = &list.List

	// Wrap list in a card for better visual appearance
	return widget.NewCard("Entries", "", list)
}

func (x *LdapAdmin) createDetailView() fyne.CanvasObject {
	x.detailContent = widget.NewTextGrid()
	x.refreshDetailView()

	// Wrap in a scroll container and card
	scroll := container.NewScroll(x.detailContent)
	return widget.NewCard("Details", "", scroll)
}

func (x *LdapAdmin) refreshDetailView() {
	if x.selectData == nil {
		x.detailContent.SetText("Select an entry to view details")
		return
	}

	var details strings.Builder
	details.WriteString(fmt.Sprintf("DN: %s\n\n", x.selectData.DN))

	// Group attributes by category
	categories := map[string][]string{
		"Name Attributes":   {"cn", "sn", "givenName"},
		"Contact Info":      {"mail", "telephoneNumber"},
		"Account Details":   {"uid", "uidNumber", "gidNumber"},
		"Organization":      {"o", "ou", "title"},
		"System Attributes": {"objectClass", "createTimestamp", "modifyTimestamp"},
	}

	// Write attributes by category
	for category, attrs := range categories {
		hasCategory := false
		for _, attr := range attrs {
			if values := x.selectData.GetAttributeValues(attr); len(values) > 0 {
				if !hasCategory {
					details.WriteString(fmt.Sprintf("\n=== %s ===\n", category))
					hasCategory = true
				}
				if len(values) == 1 {
					details.WriteString(fmt.Sprintf("%s: %s\n", attr, values[0]))
				} else {
					details.WriteString(fmt.Sprintf("%s:\n", attr))
					for _, v := range values {
						details.WriteString(fmt.Sprintf("  - %s\n", v))
					}
				}
			}
		}
	}

	// Write remaining attributes
	hasOthers := false
	for _, attr := range x.selectData.Attributes {
		if !isAttributeInCategories(attr.Name, categories) {
			if !hasOthers {
				details.WriteString("\n=== Other Attributes ===\n")
				hasOthers = true
			}
			if len(attr.Values) == 1 {
				details.WriteString(fmt.Sprintf("%s: %s\n", attr.Name, attr.Values[0]))
			} else {
				details.WriteString(fmt.Sprintf("%s:\n", attr.Name))
				for _, v := range attr.Values {
					details.WriteString(fmt.Sprintf("  - %s\n", v))
				}
			}
		}
	}

	x.detailContent.SetText(details.String())
}

func isAttributeInCategories(attr string, categories map[string][]string) bool {
	for _, attrs := range categories {
		for _, a := range attrs {
			if a == attr {
				return true
			}
		}
	}
	return false
}

func (x *LdapAdmin) showEditDialog(entry *ldap.Entry) {
	var modifyRequest *ldap.ModifyRequest
	modifyRequest = ldap.NewModifyRequest(entry.DN, nil)

	// Create form items for editable attributes
	form := &widget.Form{}
	formItems := []*widget.FormItem{}

	// Group attributes by categories
	categories := map[string][]string{
		"Basic Information": {"cn", "sn", "givenName", "displayName", "title"},
		"Contact":           {"mail", "telephoneNumber", "mobile"},
		"Organization":      {"o", "ou", "department", "company"},
		"Account":           {"uid", "uidNumber", "gidNumber", "homeDirectory", "loginShell"},
		"Other":             {},
	}

	// Create input fields for each attribute
	inputs := make(map[string]*widget.Entry)

	// Process attributes by category
	for _, attrs := range categories {
		for _, attr := range attrs {
			if value := entry.GetAttributeValue(attr); value != "" {
				input := widget.NewEntry()
				input.SetText(value)
				inputs[attr] = input
				formItems = append(formItems, widget.NewFormItem(attr, input))
			}
		}
	}

	// Add remaining attributes to "Other" category
	for _, attr := range entry.Attributes {
		attrName := attr.Name
		if !isAttributeInCategories(attrName, categories) && len(attr.Values) > 0 {
			input := widget.NewEntry()
			input.SetText(attr.Values[0])
			inputs[attrName] = input
			formItems = append(formItems, widget.NewFormItem(attrName, input))
		}
	}

	form.Items = formItems

	// Add save button
	saveButton := widget.NewButton("Save", func() {
		// Collect modifications
		for attrName, input := range inputs {
			newValue := input.Text
			oldValue := entry.GetAttributeValue(attrName)

			if newValue != oldValue {
				modifyRequest.Replace(attrName, []string{newValue})
			}
		}

		// Apply modifications
		if len(modifyRequest.Changes) > 0 {
			ldapConn := x.GetConn()
			if ldapConn == nil {
				dialog.ShowError(fmt.Errorf("failed to connect to LDAP server"), x.resultWindow)
				return
			}

			err := ldapConn.Modify(modifyRequest)
			if err != nil {
				dialog.ShowError(fmt.Errorf("failed to modify entry: %v", err), x.resultWindow)
				return
			}

			// Show success message
			dialog.ShowInformation("Success", "Entry modified successfully", x.resultWindow)

			// Refresh the display after successful modification
			x.Search()
		}
	})

	// Create content container with scroll
	content := container.NewVBox(
		form,
		container.NewHBox(
			widget.NewLabel(""), // spacer
			saveButton,
		),
	)

	scroll := container.NewScroll(content)

	// Show custom dialog with result window as parent
	customDialog := dialog.NewCustom("Edit Entry", "Close", scroll, x.resultWindow)
	customDialog.Resize(fyne.NewSize(800, 600))
	customDialog.Show()
}
