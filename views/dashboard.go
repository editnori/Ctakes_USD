package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DashboardView struct {
	width           int
	height          int
	spinner         spinner.Model
	progress        progress.Model
	
	// Stats
	documentsProcessed int
	entitiesExtracted  int
	activeConnections  int
	memoryUsage       string
	cpuUsage          string
	lastProcessTime   string
	queueSize         int
	
	// Recent activity
	recentFiles []string
	recentQueries []string
}

func NewDashboardView() DashboardView {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	
	p := progress.New(progress.WithDefaultGradient())
	
	
	return DashboardView{
		spinner:         s,
		progress:        p,
		memoryUsage:     "2.1 GB",
		cpuUsage:        "12%",
		lastProcessTime: "N/A",
		recentFiles:     []string{"sample1.txt", "patient_notes.pdf", "clinical_report.doc"},
		recentQueries:   []string{"hypertension medication", "diabetes symptoms", "lung cancer treatment"},
	}
}

func (d DashboardView) Init() tea.Cmd {
	return tea.Batch(
		d.spinner.Tick,
		d.tickStats(),
	)
}

func (d *DashboardView) tickStats() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return statsUpdateMsg{}
	})
}

type statsUpdateMsg struct{}

func (d DashboardView) Update(msg tea.Msg) (DashboardView, tea.Cmd) {
	var cmds []tea.Cmd
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height
		d.progress.Width = msg.Width / 3
		
	case tea.KeyMsg:
		
	case statsUpdateMsg:
		// Simulate stats updates
		d.documentsProcessed++
		d.entitiesExtracted += 47
		cmds = append(cmds, d.tickStats())
		
	case spinner.TickMsg:
		var cmd tea.Cmd
		d.spinner, cmd = d.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}
	
	
	return d, tea.Batch(cmds...)
}

func (d DashboardView) View() string {
	var b strings.Builder
	
	// ASCII Banner
	banner := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true).
		Render(`
   ██████╗████████╗ █████╗ ██╗  ██╗███████╗███████╗
  ██╔════╝╚══██╔══╝██╔══██╗██║ ██╔╝██╔════╝██╔════╝
  ██║        ██║   ███████║█████╔╝ █████╗  ███████╗
  ██║        ██║   ██╔══██║██╔═██╗ ██╔══╝  ╚════██║
  ╚██████╗   ██║   ██║  ██║██║  ██╗███████╗███████║
   ╚═════╝   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝╚══════╝
     Clinical Text Analysis & Knowledge Extraction System`)
	
	b.WriteString(banner)
	b.WriteString("\n\n")
	
	// Create grid layout
	leftPanel := d.renderSystemStatus()
	middlePanel := d.renderProcessingQueue()
	rightPanel := d.renderRecentActivity()
	
	// Use lipgloss to create columns
	columns := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPanel,
		"  ",
		middlePanel,
		"  ",
		rightPanel,
	)
	
	b.WriteString(columns)
	b.WriteString("\n\n")
	
	// Bottom stats bar
	statsBar := d.renderStatsBar()
	b.WriteString(statsBar)
	
	// Footer
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("\n1-3 Quick Actions • ESC Main Menu")
	b.WriteString(footer)
	
	return b.String()
}

func (d DashboardView) renderSystemStatus() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("86")).
		Padding(1).
		Width(35)
	
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render("■ System Status")
	
	content := fmt.Sprintf(`%s

Connection: %s
Memory: %s
CPU: %s
Active Pipelines: %d
Queue Size: %d

Last Process: %s`,
		title,
		d.renderConnectionStatus(),
		d.memoryUsage,
		d.cpuUsage,
		d.activeConnections,
		d.queueSize,
		d.lastProcessTime,
	)
	
	return style.Render(content)
}

func (d DashboardView) renderConnectionStatus() string {
	if d.activeConnections > 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Render("● Connected")
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("9")).
		Render("● Disconnected")
}

func (d DashboardView) renderProcessingQueue() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("86")).
		Padding(1).
		Width(40)
	
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render("■ Processing Pipeline")
	
	progressBar := d.progress.ViewAs(0.3)
	
	content := fmt.Sprintf(`%s

Current Task: %s
%s

Documents: %d processed
Entities: %d extracted

Pipeline Components:
✓ Sentence Detector
✓ Tokenizer
✓ POS Tagger
● Dictionary Lookup %s
○ Assertion Module
○ Drug NER`,
		title,
		"Analyzing clinical_notes.txt",
		progressBar,
		d.documentsProcessed,
		d.entitiesExtracted,
		d.spinner.View(),
	)
	
	return style.Render(content)
}

func (d DashboardView) renderRecentActivity() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("86")).
		Padding(1).
		Width(35)
	
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render("■ Recent Activity")
	
	var files strings.Builder
	for i, f := range d.recentFiles {
		if i >= 3 {
			break
		}
		files.WriteString(fmt.Sprintf("  → %s\n", f))
	}
	
	var queries strings.Builder
	for i, q := range d.recentQueries {
		if i >= 3 {
			break
		}
		queries.WriteString(fmt.Sprintf("  → %s\n", q))
	}
	
	content := fmt.Sprintf(`%s

Recent Files:
%s
Recent Queries:
%s
Quick Actions:
[1] Process New
[2] View Results
[3] Configure`,
		title,
		files.String(),
		queries.String(),
	)
	
	return style.Render(content)
}

func (d DashboardView) renderStatsBar() string {
	style := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("252")).
		Padding(0, 1)
	
	stats := fmt.Sprintf(
		"Documents: %d | Entities: %d | Memory: %s | CPU: %s | Uptime: %s",
		d.documentsProcessed,
		d.entitiesExtracted,
		d.memoryUsage,
		d.cpuUsage,
		"2h 14m",
	)
	
	return style.Render(stats)
}