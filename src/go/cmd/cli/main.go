package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/simon/repo-template/internal/counter"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			Background(lipgloss.Color("99"))

	numberStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true).
			Padding(0, 2)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

type tuiModel struct {
	count    int
	quitting bool
}

func (m tuiModel) Init() tea.Cmd {
	return nil
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			m.count++
		case "down", "j":
			m.count--
		}
	}
	return m, nil
}

func (m tuiModel) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}
	return fmt.Sprintf(
		"%s\n\n  %s %s  %s\n\n  %s\n",
		titleStyle.Render("🧮 Counter TUI"),
		"Count:",
		numberStyle.Render(fmt.Sprintf("%d", m.count)),
		"",
		helpStyle.Render("↑/k to increase  •  ↓/j to decrease  •  q to quit"),
	)
}

func runTUI() {
	p := tea.NewProgram(tuiModel{})
	if err := p.Start(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

func runCLI() {
	c := counter.New()

	if len(os.Args) < 2 {
		fmt.Println("Usage: cli [tui|inc|dec|get|reset]")
		fmt.Println("  tui    - run interactive TUI")
		fmt.Println("  inc    - increase counter")
		fmt.Println("  dec    - decrease counter")
		fmt.Println("  get    - show current value")
		fmt.Println("  reset  - reset to zero")
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "tui":
		runTUI()
	case "inc":
		c.Inc()
		fmt.Printf("Count: %d\n", c.Get())
	case "dec":
		c.Dec()
		fmt.Printf("Count: %d\n", c.Get())
	case "get":
		fmt.Printf("Count: %d\n", c.Get())
	case "reset":
		c.Reset()
		fmt.Println("Reset to 0")
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		os.Exit(1)
	}
}

func main() {
	runCLI()
}
