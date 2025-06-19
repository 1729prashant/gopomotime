package main

import (
	"fmt"
	"math"
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
	blinkRate  = 800 * time.Millisecond
	height     = 13 // Donut height
	width      = 29 // Donut width
)

type model struct {
	totalTime   time.Duration
	elapsedTime time.Duration
	isRunning   bool
	isPaused    bool
	blink       bool // For blinking effect
}

type tickMsg time.Time
type blinkMsg time.Time

// Styling for the circle and text
var (
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")) // Red for remaining time
	whiteStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")) // White for elapsed time and timer
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")) // Green for finished text
	circleStyle = lipgloss.NewStyle()                                       // No center alignment
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
	return tea.Batch(tickCmd(), blinkCmd()) // Start ticking and blinking
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
		case "r":
			m.isRunning = true
			m.isPaused = false
			m.elapsedTime = 0
			return m, tickCmd() // Restart immediately
		case "p":
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
		return m, blinkCmd() // Continue blinking when finished
	case blinkMsg:
		m.blink = !m.blink
		if !m.isRunning && m.elapsedTime >= m.totalTime {
			return m, blinkCmd() // Keep blinking only when finished
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

	// Draw the ASCII donut
	circle := drawCircle(progress, timer)

	// Combine the circle with status text
	var status string
	if !m.isRunning {
		if m.elapsedTime >= m.totalTime {
			// Apply green and blinking to "Timer finished!" only
			finishedText := "Timer finished!"
			padding := (width - len(finishedText)) / 2 // 7 spaces
			if m.blink {
				finishedText = strings.Repeat(" ", padding) + greenStyle.Render(finishedText) + strings.Repeat(" ", padding)
			} else {
				finishedText = strings.Repeat(" ", width) // Full width blank for centering
			}
			controls := " [q]uit [r]eset [p]ause"
			controlsPadding := (width - len(controls)) / 2 // 4 spaces
			controls = strings.Repeat(" ", controlsPadding) + controls
			status = finishedText + "\n" + controls
		} else {
			status = "Timer stopped. \n    [q]uit [r]eset [p]ause"
		}
	} else if m.isPaused {
		status = "Timer paused. \n    [q]uit [r]eset un[p]ause"
	} else {
		status = " \n    [q]uit [r]eset [p]ause"
	}

	// Center status text within 29-column width
	statusLines := strings.Split(status, "\n")
	for i, line := range statusLines {
		if !(i == 0 && !m.isRunning && m.elapsedTime >= m.totalTime) { // Skip finishedText, already padded
			padding := (width - len(strings.TrimSpace(line))) / 2
			if padding < 0 {
				padding = 0
			}
			statusLines[i] = strings.Repeat(" ", padding) + line
		}
	}
	centeredStatus := strings.Join(statusLines, "\n")

	// Add left padding to shift entire block left
	leftPadding := strings.Repeat(" ", 4)
	output := strings.Join(strings.Split(circle, "\n"), "\n"+leftPadding) + "\n" + leftPadding + centeredStatus
	return circleStyle.Render(leftPadding + output)
}

// tickCmd sends a tick message every second
func tickCmd() tea.Cmd {
	return tea.Tick(tickRate, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// blinkCmd sends a blink message every 500ms
func blinkCmd() tea.Cmd {
	return tea.Tick(blinkRate, func(t time.Time) tea.Msg {
		return blinkMsg(t)
	})
}

// drawCircle creates a 13x29 ASCII donut with progress and timer
func drawCircle(progress float64, timer string) string {
	// Provided 13x29 ASCII donut template
	donutTemplate := []string{
		"          *********          ",
		"      *****************      ",
		"    *********************    ",
		"  **********     **********  ",
		" ********           ******** ",
		" ******               ****** ",
		" ******     mm:ss     ****** ",
		" ******               ****** ",
		" *******             ******* ",
		"  **********     **********  ",
		"    *********************    ",
		"      *****************      ",
		"          *********          ",
	}

	centerX, centerY := float64(width/2), float64(height/2) // 14.5, 6.5
	totalSegments := 120                                    // Smooth progress
	lines := make([]string, height)

	for y := 0; y < height; y++ {
		line := ""
		timerStart := (width - len(timer)) / 2 // 12 for timer length 5
		timerEnd := timerStart + len(timer)    // 17
		for x, char := range donutTemplate[y] {
			if char == '*' {
				// Calculate angle for progress marker (0 at 12 o'clock, clockwise)
				dx := float64(x) - centerX
				dy := float64(y) - centerY
				angle := math.Atan2(dy, dx) + math.Pi/2 // Start at 12 o'clock
				if angle < 0 {
					angle += 2 * math.Pi
				}
				segment := int((angle / (2 * math.Pi)) * float64(totalSegments))
				if segment < int(progress*float64(totalSegments)) {
					line += whiteStyle.Render("*")
				} else {
					line += redStyle.Render("*")
				}
			} else if y == height/2 && x >= timerStart && x < timerEnd {
				// Place the actual timer in the center
				line += whiteStyle.Render(string(timer[x-timerStart]))
			} else {
				line += " "
			}
		}
		lines[y] = line
	}

	return strings.Join(lines, "\n")
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
		blink:       true, // Start with text visible
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
