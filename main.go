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

	// For key highlight feedback
	highlightKey   string
	highlightUntil time.Time
}

type tickMsg time.Time
type blinkMsg time.Time

// Styling for the circle and text
var (
	redStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")) // Red for remaining time
	whiteStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")) // White for elapsed time and timer
	greenStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")) // Green for finished text
	circleStyle    = lipgloss.NewStyle()                                       // No center alignment
	highlightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFa400")) // Orange highlight
)

// Highlight duration for key feedback
const highlightDuration = 500 * time.Millisecond

type highlightMsg struct{}

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
		now := time.Now()
		switch msg.String() {
		case "q":
			m.highlightKey = "q"
			m.highlightUntil = now.Add(highlightDuration)
			return m, tea.Batch(tea.Tick(highlightDuration, func(t time.Time) tea.Msg { return highlightMsg{} }), tea.Quit)
		case "r":
			m.isRunning = true
			m.isPaused = false
			m.elapsedTime = 0
			m.highlightKey = "r"
			m.highlightUntil = now.Add(highlightDuration)
			return m, tea.Batch(tea.Tick(highlightDuration, func(t time.Time) tea.Msg { return highlightMsg{} }), tickCmd())
		case "p":
			m.highlightKey = "p"
			m.highlightUntil = now.Add(highlightDuration)
			if m.isRunning {
				m.isPaused = !m.isPaused
				if !m.isPaused {
					return m, tea.Batch(tea.Tick(highlightDuration, func(t time.Time) tea.Msg { return highlightMsg{} }), tickCmd())
				}
			} else {
				m.isRunning = true
				m.isPaused = false
				return m, tea.Batch(tea.Tick(highlightDuration, func(t time.Time) tea.Msg { return highlightMsg{} }), tickCmd())
			}
			return m, tea.Tick(highlightDuration, func(t time.Time) tea.Msg { return highlightMsg{} })
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
	case highlightMsg:
		m.highlightKey = ""
		m.highlightUntil = time.Time{}
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

	// Center status text within 29-column width, with highlight if needed
	statusLines := strings.Split(status, "\n")
	for i, line := range statusLines {
		if !(i == 0 && !m.isRunning && m.elapsedTime >= m.totalTime) { // Skip finishedText, already padded
			// Highlight logic
			if m.highlightKey != "" && time.Now().Before(m.highlightUntil) {
				if m.highlightKey == "q" && strings.Contains(line, "[q]uit") {
					// Replace with highlighted, but pad to same width
					h := highlightStyle.Render("[q]uit")
					pad := len("[q]uit") - len([]rune("[q]uit")) + len([]rune(h)) - len(h)
					line = strings.Replace(line, "[q]uit", h+strings.Repeat(" ", pad), 1)
				} else if m.highlightKey == "r" && strings.Contains(line, "[r]eset") {
					h := highlightStyle.Render("[r]eset")
					pad := len("[r]eset") - len([]rune("[r]eset")) + len([]rune(h)) - len(h)
					line = strings.Replace(line, "[r]eset", h+strings.Repeat(" ", pad), 1)
				} else if m.highlightKey == "p" {
					if strings.Contains(line, "[p]ause") {
						h := highlightStyle.Render("[p]ause")
						pad := len("[p]ause") - len([]rune("[p]ause")) + len([]rune(h)) - len(h)
						line = strings.Replace(line, "[p]ause", h+strings.Repeat(" ", pad), 1)
					}
					if strings.Contains(line, "un[p]ause") {
						h := highlightStyle.Render("un[p]ause")
						pad := len("un[p]ause") - len([]rune("un[p]ause")) + len([]rune(h)) - len(h)
						line = strings.Replace(line, "un[p]ause", h+strings.Repeat(" ", pad), 1)
					}
				}
			}
			// Always center using the original line length (without color codes)
			plainLine := stripANSI(line)
			padding := (width - len([]rune(strings.TrimSpace(plainLine)))) / 2
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

// stripANSI removes ANSI escape codes for accurate width calculation
func stripANSI(str string) string {
	in := false
	out := make([]rune, 0, len(str))
	for _, r := range str {
		if r == 27 { // ESC
			in = true
			continue
		}
		if in {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				in = false
			}
			continue
		}
		out = append(out, r)
	}
	return string(out)
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
