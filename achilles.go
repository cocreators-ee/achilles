package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cocreators-ee/achilles/achilleslib"
)

func main() {
	init := achilleslib.InitialModel()
	achilleslib.GlobalModel = &init
	p := tea.NewProgram(init)
	go achilleslib.GlobalModel.FindFiles()
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
