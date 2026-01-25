package ui

import tea "github.com/charmbracelet/bubbletea"

// Dialog wraps any tea.Model content and automatically adds a header with title.
// This enforces consistent dialog headers across the application "by design" -
// you cannot create a dialog without getting a header.
//
// Usage:
//   contentForm := NewSessionForm(...)
//   dialog := NewDialog("Create Session", contentForm, devMode)
//   dialog.Init()  // Delegates to contentForm.Init()
//   dialog.Update(msg)  // Delegates to contentForm.Update(msg)
//   dialog.View()  // Returns header + contentForm.View()
//
// Design principles:
// - Composition: Dialog wraps content via delegation (not inheritance)
// - Small interfaces: Uses existing tea.Model interface (Init, Update, View)
// - Open/Closed: Add new dialogs by wrapping content, no Dialog changes needed
// - Single responsibility: Dialog handles structure, content handles business logic
type Dialog struct {
	content tea.Model
	devMode bool
	title   string
}

// NewDialog creates a new dialog wrapper that automatically adds headers.
// The wrapped content will have renderDialogHeader() automatically prepended to its View().
func NewDialog(title string, content tea.Model, devMode bool) *Dialog {
	return &Dialog{
		content: content,
		devMode: devMode,
		title:   title,
	}
}

// Init delegates to wrapped content's Init method.
func (d *Dialog) Init() tea.Cmd {
	return d.content.Init()
}

// Update delegates to wrapped content's Update method.
// The returned tea.Model is the Dialog itself with updated content.
func (d *Dialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updatedContent, cmd := d.content.Update(msg)
	d.content = updatedContent
	return d, cmd
}

// View automatically prepends the dialog header to the wrapped content's view.
// This ensures all dialogs have consistent headers without manual calls to renderDialogHeader().
func (d *Dialog) View() string {
	return renderDialogHeader(d.devMode, d.title) + d.content.View()
}

// Content returns the wrapped content for type assertion.
// This allows callers to access content-specific fields after Update().
//
// Example:
//   if content, ok := dialog.Content().(*SessionForm); ok {
//       if content.Completed {
//           result := content.Result()
//       }
//   }
func (d *Dialog) Content() tea.Model {
	return d.content
}
