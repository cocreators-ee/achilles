package achilleslib

// bubbletea model and its methods

import (
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tickMsg time.Time

const (
	fps     = 5
	padding = 2
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("21"))

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second/fps, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type Model struct {
	State           state
	ScannedLibs     *atomic.Int32
	ScannedBins     *atomic.Int32
	LoadingProgress float64
	Libs            map[string]*atomic.Int32
	Bins            []string
	Messages        []string
	Progress        progress.Model
	Table           table.Model
}

func (m *Model) addMessage(message string) {
	messageMutex.Lock()
	defer messageMutex.Unlock()
	m.Messages = append(m.Messages, message)
	GlobalModel.Messages = m.Messages
}

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd = nil

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return GlobalModel, tea.Quit
		default:
			GlobalModel.Table, cmd = GlobalModel.Table.Update(msg)
			return GlobalModel, cmd
		}

	case progress.FrameMsg:
		progressModel, cmd := GlobalModel.Progress.Update(msg)
		GlobalModel.Progress = progressModel.(progress.Model)
		return GlobalModel, cmd

	case tickMsg:
		state := GlobalModel.State
		if state == done {
			cmd = GlobalModel.Progress.SetPercent(1.0)
			if !finished {
				finished = true
				GlobalModel.Table = getTable()
			}
		} else if state == scanningFiles || state == searchingFiles {
			mapMutex.Lock()
			total := float64(len(GlobalModel.Libs) + len(GlobalModel.Bins))
			mapMutex.Unlock()
			scanned := float64(GlobalModel.ScannedBins.Load() + GlobalModel.ScannedLibs.Load())
			cmd = GlobalModel.Progress.SetPercent(scanned / total)

			GlobalModel.Table = getTable()
		}

		return GlobalModel, tea.Batch(tickCmd(), cmd)
	}

	return GlobalModel, cmd
}

func (m Model) View() string {
	pad := strings.Repeat(" ", padding)

	// Show "searching..." on first line while we're still searching for more files
	v := ""
	if m.State == searchingFiles {
		v += pad + "Searching... "
	}
	v += "\n"

	// Show scan % as we're scanning files
	v += pad + m.Progress.View() + " Scanned " + pad

	// Incl libs, bins, and total count
	v += AtomicIntFormat(m.ScannedLibs) + " / " + IntFormat(LibsLen(m)) + " libs" + pad
	v += AtomicIntFormat(m.ScannedBins) + " / " + IntFormat(len(m.Bins)) + " bins" + pad
	v += "(" + IntFormat(LibsLen(m)+len(m.Bins)) + " tot)\n"

	// Render results table
	v += baseStyle.Render(m.Table.View()) + "\n"

	// List latest messages if any
	messageMutex.Lock()
	start := len(m.Messages) - 10
	if start < 0 {
		start = 0
	}
	for _, msg := range m.Messages[start:] {
		v += msg + "\n"
	}
	messageMutex.Unlock()

	return v
}
