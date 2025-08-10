package components

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/ctakes-tui/ctakes-tui/internal/theme"
)

// InputField - Reusable text input component
type InputField struct {
	// Content
	value       string
	placeholder string
	cursor      int
	viewStart   int

	// Configuration
	width     int
	maxLength int
	password  bool
	multiline bool
	label     string

	// State
	focused  bool
	disabled bool

	// Validation
	validator func(string) error
	required  bool

	// Callbacks
	onChange func(value string)
	onSubmit func(value string)
	onCancel func()
}

// NewInputField creates a new input field component
func NewInputField(placeholder string) *InputField {
	return &InputField{
		placeholder: placeholder,
		width:       40,
		maxLength:   256,
		cursor:      0,
		viewStart:   0,
	}
}

// Configuration methods
func (inp *InputField) SetWidth(width int) *InputField {
	inp.width = width
	inp.adjustView()
	return inp
}

func (inp *InputField) SetMaxLength(maxLength int) *InputField {
	inp.maxLength = maxLength
	if len(inp.value) > maxLength {
		inp.value = inp.value[:maxLength]
		if inp.cursor > len(inp.value) {
			inp.cursor = len(inp.value)
		}
	}
	inp.adjustView()
	return inp
}

func (inp *InputField) SetPassword(password bool) *InputField {
	inp.password = password
	return inp
}

func (inp *InputField) SetMultiline(multiline bool) *InputField {
	inp.multiline = multiline
	return inp
}

func (inp *InputField) SetLabel(label string) *InputField {
	inp.label = label
	return inp
}

func (inp *InputField) SetRequired(required bool) *InputField {
	inp.required = required
	return inp
}

func (inp *InputField) SetValidator(validator func(string) error) *InputField {
	inp.validator = validator
	return inp
}

func (inp *InputField) OnChange(callback func(value string)) *InputField {
	inp.onChange = callback
	return inp
}

func (inp *InputField) OnSubmit(callback func(value string)) *InputField {
	inp.onSubmit = callback
	return inp
}

func (inp *InputField) OnCancel(callback func()) *InputField {
	inp.onCancel = callback
	return inp
}

// State methods
func (inp *InputField) SetValue(value string) *InputField {
	if len(value) > inp.maxLength {
		value = value[:inp.maxLength]
	}
	inp.value = value
	inp.cursor = len(value)
	inp.adjustView()
	if inp.onChange != nil {
		inp.onChange(inp.value)
	}
	return inp
}

func (inp *InputField) GetValue() string {
	return inp.value
}

func (inp *InputField) SetFocused(focused bool) *InputField {
	inp.focused = focused
	return inp
}

func (inp *InputField) SetDisabled(disabled bool) *InputField {
	inp.disabled = disabled
	return inp
}

func (inp *InputField) Clear() *InputField {
	inp.value = ""
	inp.cursor = 0
	inp.viewStart = 0
	if inp.onChange != nil {
		inp.onChange(inp.value)
	}
	return inp
}

// Input methods
func (inp *InputField) InsertRune(r rune) {
	if inp.disabled {
		return
	}

	if len(inp.value)+utf8.RuneLen(r) > inp.maxLength {
		return
	}

	// Insert rune at cursor position
	before := inp.value[:inp.cursor]
	after := inp.value[inp.cursor:]
	inp.value = before + string(r) + after
	inp.cursor++

	inp.adjustView()
	if inp.onChange != nil {
		inp.onChange(inp.value)
	}
}

func (inp *InputField) InsertString(s string) {
	if inp.disabled {
		return
	}

	for _, r := range s {
		inp.InsertRune(r)
	}
}

func (inp *InputField) DeleteBefore() {
	if inp.disabled || inp.cursor == 0 {
		return
	}

	// Find the previous rune boundary
	i := inp.cursor - 1
	for i > 0 {
		if utf8.ValidString(inp.value[i:inp.cursor]) {
			break
		}
		i--
	}

	inp.value = inp.value[:i] + inp.value[inp.cursor:]
	inp.cursor = i
	inp.adjustView()

	if inp.onChange != nil {
		inp.onChange(inp.value)
	}
}

func (inp *InputField) DeleteAfter() {
	if inp.disabled || inp.cursor >= len(inp.value) {
		return
	}

	// Find the next rune boundary
	_, size := utf8.DecodeRuneInString(inp.value[inp.cursor:])
	inp.value = inp.value[:inp.cursor] + inp.value[inp.cursor+size:]
	inp.adjustView()

	if inp.onChange != nil {
		inp.onChange(inp.value)
	}
}

func (inp *InputField) DeleteWord() {
	if inp.disabled || inp.cursor == 0 {
		return
	}

	// Find the start of the current word
	i := inp.cursor - 1

	// Skip whitespace
	for i >= 0 && inp.value[i] == ' ' {
		i--
	}

	// Skip word characters
	for i >= 0 && inp.value[i] != ' ' {
		i--
	}
	i++

	inp.value = inp.value[:i] + inp.value[inp.cursor:]
	inp.cursor = i
	inp.adjustView()

	if inp.onChange != nil {
		inp.onChange(inp.value)
	}
}

// Navigation methods
func (inp *InputField) MoveCursorLeft() {
	if inp.cursor > 0 {
		inp.cursor--
		inp.adjustView()
	}
}

func (inp *InputField) MoveCursorRight() {
	if inp.cursor < len(inp.value) {
		inp.cursor++
		inp.adjustView()
	}
}

func (inp *InputField) MoveCursorToStart() {
	inp.cursor = 0
	inp.adjustView()
}

func (inp *InputField) MoveCursorToEnd() {
	inp.cursor = len(inp.value)
	inp.adjustView()
}

func (inp *InputField) MoveCursorWordLeft() {
	if inp.cursor == 0 {
		return
	}

	i := inp.cursor - 1

	// Skip whitespace
	for i >= 0 && inp.value[i] == ' ' {
		i--
	}

	// Skip word characters
	for i >= 0 && inp.value[i] != ' ' {
		i--
	}
	i++

	inp.cursor = i
	inp.adjustView()
}

func (inp *InputField) MoveCursorWordRight() {
	if inp.cursor >= len(inp.value) {
		return
	}

	i := inp.cursor

	// Skip word characters
	for i < len(inp.value) && inp.value[i] != ' ' {
		i++
	}

	// Skip whitespace
	for i < len(inp.value) && inp.value[i] == ' ' {
		i++
	}

	inp.cursor = i
	inp.adjustView()
}

// Adjust the view to keep cursor visible
func (inp *InputField) adjustView() {
	displayWidth := inp.width - 2 // Account for borders/padding
	if displayWidth < 1 {
		displayWidth = 1
	}

	// Adjust view start to keep cursor visible
	if inp.cursor < inp.viewStart {
		inp.viewStart = inp.cursor
	} else if inp.cursor >= inp.viewStart+displayWidth {
		inp.viewStart = inp.cursor - displayWidth + 1
	}

	// Don't scroll past the beginning
	if inp.viewStart < 0 {
		inp.viewStart = 0
	}
}

// Action methods
func (inp *InputField) Submit() {
	if inp.disabled {
		return
	}

	// Validate if validator is set
	if inp.validator != nil {
		if err := inp.validator(inp.value); err != nil {
			return // Don't submit if validation fails
		}
	}

	// Check required field
	if inp.required && strings.TrimSpace(inp.value) == "" {
		return // Don't submit empty required field
	}

	if inp.onSubmit != nil {
		inp.onSubmit(inp.value)
	}
}

func (inp *InputField) Cancel() {
	if inp.onCancel != nil {
		inp.onCancel()
	}
}

// Validation
func (inp *InputField) Validate() error {
	if inp.required && strings.TrimSpace(inp.value) == "" {
		return fmt.Errorf("This field is required")
	}

	if inp.validator != nil {
		return inp.validator(inp.value)
	}

	return nil
}

func (inp *InputField) IsValid() bool {
	return inp.Validate() == nil
}

// Rendering
func (inp *InputField) Render() string {
	lines := []string{}

	// Label
	if inp.label != "" {
		label := inp.label
		if inp.required {
			label += " *"
		}
		lines = append(lines, theme.RenderTextBold(label))
	}

	// Input field
	inputLine := inp.renderInputField()
	lines = append(lines, inputLine)

	// Validation error
	if err := inp.Validate(); err != nil && inp.focused {
		errorLine := theme.RenderStatus("error", err.Error())
		lines = append(lines, errorLine)
	}

	return strings.Join(lines, "\n")
}

func (inp *InputField) renderInputField() string {
	displayWidth := inp.width - 2 // Account for borders
	if displayWidth < 1 {
		displayWidth = 1
	}

	// Get visible portion of text
	var displayText string
	if inp.password {
		displayText = strings.Repeat("•", len(inp.value))
	} else {
		displayText = inp.value
	}

	// Apply view window
	endIndex := inp.viewStart + displayWidth
	if endIndex > len(displayText) {
		endIndex = len(displayText)
	}

	if inp.viewStart < len(displayText) {
		displayText = displayText[inp.viewStart:endIndex]
	} else {
		displayText = ""
	}

	// Pad to full width
	if len(displayText) < displayWidth {
		displayText += strings.Repeat(" ", displayWidth-len(displayText))
	}

	// Add cursor if focused
	if inp.focused && !inp.disabled {
		cursorPos := inp.cursor - inp.viewStart
		if cursorPos >= 0 && cursorPos <= len(displayText) {
			if cursorPos == len(displayText) {
				// Cursor at end
				if len(displayText) < displayWidth {
					displayText = displayText[:len(displayText)-1] + "│"
				} else {
					displayText = displayText[:displayWidth-1] + "│"
				}
			} else {
				// Cursor in middle
				displayText = displayText[:cursorPos] + "│" + displayText[cursorPos+1:]
			}
		}
	}

	// Show placeholder if empty and not focused
	if inp.value == "" && !inp.focused && inp.placeholder != "" {
		placeholderText := inp.placeholder
		if len(placeholderText) > displayWidth {
			placeholderText = placeholderText[:displayWidth]
		}
		if len(placeholderText) < displayWidth {
			placeholderText += strings.Repeat(" ", displayWidth-len(placeholderText))
		}
		displayText = theme.RenderTextDim(placeholderText)
	}

	// Apply styling based on state
	var style func(string) string
	if inp.disabled {
		style = theme.RenderTextDim
	} else if inp.focused {
		style = func(s string) string {
			return theme.BorderActiveStyle.Width(inp.width).Render(s)
		}
	} else {
		style = func(s string) string {
			return theme.BorderStyle.Width(inp.width).Render(s)
		}
	}

	return style(displayText)
}

// RenderCompact renders input field without label (for inline use)
func (inp *InputField) RenderCompact() string {
	return inp.renderInputField()
}

// Common validators
func ValidateNotEmpty(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("This field cannot be empty")
	}
	return nil
}

func ValidateMinLength(minLength int) func(string) error {
	return func(value string) error {
		if len(strings.TrimSpace(value)) < minLength {
			return fmt.Errorf("Must be at least %d characters", minLength)
		}
		return nil
	}
}

func ValidateMaxLength(maxLength int) func(string) error {
	return func(value string) error {
		if len(value) > maxLength {
			return fmt.Errorf("Must be no more than %d characters", maxLength)
		}
		return nil
	}
}
