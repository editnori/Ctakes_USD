package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ctakes-tui/ctakes-tui/internal/theme"
)

type PipelineModel struct {
	width      int
	height     int
	cursor     int
	components []PipelineComponent
	spinner    spinner.Model
	saving     bool
	ready      bool
	message    string
	messageType string
}

type PipelineComponent struct {
	Name        string
	Description string
	Enabled     bool
	Required    bool
	Category    string
	Icon        string
	Config      map[string]interface{}
}

func NewPipelineModel() PipelineModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.ColorAccent)

	return PipelineModel{
		spinner: s,
		components: []PipelineComponent{
			// Tokenization
			{Name: "Sentence Detector", Description: "Splits text into sentences", Enabled: true, Required: true, Category: "Tokenization", Icon: "▫"},
			{Name: "Tokenizer", Description: "Breaks sentences into tokens", Enabled: true, Required: true, Category: "Tokenization", Icon: "▦"},
			
			// Core NLP
			{Name: "Part-of-Speech Tagger", Description: "Tags words with grammatical roles", Enabled: true, Required: false, Category: "Core NLP", Icon: "◈"},
			{Name: "Constituency Parser", Description: "Analyzes sentence structure", Enabled: false, Required: false, Category: "Core NLP", Icon: "▥"},
			{Name: "Dependency Parser", Description: "Identifies word relationships", Enabled: true, Required: false, Category: "Core NLP", Icon: "◆"},
			
			// Clinical NLP
			{Name: "Named Entity Recognizer", Description: "Identifies medical entities", Enabled: true, Required: false, Category: "Clinical", Icon: theme.IconMedical},
			{Name: "Assertion Analyzer", Description: "Determines negation and uncertainty", Enabled: true, Required: false, Category: "Clinical", Icon: "◔"},
			{Name: "Context Annotator", Description: "Adds contextual information", Enabled: true, Required: false, Category: "Clinical", Icon: "◎"},
			
			// Ontology Mapping
			{Name: "UMLS Lookup", Description: "Maps to UMLS concepts", Enabled: true, Required: false, Category: "Ontology", Icon: "◎", 
				Config: map[string]interface{}{"dictionary": "SNOMEDCT_US"}},
			{Name: "Drug NER", Description: "Identifies medications", Enabled: true, Required: false, Category: "Ontology", Icon: "◉",
				Config: map[string]interface{}{"source": "RxNorm"}},
			{Name: "Side Effect Extractor", Description: "Extracts adverse events", Enabled: false, Required: false, Category: "Ontology", Icon: "▲"},
			
			// Advanced
			{Name: "Relation Extractor", Description: "Finds relationships between entities", Enabled: false, Required: false, Category: "Advanced", Icon: "↔"},
			{Name: "Temporal Expression", Description: "Extracts time-related information", Enabled: true, Required: false, Category: "Advanced", Icon: "◐"},
			{Name: "Coreference Resolver", Description: "Resolves pronouns and references", Enabled: false, Required: false, Category: "Advanced", Icon: "⇄"},
		},
	}
}

func (m PipelineModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m PipelineModel) Update(msg tea.Msg) (PipelineModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, pipelineKeys.Back):
			return m, func() tea.Msg { return "main_menu" }
			
		case key.Matches(msg, pipelineKeys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
			
		case key.Matches(msg, pipelineKeys.Down):
			if m.cursor < len(m.components)-1 {
				m.cursor++
			}
			
		case key.Matches(msg, pipelineKeys.Toggle):
			if !m.components[m.cursor].Required {
				m.components[m.cursor].Enabled = !m.components[m.cursor].Enabled
				m.message = fmt.Sprintf("%s %s", m.components[m.cursor].Name,
					map[bool]string{true: "enabled", false: "disabled"}[m.components[m.cursor].Enabled])
				m.messageType = "info"
			}
			
		case key.Matches(msg, pipelineKeys.Save):
			m.saving = true
			m.message = "Saving configuration..."
			m.messageType = "info"
			cmds = append(cmds, m.saveConfiguration())
			
		case key.Matches(msg, pipelineKeys.Reset):
			m.resetToDefaults()
			m.message = "Reset to default configuration"
			m.messageType = "success"
			
		case key.Matches(msg, pipelineKeys.EnableAll):
			for i := range m.components {
				m.components[i].Enabled = true
			}
			m.message = "All components enabled"
			m.messageType = "success"
			
		case key.Matches(msg, pipelineKeys.DisableOptional):
			for i := range m.components {
				if !m.components[i].Required {
					m.components[i].Enabled = false
				}
			}
			m.message = "Optional components disabled"
			m.messageType = "info"
		}
		
	case configSavedMsg:
		m.saving = false
		m.message = "Configuration saved successfully"
		m.messageType = "success"
		
	case spinner.TickMsg:
		if m.saving {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m PipelineModel) View() string {
	if !m.ready {
		return theme.BaseStyle.Render("Loading pipeline configuration...")
	}

	topPadding := "\n\n"
	header := m.renderHeader()
	content := m.renderContent()
	footer := m.renderFooter()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		topPadding,
		header,
		content,
		footer,
	)
}

func (m *PipelineModel) renderHeader() string {
	title := theme.RenderTitle(theme.IconPipeline, "cTAKES Pipeline Configuration")
	
	enabledCount := 0
	for _, c := range m.components {
		if c.Enabled {
			enabledCount++
		}
	}
	
	status := fmt.Sprintf("%d/%d components enabled", enabledCount, len(m.components))
	statusStyle := theme.StatusStyle
	
	if m.saving {
		status = m.spinner.View() + " Saving..."
		statusStyle = theme.StatusInfoStyle
	} else if m.message != "" {
		status = m.message
		switch m.messageType {
		case "success":
			statusStyle = theme.StatusSuccessStyle
		case "error":
			statusStyle = theme.StatusErrorStyle
		case "warning":
			statusStyle = theme.StatusWarningStyle
		default:
			statusStyle = theme.StatusInfoStyle
		}
	}
	
	statusBar := statusStyle.Render(status)
	
	header := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			theme.SubtitleStyle.Render("Configure NLP processing components"),
			strings.Repeat(" ", m.width-50-lipgloss.Width(statusBar)),
			statusBar,
		),
		strings.Repeat(theme.BorderDividerH, m.width-4),
	)
	
	return header
}

func (m *PipelineModel) renderContent() string {
	leftWidth := m.width * 2 / 3
	rightWidth := m.width - leftWidth - 2
	
	// Component list
	componentList := m.renderComponentList(leftWidth-2, m.height-10)
	
	// Details panel
	detailsPanel := m.renderDetailsPanel(rightWidth-2, m.height-10)
	
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		componentList,
		lipgloss.NewStyle().Width(2).Render("  "),
		detailsPanel,
	)
}

func (m *PipelineModel) renderComponentList(width, height int) string {
	title := theme.RenderTitle("▥", "Pipeline Components")
	
	var content strings.Builder
	content.WriteString(title + "\n")
	content.WriteString(strings.Repeat(theme.BorderDividerH, width-4) + "\n\n")
	
	// Group components by category
	categories := []string{"Tokenization", "Core NLP", "Clinical", "Ontology", "Advanced"}
	
	for _, cat := range categories {
		// Category header
		catHeader := lipgloss.NewStyle().
			Foreground(theme.ColorSecondary).
			Bold(true).
			Render(fmt.Sprintf("▼ %s", cat))
		content.WriteString(catHeader + "\n")
		
		// Components in category
		for i, comp := range m.components {
			if comp.Category != cat {
				continue
			}
			
			selected := i == m.cursor
			
			// Status indicator
			var status string
			if comp.Required {
				status = lipgloss.NewStyle().
					Foreground(theme.ColorWarning).
					Render("[REQ]")
			} else if comp.Enabled {
				status = lipgloss.NewStyle().
					Foreground(theme.ColorSuccess).
					Render("[ON] ")
			} else {
				status = lipgloss.NewStyle().
					Foreground(theme.ColorForegroundDim).
					Render("[OFF]")
			}
			
			// Component name
			nameStyle := lipgloss.NewStyle().Foreground(theme.ColorForeground)
			if selected {
				nameStyle = nameStyle.
					Background(theme.ColorBackgroundLighter).
					Bold(true)
			}
			if !comp.Enabled && !comp.Required {
				nameStyle = nameStyle.Foreground(theme.ColorForegroundDim)
			}
			
			line := fmt.Sprintf("  %s %s %s", 
				status,
				comp.Icon,
				nameStyle.Width(width-16).Render(comp.Name))
				
			if selected {
				line = lipgloss.NewStyle().Foreground(theme.ColorAccent).Render(">") + line[1:]
			}
			
			content.WriteString(line + "\n")
		}
		content.WriteString("\n")
	}
	
	return theme.PanelActiveStyle.
		Width(width).
		Height(height).
		Render(content.String())
}

func (m *PipelineModel) renderDetailsPanel(width, height int) string {
	if m.cursor >= len(m.components) {
		return theme.PanelStyle.
			Width(width).
			Height(height).
			Render("")
	}
	
	comp := m.components[m.cursor]
	
	title := theme.RenderTitle(comp.Icon, "Component Details")
	
	var content strings.Builder
	content.WriteString(title + "\n")
	content.WriteString(strings.Repeat(theme.BorderDividerH, width-4) + "\n\n")
	
	// Component name
	nameStyle := lipgloss.NewStyle().
		Foreground(theme.ColorAccent).
		Bold(true)
	content.WriteString(nameStyle.Render(comp.Name) + "\n\n")
	
	// Description
	descStyle := lipgloss.NewStyle().
		Foreground(theme.ColorForeground).
		Width(width - 8)
	content.WriteString(descStyle.Render(comp.Description) + "\n\n")
	
	// Status
	content.WriteString(lipgloss.NewStyle().
		Foreground(theme.ColorForegroundDim).
		Render("Status: "))
		
	if comp.Required {
		content.WriteString(lipgloss.NewStyle().
			Foreground(theme.ColorWarning).
			Bold(true).
			Render("Required (Always Enabled)"))
	} else if comp.Enabled {
		content.WriteString(lipgloss.NewStyle().
			Foreground(theme.ColorSuccess).
			Bold(true).
			Render("Enabled"))
	} else {
		content.WriteString(lipgloss.NewStyle().
			Foreground(theme.ColorError).
			Render("Disabled"))
	}
	content.WriteString("\n\n")
	
	// Category
	content.WriteString(lipgloss.NewStyle().
		Foreground(theme.ColorForegroundDim).
		Render("Category: "))
	content.WriteString(lipgloss.NewStyle().
		Foreground(theme.ColorInfo).
		Render(comp.Category))
	content.WriteString("\n\n")
	
	// Configuration
	if len(comp.Config) > 0 {
		content.WriteString(lipgloss.NewStyle().
			Foreground(theme.ColorForegroundDim).
			Render("Configuration:\n"))
			
		for k, v := range comp.Config {
			content.WriteString(fmt.Sprintf("  • %s: %v\n", 
				lipgloss.NewStyle().Foreground(theme.ColorSecondary).Render(k),
				lipgloss.NewStyle().Foreground(theme.ColorForeground).Render(fmt.Sprintf("%v", v))))
		}
	}
	
	return theme.PanelStyle.
		Width(width).
		Height(height).
		Render(content.String())
}

func (m *PipelineModel) renderFooter() string {
	keys := []string{
		theme.RenderKeyHelp("↑↓", "Navigate"),
		theme.RenderKeyHelp("Space", "Toggle"),
		theme.RenderKeyHelp("s", "Save"),
		theme.RenderKeyHelp("r", "Reset"),
		theme.RenderKeyHelp("a", "Enable All"),
		theme.RenderKeyHelp("d", "Disable Optional"),
		theme.RenderKeyHelp("Esc", "Back"),
	}
	
	return theme.FooterStyle.
		Width(m.width).
		Render(lipgloss.JoinHorizontal(lipgloss.Left, keys...))
}

func (m *PipelineModel) saveConfiguration() tea.Cmd {
	return func() tea.Msg {
		// Simulate save delay
		// In real implementation, this would save to config file
		return configSavedMsg{}
	}
}

func (m *PipelineModel) resetToDefaults() {
	defaults := map[string]bool{
		"Sentence Detector":      true,
		"Tokenizer":             true,
		"Part-of-Speech Tagger": true,
		"Dependency Parser":     true,
		"Named Entity Recognizer": true,
		"Assertion Analyzer":    true,
		"Context Annotator":     true,
		"UMLS Lookup":          true,
		"Drug NER":             true,
		"Temporal Expression":   true,
	}
	
	for i := range m.components {
		if !m.components[i].Required {
			m.components[i].Enabled = defaults[m.components[i].Name]
		}
	}
}

type configSavedMsg struct{}

type pipelineKeyMap struct {
	Up              key.Binding
	Down            key.Binding
	Toggle          key.Binding
	Save            key.Binding
	Reset           key.Binding
	EnableAll       key.Binding
	DisableOptional key.Binding
	Back            key.Binding
}

var pipelineKeys = pipelineKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Toggle: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle"),
	),
	Save: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "save"),
	),
	Reset: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "reset"),
	),
	EnableAll: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "enable all"),
	),
	DisableOptional: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "disable optional"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "q"),
		key.WithHelp("esc", "back"),
	),
}