package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	maxMinutes = 99
	maxSeconds = 59
	tickRate   = time.Second
	circleSize = 12 // Height and width of the circle in characters
)

type model struct {
	totalTime   time.Duration
	elapsedTime time.Duration
	isRunning   bool
	isPaused    bool
}

type tickMsg time.Time

// Styling for the circle and text
var (
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")) // Red for remaining time
	whiteStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")) // White for elapsed time and timer
	circleStyle = lipgloss.NewStyle().Align(lipgloss.Center)
)

// parseDuration parses the input "mm:ss" into a time.Duration
func parseDuration(input string) (time.Duration, error) {
	parts := strings.Split(input, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid format, expected mm:ss")
	}

	minutes, err := strconv.Atoi(parts[0])
	if err != nil || minutes < 0 || minutes > maxMinutes {
		return 0, fmt.Errorf("minutes must be a number between 0 and %d", maxMinutes)
	}

	seconds, err := strconv.Atoi(parts[1])
	if err != nil || seconds < 0 || seconds > maxSeconds {
		return 0, fmt.Errorf("seconds must be a number between 0 and %d", maxSeconds)
	}

	return time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second, nil
}

// Initialize the model
func (m model) Init() tea.Cmd {
	return tickCmd() // Start ticking immediately
}

// Update the model based on messages
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "s":
			m.isRunning = false
			m.isPaused = false
			m.elapsedTime = 0
			return m, nil
		case "p", "r":
			if m.isRunning {
				m.isPaused = !m.isPaused
				if !m.isPaused {
					return m, tickCmd()
				}
			} else {
				m.isRunning = true
				m.isPaused = false
				return m, tickCmd()
			}
		}
	case tickMsg:
		if m.isRunning && !m.isPaused && m.elapsedTime < m.totalTime {
			m.elapsedTime += tickRate
			if m.elapsedTime >= m.totalTime {
				m.isRunning = false
				m.isPaused = false
				m.elapsedTime = m.totalTime // Ensure no rollover
			}
			if m.isRunning {
				return m, tickCmd()
			}
		}
	}
	return m, nil
}

// View renders the TUI
func (m model) View() string {
	// Calculate remaining time
	remaining := m.totalTime - m.elapsedTime
	if remaining < 0 {
		remaining = 0 // Prevent negative display
	}
	minutes := int(remaining.Minutes()) % 60
	seconds := int(remaining.Seconds()) % 60
	timer := fmt.Sprintf("%02d:%02d", minutes, seconds)

	// Calculate progress for the circle
	progress := 0.0
	if m.totalTime > 0 {
		progress = float64(m.elapsedTime) / float64(m.totalTime)
		if progress > 1.0 {
			progress = 1.0 // Cap progress at 100%
		}
	}

	// Draw the circle
	circle := drawCircle(progress)

	// Center the timer inside the circle
	lines := strings.Split(circle, "\n")
	timerLine := (circleSize - 1) / 2
	lines[timerLine] = centerText(timer, len(lines[timerLine]), whiteStyle)

	// Combine the circle with status text
	status := "\n [s]reset  [p]pause  [r]resume  [q]quit"
	if !m.isRunning {
		if m.elapsedTime >= m.totalTime {
			status = "Timer finished! \n [s]reset  [p]pause  [r]resume  [q]quit"
		} else {
			status = "Timer stopped. \n [s]reset  [p]pause  [r]resume  [q]quit"
		}
	} else if m.isPaused {
		status = "Timer paused. \n [s]reset  [p]pause  [r]resume  [q]quit"
	}

	return circleStyle.Render(strings.Join(lines, "\n") + "\n\n" + status)
}

// tickCmd sends a tick message every second
func tickCmd() tea.Cmd {
	return tea.Tick(tickRate, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// drawCircle creates a simple ASCII circle with a marker
func drawCircle(progress float64) string {
	// Simple circle representation using Unicode block characters
	lines := make([]string, circleSize)
	center := circleSize / 2
	radius := center - 1

	// Calculate the number of "segments" in the circle
	totalSegments := 40 // Arbitrary number of segments for the circle
	elapsedSegments := int(progress * float64(totalSegments))

	for y := 0; y < circleSize; y++ {
		line := ""
		for x := 0; x < circleSize*2; x++ {
			dx := float64(x - circleSize)
			dy := float64(y - center)
			distance := sqrt(dx*dx + dy*dy)

			// Draw thick circle (inner and outer radius)
			if distance >= float64(radius-1) && distance <= float64(radius) {
				// Calculate angle to determine if this is elapsed or remaining
				angle := (atan2(dy, dx) + 2*3.14159) / (2 * 3.14159) * float64(totalSegments)
				segment := int(angle) % totalSegments
				if segment < elapsedSegments {
					line += whiteStyle.Render("█")
				} else {
					line += redStyle.Render("█")
				}
			} else {
				line += " "
			}
		}
		lines[y] = line
	}

	return strings.Join(lines, "\n")
}

// centerText centers the text within a given width
func centerText(text string, width int, style lipgloss.Style) string {
	padding := (width - len(text)) / 2
	if padding < 0 {
		padding = 0
	}
	return strings.Repeat(" ", padding) + style.Render(text) + strings.Repeat(" ", width-padding-len(text))
}

// sqrt and atan2 for circle calculations
func sqrt(x float64) float64 {
	return float64(int(1000*x+0.5)) / 1000 // Simplified for integer-based rendering
}

func atan2(y, x float64) float64 {
	// Simplified atan2 for angle calculation
	if x == 0 {
		if y > 0 {
			return 3.14159 / 2
		} else if y < 0 {
			return -3.14159 / 2
		}
		return 0
	}
	return float64(int(1000*3.14159*x/y+0.5)) / 1000 // Approximation
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: gopomotime mm:ss")
		os.Exit(1)
	}

	duration, err := parseDuration(os.Args[1])
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	m := model{
		totalTime:   duration,
		elapsedTime: 0,
		isRunning:   true, // Start timer immediately
		isPaused:    false,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
