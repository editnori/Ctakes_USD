package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DashboardView struct {
	width        int
	height       int
	choices      []string
	cursor       int
	ctakesStatus string
}

func NewDashboardView() DashboardView {
	return DashboardView{
		choices: []string{
			"▪ Process Documents",
			"▪ Analyze Text",
			"▪ Configure Pipeline",
			"▪ Exit",
		},
		ctakesStatus: "Not Connected",
	}
}

func (d DashboardView) Init() tea.Cmd {
	return nil
}

func (d DashboardView) Update(msg tea.Msg) (DashboardView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height
		
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if d.cursor > 0 {
				d.cursor--
			}
		case "down", "j":
			if d.cursor < len(d.choices)-1 {
				d.cursor++
			}
		}
	}
	
	return d, nil
}

func (d DashboardView) View() string {
	var b strings.Builder
	
	// ASCII Banner
	banner := lipgloss.NewStyle().
		Foreground(lipgloss.Color("51")).
		Bold(true).
		Render(`
   ██████╗████████╗ █████╗ ██╗  ██╗███████╗███████╗     ██████╗██╗     ██╗
  ██╔════╝╚══██╔══╝██╔══██╗██║ ██╔╝██╔════╝██╔════╝    ██╔════╝██║     ██║
  ██║        ██║   ███████║█████╔╝ █████╗  ███████╗    ██║     ██║     ██║
  ██║        ██║   ██╔══██║██╔═██╗ ██╔══╝  ╚════██║    ██║     ██║     ██║
  ╚██████╗   ██║   ██║  ██║██║  ██╗███████╗███████║    ╚██████╗███████╗██║
   ╚═════╝   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝╚══════╝     ╚═════╝╚══════╝╚═╝`)
	
	subtitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true).
		Render("     Clinical Text Analysis & Knowledge Extraction System")
	
	author := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Render("                      by Dr. Layth M Qassem")
	
	b.WriteString(banner)
	b.WriteString("\n")
	b.WriteString(subtitle)
	b.WriteString("\n")
	b.WriteString(author)
	b.WriteString("\n\n")
	
	// Status
	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Padding(0, 2)
	
	statusIndicator := "●"
	statusColor := "196"
	if d.ctakesStatus == "Connected" {
		statusColor = "46"
	}
	
	status := fmt.Sprintf("%s Status: %s %s",
		statusStyle.Render("System"),
		lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render(statusIndicator),
		d.ctakesStatus)
	
	b.WriteString(lipgloss.NewStyle().Padding(0, 2).Render(status))
	b.WriteString("\n\n")
	
	// Menu
	menuStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("51")).
		Padding(1, 2).
		Width(50).
		Align(lipgloss.Center)
	
	menuTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("51")).
		Render("Main Menu")
	
	menu := menuTitle + "\n\n"
	for i, choice := range d.choices {
		cursor := "  "
		if d.cursor == i {
			cursor = "→"
			choice = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("214")).
				Render(choice)
		} else {
			choice = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Render(choice)
		}
		menu += fmt.Sprintf("%s %s\n", cursor, choice)
	}
	
	centeredMenu := lipgloss.Place(
		d.width,
		10,
		lipgloss.Center,
		lipgloss.Center,
		menuStyle.Render(menu),
	)
	
	b.WriteString(centeredMenu)
	
	// Footer
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Width(d.width).
		Align(lipgloss.Center).
		Render("\n↑↓ Navigate • ⏎ Select • Ctrl+C Quit")
	
	b.WriteString("\n")
	b.WriteString(footer)
	
	return b.String()
}

func (d DashboardView) GetCursor() int {
	return d.cursor
}