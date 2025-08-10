package components

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ctakes-tui/ctakes-tui/internal/theme"
	"github.com/ctakes-tui/ctakes-tui/internal/utils"
)

// ProgressBar - Reusable progress display component
type ProgressBar struct {
	// Progress data
	current    int64
	total      int64
	percentage float64

	// Configuration
	width       int
	showPercent bool
	showNumbers bool
	showRate    bool
	showETA     bool
	label       string

	// Animation
	indeterminate bool
	animFrame     int

	// Rate calculation
	startTime   time.Time
	lastUpdate  time.Time
	lastCurrent int64

	// Styling
	fillChar  string
	emptyChar string
	animated  bool
}

// NewProgressBar creates a new progress bar component
func NewProgressBar(total int64) *ProgressBar {
	return &ProgressBar{
		total:       total,
		width:       40,
		showPercent: true,
		showNumbers: true,
		fillChar:    "█",
		emptyChar:   "░",
		startTime:   time.Now(),
		lastUpdate:  time.Now(),
	}
}

// NewIndeterminateProgressBar creates a progress bar for unknown duration tasks
func NewIndeterminateProgressBar() *ProgressBar {
	return &ProgressBar{
		indeterminate: true,
		width:         40,
		fillChar:      "▶",
		emptyChar:     "─",
		animated:      true,
		startTime:     time.Now(),
		lastUpdate:    time.Now(),
	}
}

// Configuration methods
func (pb *ProgressBar) SetWidth(width int) *ProgressBar {
	pb.width = width
	return pb
}

func (pb *ProgressBar) SetLabel(label string) *ProgressBar {
	pb.label = label
	return pb
}

func (pb *ProgressBar) SetShowPercent(show bool) *ProgressBar {
	pb.showPercent = show
	return pb
}

func (pb *ProgressBar) SetShowNumbers(show bool) *ProgressBar {
	pb.showNumbers = show
	return pb
}

func (pb *ProgressBar) SetShowRate(show bool) *ProgressBar {
	pb.showRate = show
	return pb
}

func (pb *ProgressBar) SetShowETA(show bool) *ProgressBar {
	pb.showETA = show
	return pb
}

func (pb *ProgressBar) SetFillChar(char string) *ProgressBar {
	pb.fillChar = char
	return pb
}

func (pb *ProgressBar) SetEmptyChar(char string) *ProgressBar {
	pb.emptyChar = char
	return pb
}

func (pb *ProgressBar) SetAnimated(animated bool) *ProgressBar {
	pb.animated = animated
	return pb
}

// Progress methods
func (pb *ProgressBar) SetProgress(current, total int64) *ProgressBar {
	pb.lastCurrent = pb.current
	pb.current = current
	pb.total = total
	pb.lastUpdate = time.Now()

	if total > 0 {
		pb.percentage = float64(current) / float64(total) * 100
		if pb.percentage > 100 {
			pb.percentage = 100
		}
	}

	return pb
}

func (pb *ProgressBar) SetCurrent(current int64) *ProgressBar {
	return pb.SetProgress(current, pb.total)
}

func (pb *ProgressBar) SetTotal(total int64) *ProgressBar {
	return pb.SetProgress(pb.current, total)
}

func (pb *ProgressBar) SetPercentage(percentage float64) *ProgressBar {
	pb.percentage = percentage
	if percentage < 0 {
		pb.percentage = 0
	} else if percentage > 100 {
		pb.percentage = 100
	}

	if pb.total > 0 {
		pb.current = int64(pb.percentage / 100 * float64(pb.total))
	}

	pb.lastUpdate = time.Now()
	return pb
}

func (pb *ProgressBar) Increment(amount int64) *ProgressBar {
	return pb.SetCurrent(pb.current + amount)
}

func (pb *ProgressBar) IncrementOne() *ProgressBar {
	return pb.Increment(1)
}

// Animation methods (for indeterminate progress)
func (pb *ProgressBar) Tick() *ProgressBar {
	if pb.indeterminate && pb.animated {
		pb.animFrame++
		if pb.animFrame >= pb.width {
			pb.animFrame = 0
		}
	}
	return pb
}

// Status methods
func (pb *ProgressBar) IsComplete() bool {
	if pb.indeterminate {
		return false
	}
	return pb.current >= pb.total || pb.percentage >= 100
}

func (pb *ProgressBar) GetPercentage() float64 {
	return pb.percentage
}

func (pb *ProgressBar) GetCurrent() int64 {
	return pb.current
}

func (pb *ProgressBar) GetTotal() int64 {
	return pb.total
}

func (pb *ProgressBar) GetElapsedTime() time.Duration {
	return time.Since(pb.startTime)
}

func (pb *ProgressBar) GetRate() float64 {
	elapsed := time.Since(pb.startTime).Seconds()
	if elapsed <= 0 {
		return 0
	}
	return float64(pb.current) / elapsed
}

func (pb *ProgressBar) GetETA() time.Duration {
	if pb.indeterminate || pb.current <= 0 || pb.total <= 0 {
		return 0
	}

	rate := pb.GetRate()
	if rate <= 0 {
		return 0
	}

	remaining := pb.total - pb.current
	return time.Duration(float64(remaining)/rate) * time.Second
}

// Rendering
func (pb *ProgressBar) Render() string {
	lines := []string{}

	// Label
	if pb.label != "" {
		lines = append(lines, theme.RenderTextBold(pb.label))
	}

	// Progress bar
	barLine := pb.renderBar()
	lines = append(lines, barLine)

	// Status line
	statusLine := pb.renderStatus()
	if statusLine != "" {
		lines = append(lines, statusLine)
	}

	return strings.Join(lines, "\n")
}

func (pb *ProgressBar) renderBar() string {
	if pb.indeterminate {
		return pb.renderIndeterminateBar()
	}

	return pb.renderDeterminateBar()
}

func (pb *ProgressBar) renderDeterminateBar() string {
	barWidth := pb.width

	// Calculate fill width
	fillWidth := int(math.Round(pb.percentage / 100.0 * float64(barWidth)))
	if fillWidth > barWidth {
		fillWidth = barWidth
	}

	// Build bar
	bar := ""

	// Fill portion
	for i := 0; i < fillWidth; i++ {
		bar += pb.fillChar
	}

	// Empty portion
	for i := fillWidth; i < barWidth; i++ {
		bar += pb.emptyChar
	}

	// Add border if there's room
	if pb.width > 20 {
		bar = "│" + bar + "│"
	}

	return theme.RenderText(bar)
}

func (pb *ProgressBar) renderIndeterminateBar() string {
	barWidth := pb.width
	bar := strings.Repeat(pb.emptyChar, barWidth)

	// Add moving indicator
	if pb.animated {
		indicatorWidth := 3
		start := pb.animFrame % (barWidth + indicatorWidth)

		// Draw indicator
		for i := 0; i < indicatorWidth && start+i < barWidth; i++ {
			if start+i >= 0 {
				barRunes := []rune(bar)
				if start+i < len(barRunes) {
					barRunes[start+i] = []rune(pb.fillChar)[0]
					bar = string(barRunes)
				}
			}
		}
	}

	// Add border if there's room
	if pb.width > 20 {
		bar = "│" + bar + "│"
	}

	return theme.RenderText(bar)
}

func (pb *ProgressBar) renderStatus() string {
	if pb.indeterminate {
		return pb.renderIndeterminateStatus()
	}

	return pb.renderDeterminateStatus()
}

func (pb *ProgressBar) renderDeterminateStatus() string {
	parts := []string{}

	// Percentage
	if pb.showPercent {
		parts = append(parts, fmt.Sprintf("%.1f%%", pb.percentage))
	}

	// Numbers
	if pb.showNumbers {
		if pb.total > 0 {
			parts = append(parts, fmt.Sprintf("%s / %s",
				utils.FormatNumber(pb.current),
				utils.FormatNumber(pb.total)))
		} else {
			parts = append(parts, utils.FormatNumber(pb.current))
		}
	}

	// Rate
	if pb.showRate {
		rate := pb.GetRate()
		if rate > 0 {
			parts = append(parts, fmt.Sprintf("%.1f/s", rate))
		}
	}

	// ETA
	if pb.showETA {
		eta := pb.GetETA()
		if eta > 0 {
			parts = append(parts, fmt.Sprintf("ETA: %s", utils.FormatDurationShort(eta)))
		}
	}

	if len(parts) == 0 {
		return ""
	}

	status := strings.Join(parts, "  ")
	return theme.RenderTextDim(status)
}

func (pb *ProgressBar) renderIndeterminateStatus() string {
	parts := []string{}

	// Elapsed time
	elapsed := pb.GetElapsedTime()
	parts = append(parts, fmt.Sprintf("Elapsed: %s", utils.FormatDurationShort(elapsed)))

	// Rate (if we have current progress)
	if pb.showRate && pb.current > 0 {
		rate := pb.GetRate()
		if rate > 0 {
			parts = append(parts, fmt.Sprintf("%.1f/s", rate))
		}
	}

	// Current count
	if pb.showNumbers && pb.current > 0 {
		parts = append(parts, utils.FormatNumber(pb.current))
	}

	if len(parts) == 0 {
		return ""
	}

	status := strings.Join(parts, "  ")
	return theme.RenderTextDim(status)
}

// RenderCompact renders a single-line progress bar
func (pb *ProgressBar) RenderCompact() string {
	if pb.indeterminate {
		return pb.renderIndeterminateBar()
	}

	bar := pb.renderDeterminateBar()

	// Add percentage if there's room
	if pb.showPercent && pb.width > 15 {
		percentText := fmt.Sprintf(" %.0f%%", pb.percentage)
		bar += percentText
	}

	return bar
}

// Utility functions removed - now using common utilities

// Preset configurations
func NewFileTransferProgressBar(totalBytes int64) *ProgressBar {
	return NewProgressBar(totalBytes).
		SetLabel("Transfer Progress").
		SetShowNumbers(true).
		SetShowPercent(true).
		SetShowRate(true).
		SetShowETA(true).
		SetFillChar("█").
		SetEmptyChar("▒")
}

func NewTaskProgressBar(totalTasks int64, taskName string) *ProgressBar {
	return NewProgressBar(totalTasks).
		SetLabel(taskName + " Progress").
		SetShowNumbers(true).
		SetShowPercent(true).
		SetShowETA(true).
		SetFillChar("▆").
		SetEmptyChar("▁")
}

func NewSpinnerProgressBar(label string) *ProgressBar {
	return NewIndeterminateProgressBar().
		SetLabel(label).
		SetAnimated(true).
		SetFillChar("▶").
		SetEmptyChar("─").
		SetWidth(30)
}
