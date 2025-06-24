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
	tickRate   = 120 * time.Millisecond // ~30 FPS for smooth progress
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
	highlightKey       string
	highlightUntil     time.Time
	pendingPauseToggle bool // If true, toggle pause on highlightMsg

	// Add a new field to model to track the start time for smooth progress
	startTime time.Time
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
const highlightDuration = 150 * time.Millisecond

type highlightMsg struct{}

// parseDuration parses the input string in "mm:ss" format into a time.Duration.
// Returns an error if the format is invalid or out of bounds.
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

// Init initializes the Bubble Tea model, starting the tick and blink commands.
func (m model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), blinkCmd()) // Start ticking and blinking
}

// Update handles all messages (key presses, ticks, blinks, highlight timeouts) and updates the model state accordingly.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle key presses
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
		now := time.Now()
		switch msg.String() {
		case "q":
			// Highlight [q]uit and quit after highlightDuration
			m.highlightKey = "q"
			m.highlightUntil = now.Add(highlightDuration)
			return m, tea.Batch(tea.Tick(highlightDuration, func(t time.Time) tea.Msg { return highlightMsg{} }), tea.Quit)
		case "r":
			// Highlight [r]eset and reset timer
			wasRunning := m.isRunning && !m.isPaused
			m.isRunning = true
			m.isPaused = false
			m.elapsedTime = 0
			m.startTime = time.Now() // Reset start time for smooth progress
			m.highlightKey = "r"
			m.highlightUntil = now.Add(highlightDuration)
			if wasRunning {
				return m, tea.Tick(highlightDuration, func(t time.Time) tea.Msg { return highlightMsg{} })
			} else {
				return m, tea.Batch(tea.Tick(highlightDuration, func(t time.Time) tea.Msg { return highlightMsg{} }), tickCmd())
			}
		case "p":
			// Highlight [p]ause or un[p]ause and toggle pause state after delay
			m.highlightKey = "p"
			m.highlightUntil = now.Add(highlightDuration)
			m.pendingPauseToggle = true
			return m, tea.Tick(highlightDuration, func(t time.Time) tea.Msg { return highlightMsg{} })
		}
	case tickMsg:
		// Handle timer tick for smooth progress
		if m.isRunning && !m.isPaused && m.elapsedTime < m.totalTime {
			// Use wall clock time for smooth progress
			now := time.Now()
			m.elapsedTime = now.Sub(m.startTime)
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
		// Handle blinking for finished timer
		m.blink = !m.blink
		if !m.isRunning && m.elapsedTime >= m.totalTime {
			return m, blinkCmd() // Keep blinking only when finished
		}
	case highlightMsg:
		// Clear highlight after duration
		m.highlightKey = ""
		m.highlightUntil = time.Time{}
		// If a pause toggle is pending, perform it now
		if m.pendingPauseToggle {
			if m.isRunning {
				m.isPaused = !m.isPaused
				if !m.isPaused {
					// When unpausing, adjust startTime so elapsedTime is continuous
					m.startTime = time.Now().Add(-m.elapsedTime)
					m.pendingPauseToggle = false
					return m, tickCmd()
				}
			} else {
				m.isRunning = true
				m.isPaused = false
				m.startTime = time.Now().Add(-m.elapsedTime)
			}
			m.pendingPauseToggle = false
		}
	}
	return m, nil
}

// View renders the TUI, including the donut, timer, and status/controls, with proper centering and highlighting.
func (m model) View() string {
	// Calculate remaining time for the timer
	remaining := m.totalTime - m.elapsedTime
	if remaining < 0 {
		remaining = 0 // Prevent negative display
	}
	minutes := int(remaining.Minutes()) % 60
	seconds := int(remaining.Seconds()) % 60
	timer := fmt.Sprintf("%02d:%02d", minutes, seconds)

	// Calculate progress for the donut (0.0 to 1.0), use wall clock for smoothness
	progress := 0.0
	if m.totalTime > 0 {
		progress = float64(m.elapsedTime) / float64(m.totalTime)
		if progress > 1.0 {
			progress = 1.0 // Cap progress at 100%
		}
	}

	// Draw the ASCII donut with progress and timer
	circle := drawCircle(progress, timer)

	// Build the status/control text block
	var status string
	if !m.isRunning {
		if m.elapsedTime >= m.totalTime {
			// Timer finished: show blinking green message and controls
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
			// Timer stopped: show stopped message and controls
			status = "Timer stopped. \n    [q]uit [r]eset [p]ause"
		}
	} else if m.isPaused {
		// Timer paused: show paused message and controls
		status = "Timer paused. \n    [q]uit [r]eset un[p]ause"
	} else {
		// Timer running: show only controls
		status = " \n    [q]uit [r]eset [p]ause"
	}

	// Center status text within 29-column width, with highlight if needed
	statusLines := strings.Split(status, "\n")
	for i, line := range statusLines {
		// Only center and highlight control/status lines, not the blinking finished text (already padded)
		if !(i == 0 && !m.isRunning && m.elapsedTime >= m.totalTime) {
			// Highlight the relevant key if pressed recently
			if m.highlightKey != "" && time.Now().Before(m.highlightUntil) {
				if m.highlightKey == "q" && strings.Contains(line, "[q]uit") {
					// Highlight [q]uit, pad to same width
					h := highlightStyle.Render("[q]uit")
					pad := len("[q]uit") - len([]rune("[q]uit")) + len([]rune(h)) - len(h)
					line = strings.Replace(line, "[q]uit", h+strings.Repeat(" ", pad), 1)
				} else if m.highlightKey == "r" && strings.Contains(line, "[r]eset") {
					// Highlight [r]eset, pad to same width
					h := highlightStyle.Render("[r]eset")
					pad := len("[r]eset") - len([]rune("[r]eset")) + len([]rune(h)) - len(h)
					line = strings.Replace(line, "[r]eset", h+strings.Repeat(" ", pad), 1)
				} else if m.highlightKey == "p" {
					// Highlight [p]ause and/or un[p]ause, pad to same width
					if strings.Contains(line, "[p]ause") && !strings.Contains(line, "un[p]ause") {
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
			// Center the line using the printable width (strip ANSI codes)
			plainLine := stripANSI(line)
			padding := (width - len([]rune(strings.TrimSpace(plainLine)))) / 2
			if padding < 0 {
				padding = 0
			}
			statusLines[i] = strings.Repeat(" ", padding) + line
		}
	}
	centeredStatus := strings.Join(statusLines, "\n")

	// Add left padding to shift entire block left for donut and status
	leftPadding := strings.Repeat(" ", 4)
	output := strings.Join(strings.Split(circle, "\n"), "\n"+leftPadding) + "\n" + leftPadding + centeredStatus
	return circleStyle.Render(leftPadding + output)
}

// tickCmd returns a Bubble Tea command that sends a tickMsg every second.
func tickCmd() tea.Cmd {
	return tea.Tick(tickRate, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// blinkCmd returns a Bubble Tea command that sends a blinkMsg every blinkRate interval.
func blinkCmd() tea.Cmd {
	return tea.Tick(blinkRate, func(t time.Time) tea.Msg {
		return blinkMsg(t)
	})
}

// drawCircle creates a 13x29 ASCII donut with progress and timer in the center.
// The donut fills clockwise as time elapses.
func drawCircle(progress float64, timer string) string {
	// ASCII donut template, 13 rows x 29 columns
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

	centerX, centerY := float64(width/2), float64(height/2) // Center of donut
	totalSegments := 120                                    // Number of progress segments for smoothness
	lines := make([]string, height)

	// Loop over each row of the donut
	for y := 0; y < height; y++ {
		line := ""
		timerStart := (width - len(timer)) / 2 // Center timer horizontally
		timerEnd := timerStart + len(timer)
		// Loop over each character in the row
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
				// Fill with white for elapsed, red for remaining
				if segment < int(progress*float64(totalSegments)) {
					line += whiteStyle.Render("*")
				} else {
					line += redStyle.Render("*")
				}
			} else if y == height/2 && x >= timerStart && x < timerEnd {
				// Place the actual timer in the center row
				line += whiteStyle.Render(string(timer[x-timerStart]))
			} else {
				line += " "
			}
		}
		lines[y] = line
	}

	return strings.Join(lines, "\n")
}

// stripANSI removes ANSI escape codes for accurate width calculation when centering highlighted text.
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

// main is the entry point. It parses arguments, initializes the model, and runs the Bubble Tea program.
func main() {
	// Check for correct argument count
	if len(os.Args) != 2 {
		fmt.Println("Usage: gopomotime mm:ss")
		os.Exit(1)
	}

	// Parse the duration argument
	duration, err := parseDuration(os.Args[1])
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	// Initialize the model with the parsed duration
	m := model{
		totalTime:   duration,
		elapsedTime: 0,
		isRunning:   true, // Start timer immediately
		isPaused:    false,
		blink:       true,       // Start with text visible
		startTime:   time.Now(), // For smooth progress
	}

	// Start the Bubble Tea program with alternate screen
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
