# gopomotime

`gopomotime` is a higlhy simplified terminal-based pomodoro written in Go, featuring a simplr ASCII art donut that displays a countdown timer.

<img src="https://github.com/1729prashant/gopomotime/blob/main/demo.gif" width="480" height="270" />

## Features
- **ASCII Donut Timer**: A 13x29 character ASCII donut displays the timer (`MM:SS`) centered at columns 12–16.
- **Progress Visualization**: Progress starts at 12 o'clock, filling clockwise (white for elapsed, red for remaining).
- **Interactive Controls**:
  - `r`: Reset and restart the timer.
  - `p`: Pause/resume or start if stopped.
  - `q` or `Ctrl+C`: Quit the program.
- **Status Messages**:
  - "Timer finished!" (green, blinking when timer reaches 00:00).
  - "Timer paused." and "Timer stopped." (centered).
  - All status messages are centered within 29 columns with a 4-space left margin.
- **Input**: Accepts `mm:ss` format (e.g., `01:30` for 1 minute 30 seconds).
- **Robustness**: Input validation, error handling, and smooth rendering suitable for widespread use.
- **Alias**: Supports `tea` command alias for Bubble Tea framework compatibility.

## Prerequisites
To run `gopomotime`, ensure you have the following installed:

### 1. Go (Latest Version)
`gopomotime` requires Go to compile and run. Install Go from [here](https://go.dev/doc/install)


### 2. Terminal Emulator
- A terminal emulator supporting at least **33 columns** (29 for content + 4 for margin) and **20 rows**.
- **Recommended Fonts**: Monospaced fonts like Fira Code, Menlo, or Consolas for proper ASCII alignment.
- **Check Terminal Size**:
  ```bash
  tput cols
  tput lines
  ```
  Ensure `cols >= 33` and `lines >= 20`.



## Installation
Follow these steps to download, set up, and run `gopomotime`:

### 1. Clone the Repository
Clone the `gopomotime` repository from GitHub:
```bash
git clone https://github.com/1729prashant/gopomotime.git
cd gopomotime
```

### 2. Install Dependencies
`gopomotime` uses the following Go modules:
- `github.com/charmbracelet/bubbletea@latest` (TUI framework).
- `github.com/charmbracelet/lipgloss@latest` (styling).

Install them using:
```bash
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss
go mod tidy
```


### 3. Build the Program
Compile `gopomotime` to create an executable:
```
go build -o gopomotime
```

This generates a `gopomotime` binary in the project directory.

## Running the Program
Run the program with a timer duration in `mm:ss` format (e.g., `00:05` for 5 seconds):
```
./gopomotime 00:05
```

### Example Usage
```bash
./gopomotime 01:30
```
- Starts a 1-minute 30-second timer.
- Displays the ASCII donut with `01:30` centered.
- Progress fills clockwise from 12 o'clock.
- Press `r` to reset/restart, `p` to pause/resume, `q` or `Ctrl+C` to quit.
- When timer reaches `00:00`, "Timer finished!" blinks green and is centered.

### Input Format
- Format: `mm:ss` (minutes:seconds).
- Minutes: 0–99.
- Seconds: 00–59.
- Example: `05:00` (5 minutes), `00:30` (30 seconds).
- Invalid input (e.g., `abc`, `100:00`) shows an error and exits.

## Modifying the Program
To customize `gopomotime`, edit the source code in `main.go`. Common modifications include:

### 1. Changing the ASCII Template
Modify the `donutTemplate` in `drawCircle` to alter the donut shape (ensure 13x29 dimensions):
```go
donutTemplate := []string{
    "          *********          ",
    // ... (13 lines, 29 characters each)
}
```

### 2. Adjusting Colors
Change colors in the style definitions:
```go
redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))  // Red
whiteStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")) // White
greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")) // Green
```

### 3. Modifying Blinking Rate
Adjust `blinkRate` for the "Timer finished!" effect:
```go
const blinkRate = 500 * time.Millisecond // Change to 250ms for faster blinking
```

### 4. Changing Progress Start
Alter the progress fill start in `drawCircle`:
```go
angle := math.Atan2(dy, dx) // 12 o'clock
// For 3 o'clock: angle := math.Atan2(dy, dx) - math.Pi/2
```

After modifications, rebuild:
```bash
go build -o gopomotime
```


## Contributing
Contributions are welcome! To contribute:
1. Fork the repository.
2. Create a branch:
   ```bash
   git checkout -b feature/your-feature
   ```
3. Make changes and commit:
   ```bash
   git commit -m "Add your feature"
   ```
4. Push to your fork:
   ```bash
   git push origin feature/your-feature
   ```
5. Open a pull request on GitHub.

Please include tests and update this README if necessary.

## License
This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
