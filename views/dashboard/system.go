package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ctakes-tui/ctakes-tui/internal/theme"
	"github.com/ctakes-tui/ctakes-tui/internal/utils"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

func (m *Model) updateSystemInfo() {
	cpuPercent, _ := cpu.Percent(0, false)
	if len(cpuPercent) > 0 {
		m.cpuPercent = cpuPercent[0]
	}

	m.cpuCores, _ = cpu.Counts(true)

	if vmStat, err := mem.VirtualMemory(); err == nil {
		m.memPercent = vmStat.UsedPercent
		m.totalMem = vmStat.Total
		m.usedMem = vmStat.Used
	}

	if diskStat, err := disk.Usage("/"); err == nil {
		m.diskPercent = diskStat.UsedPercent
		m.totalDisk = diskStat.Total
		m.usedDisk = diskStat.Used
	}

	processes, _ := process.Processes()
	m.processes = []ProcessInfo{}

	for _, p := range processes {
		if len(m.processes) >= 10 {
			break
		}

		name, _ := p.Name()
		cpuP, _ := p.CPUPercent()
		memP, _ := p.MemoryPercent()
		status, _ := p.Status()

		if cpuP > 0.1 || memP > 0.1 {
			m.processes = append(m.processes, ProcessInfo{
				PID:    p.Pid,
				Name:   name,
				CPU:    cpuP,
				Memory: memP,
				Status: status[0],
			})
		}
	}
}

func (m *Model) renderSystemPanel(width, height int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorPrimary).
		MarginBottom(1)

	title := titleStyle.Render("System Monitor")

	cpuBar := m.renderStatBar("CPU", m.cpuPercent, width-4, theme.ColorAccent)
	memBar := m.renderStatBar("Memory", m.memPercent, width-4, theme.ColorWarning)
	diskBar := m.renderStatBar("Disk", m.diskPercent, width-4, theme.ColorSuccess)

	memInfo := fmt.Sprintf("Memory: %s / %s",
		utils.FormatFileSize(int64(m.usedMem)),
		utils.FormatFileSize(int64(m.totalMem)))

	diskInfo := fmt.Sprintf("Disk: %s / %s",
		utils.FormatFileSize(int64(m.usedDisk)),
		utils.FormatFileSize(int64(m.totalDisk)))

	infoStyle := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		MarginTop(1)

	info := infoStyle.Render(fmt.Sprintf("%s | %s | %d CPU cores", memInfo, diskInfo, m.cpuCores))

	processTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorPrimary).
		MarginTop(2).
		MarginBottom(1).
		Render("Top Processes")

	var processList strings.Builder
	for _, p := range m.processes {
		processList.WriteString(fmt.Sprintf("%-20s CPU: %5.1f%% MEM: %5.1f%%\n",
			utils.TruncateString(p.Name, 20), p.CPU, p.Memory))
	}

	processStyle := lipgloss.NewStyle().
		Foreground(theme.ColorText).
		Width(width - 4)

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		cpuBar,
		memBar,
		diskBar,
		info,
		processTitle,
		processStyle.Render(processList.String()),
	)
}

func (m *Model) renderStatBar(label string, percent float64, width int, color lipgloss.Color) string {
	barWidth := width - len(label) - 10
	filled := int(float64(barWidth) * percent / 100)

	if filled > barWidth {
		filled = barWidth
	}
	if filled < 0 {
		filled = 0
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	barStyle := lipgloss.NewStyle().Foreground(color)
	labelStyle := lipgloss.NewStyle().Foreground(theme.ColorText)
	percentStyle := lipgloss.NewStyle().Foreground(theme.ColorSecondary)

	return fmt.Sprintf("%s %s %s",
		labelStyle.Render(fmt.Sprintf("%-6s", label)),
		barStyle.Render(bar),
		percentStyle.Render(fmt.Sprintf("%5.1f%%", percent)))
}
