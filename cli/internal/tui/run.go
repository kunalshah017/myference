package tui

import (
	"io"

	tea "charm.land/bubbletea/v2"
	"github.com/kunalshah017/myference/cli/internal/host"
)

func Run(input io.Reader, output io.Writer, dependencies Dependencies, candidates []host.Candidate) error {
	program := tea.NewProgram(NewModel(dependencies, candidates), tea.WithInput(input), tea.WithOutput(output))
	_, err := program.Run()
	return err
}
