package tui

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/kunalshah017/myference/cli/internal/host"
	"github.com/kunalshah017/myference/cli/internal/provider"
)

type Screen uint8

const (
	ScreenHome Screen = iota
	ScreenProviders
	ScreenAPI
	ScreenReview
	ScreenStatus
)

type apiStep uint8

const (
	apiStepURL apiStep = iota
	apiStepKey
	apiStepModel
)

type Dependencies struct {
	ListModels func(context.Context, string, string) ([]string, error)
	Configure  func(context.Context, []host.Selection) error
	Activate   func(context.Context, []host.Selection) error
	Start      func(context.Context) error
	Stop       func()
	Snapshot   func() (provider.StatusSnapshot, error)
}

type Model struct {
	dependencies Dependencies
	screen       Screen
	candidates   []host.Candidate
	cursor       int
	selected     map[string]bool
	secrets      map[string]string
	width        int
	height       int
	running      bool
	confirmQuit  bool
	busy         bool
	status       string
	err          error
	snapshot     provider.StatusSnapshot

	apiCompatible bool
	apiStep       apiStep
	apiName       string
	apiURL        textinput.Model
	apiKey        textinput.Model
	apiModel      textinput.Model
}

type catalogMsg struct {
	baseURL, providerName, secret string
	models                        []string
	err                           error
}

type startedMsg struct{ err error }
type snapshotMsg struct {
	snapshot provider.StatusSnapshot
	err      error
}
type pollStatusMsg struct{}

func NewModel(dependencies Dependencies, candidates []host.Candidate) Model {
	urlInput := textinput.New()
	urlInput.Placeholder = "https://provider.example"
	urlInput.Prompt = "Base URL: "
	urlInput.SetWidth(64)
	keyInput := textinput.New()
	keyInput.Placeholder = "API key"
	keyInput.Prompt = "API key: "
	keyInput.EchoMode = textinput.EchoPassword
	keyInput.EchoCharacter = '*'
	keyInput.SetWidth(64)
	modelInput := textinput.New()
	modelInput.Placeholder = "model-name"
	modelInput.Prompt = "Model: "
	modelInput.SetWidth(64)
	model := Model{
		dependencies: dependencies,
		screen:       ScreenHome,
		candidates:   append([]host.Candidate(nil), candidates...),
		selected:     make(map[string]bool),
		secrets:      make(map[string]string),
		apiURL:       urlInput,
		apiKey:       keyInput,
		apiModel:     modelInput,
	}
	for _, candidate := range candidates {
		if candidate.Selected {
			model.selected[candidate.ID] = true
		}
	}
	return model
}

func (model Model) Init() tea.Cmd { return nil }

func (model Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		resized := model.Resize(message.Width, message.Height)
		return resized, nil
	case catalogMsg:
		model.applyCatalog(message)
		return model, nil
	case startedMsg:
		model.applyStarted(message)
		if model.running {
			return model, model.snapshotCommand()
		}
		return model, nil
	case snapshotMsg:
		model.applySnapshot(message)
		if model.running {
			return model, tea.Tick(time.Second, func(time.Time) tea.Msg { return pollStatusMsg{} })
		}
		return model, nil
	case pollStatusMsg:
		if model.running {
			return model, model.snapshotCommand()
		}
		return model, nil
	case tea.KeyPressMsg:
		key := message.String()
		if model.screen == ScreenAPI && key != "enter" && key != "esc" && key != "ctrl+c" {
			var command tea.Cmd
			switch model.apiStep {
			case apiStepURL:
				model.apiURL, command = model.apiURL.Update(message)
			case apiStepKey:
				model.apiKey, command = model.apiKey.Update(message)
			case apiStepModel:
				model.apiModel, command = model.apiModel.Update(message)
			}
			return model, command
		}
		updated, command := model.HandleKey(key)
		return updated, command
	}
	return model, nil
}

func (model Model) View() tea.View {
	view := tea.NewView(model.ViewText())
	view.AltScreen = true
	return view
}

func (model Model) ViewText() string {
	var output strings.Builder
	output.WriteString("Myference Hosting\n")
	output.WriteString(strings.Repeat("=", 18) + "\n\n")
	switch model.screen {
	case ScreenHome:
		items := []string{"Manage providers", "Review & start", "Live status", "Quit"}
		output.WriteString("Host models without leaving your terminal.\n\n")
		for index, item := range items {
			output.WriteString(menuRow(index == model.cursor, item))
		}
		output.WriteString("\n↑/↓ navigate • Enter select • q quit\n")
	case ScreenProviders:
		output.WriteString("Discovered providers\n\n")
		for index, candidate := range model.candidates {
			checked := " "
			if model.selected[candidate.ID] {
				checked = "x"
			}
			label := fmt.Sprintf("[%s] %-8s %s", checked, candidate.Name, candidate.Model)
			if candidate.Digest != "" {
				label += "  " + short(candidate.Digest, 18)
			}
			output.WriteString(menuRow(index == model.cursor, label))
		}
		output.WriteString(menuRow(model.cursor == len(model.candidates), "+ Add OpenAI"))
		output.WriteString(menuRow(model.cursor == len(model.candidates)+1, "+ Add OpenAI-compatible"))
		output.WriteString("\nSpace toggle • Enter configure • r review • Esc back\n")
	case ScreenAPI:
		if model.apiCompatible {
			output.WriteString("Add OpenAI-compatible provider\n\n")
		} else {
			output.WriteString("Add OpenAI\n\n")
		}
		switch model.apiStep {
		case apiStepURL:
			output.WriteString(model.apiURL.View())
		case apiStepKey:
			output.WriteString(model.apiKey.View())
		case apiStepModel:
			output.WriteString("Model discovery is unavailable. Enter the model name manually.\n\n")
			output.WriteString(model.apiModel.View())
		}
		if model.busy {
			output.WriteString("\nLoading models…")
		}
		if model.err != nil {
			output.WriteString("\n" + model.err.Error())
		}
		output.WriteString("\n\nEnter continue • Esc cancel\n")
	case ScreenReview:
		selected := model.selections()
		fmt.Fprintf(&output, "%d providers selected\n\n", len(selected))
		for _, selection := range selected {
			fmt.Fprintf(&output, "• %-8s %s\n", selection.Candidate.Name, selection.Candidate.Model)
		}
		if len(selected) == 0 {
			output.WriteString("Select at least one provider before starting.\n")
		}
		if model.busy && model.status != "" {
			output.WriteString("\n" + model.status + "\n")
		}
		output.WriteString("\nEnter configure & start • Esc back\n")
	case ScreenStatus:
		state := "Stopped"
		if model.running {
			state = "Starting"
			if model.snapshot.Connected {
				state = "Connected"
			}
		}
		output.WriteString("Hosting status: " + state + "\n\n")
		if model.running {
			fmt.Fprintf(&output, "%d completed requests\n", model.snapshot.Requests)
			for _, offer := range model.snapshot.Offers {
				health := "healthy"
				if !offer.Healthy {
					health = "unhealthy"
				}
				fmt.Fprintf(&output, "• %-16s %s", offer.Model, health)
				if offer.Error != "" {
					output.WriteString(" — " + offer.Error)
				}
				output.WriteByte('\n')
			}
			output.WriteByte('\n')
		}
		if model.status != "" {
			output.WriteString(model.status + "\n")
		}
		if model.err != nil {
			output.WriteString("Action required: " + model.err.Error() + "\n")
		}
		if model.confirmQuit {
			output.WriteString("\nStop hosting and quit? y/N\n")
		} else {
			output.WriteString("\nr retry • q quit\n")
		}
	}
	return output.String()
}

func (model Model) HandleKey(key string) (Model, tea.Cmd) {
	if key == "ctrl+c" {
		if model.running {
			model.confirmQuit = true
			return model, nil
		}
		return model, tea.Quit
	}
	if model.confirmQuit {
		switch strings.ToLower(key) {
		case "y":
			if model.dependencies.Stop != nil {
				model.dependencies.Stop()
			}
			model.running = false
			return model, tea.Quit
		case "n", "esc":
			model.confirmQuit = false
		}
		return model, nil
	}
	switch model.screen {
	case ScreenHome:
		model.cursor = move(model.cursor, key, 4)
		if key == "q" {
			return model, tea.Quit
		}
		if key == "enter" {
			switch model.cursor {
			case 0:
				model.screen, model.cursor = ScreenProviders, 0
			case 1:
				model.screen, model.cursor = ScreenReview, 0
			case 2:
				model.screen, model.cursor = ScreenStatus, 0
			case 3:
				return model, tea.Quit
			}
		}
	case ScreenProviders:
		model.cursor = move(model.cursor, key, len(model.candidates)+2)
		switch key {
		case "esc":
			model.screen, model.cursor = ScreenHome, 0
		case "r":
			model.screen, model.cursor = ScreenReview, 0
		case "space":
			if model.cursor < len(model.candidates) {
				id := model.candidates[model.cursor].ID
				model.selected[id] = !model.selected[id]
			}
		case "enter":
			if model.cursor == len(model.candidates) {
				model.openAPIForm(false)
			} else if model.cursor == len(model.candidates)+1 {
				model.openAPIForm(true)
			}
		}
	case ScreenAPI:
		switch key {
		case "esc":
			model.clearAPIForm()
			model.screen, model.cursor = ScreenProviders, 0
		case "enter":
			return model.advanceAPI()
		}
	case ScreenReview:
		if key == "esc" {
			model.screen, model.cursor = ScreenProviders, 0
		} else if key == "enter" && len(model.selections()) > 0 && !model.busy {
			model.busy, model.status, model.err = true, "Configuring providers and opening wallet activation…", nil
			return model, model.startCommand()
		}
	case ScreenStatus:
		switch key {
		case "esc":
			model.screen, model.cursor = ScreenHome, 0
		case "r":
			if !model.busy {
				model.busy = true
				return model, model.startCommand()
			}
		case "q":
			if model.running {
				model.confirmQuit = true
				return model, nil
			}
			return model, tea.Quit
		}
	}
	return model, nil
}

func (model *Model) openAPIForm(compatible bool) {
	model.screen, model.apiCompatible, model.err, model.busy = ScreenAPI, compatible, nil, false
	if compatible {
		model.apiStep = apiStepURL
		model.apiURL.Focus()
	} else {
		model.apiStep = apiStepKey
		model.apiName = "OpenAI"
		model.apiURL.SetValue("https://api.openai.com")
		model.apiKey.Focus()
	}
}

func (model Model) advanceAPI() (Model, tea.Cmd) {
	if model.apiStep == apiStepURL {
		parsed, err := url.Parse(strings.TrimSpace(model.apiURL.Value()))
		if err != nil || parsed.Host == "" {
			model.err = errorsNew("Enter a valid provider URL")
			return model, nil
		}
		model.apiName = parsed.Hostname()
		model.apiStep, model.err = apiStepKey, nil
		model.apiURL.Blur()
		model.apiKey.Focus()
		return model, nil
	}
	if model.apiStep == apiStepModel {
		name := strings.TrimSpace(model.apiModel.Value())
		if name == "" {
			model.err = errorsNew("Enter a model name")
			return model, nil
		}
		model.addAPICandidates(model.apiURL.Value(), model.apiName, model.apiKey.Value(), []string{name})
		model.clearAPIForm()
		model.screen, model.cursor = ScreenProviders, 0
		return model, nil
	}
	secret := model.apiKey.Value()
	if strings.TrimSpace(secret) == "" {
		model.err = errorsNew("Enter an API key")
		return model, nil
	}
	if model.dependencies.ListModels == nil {
		model.apiStep, model.err = apiStepModel, host.ErrModelCatalogUnavailable
		model.apiKey.Blur()
		model.apiModel.Focus()
		return model, nil
	}
	baseURL, providerName := model.apiURL.Value(), model.apiName
	model.busy, model.err = true, nil
	return model, func() tea.Msg {
		models, err := model.dependencies.ListModels(context.Background(), baseURL, secret)
		return catalogMsg{baseURL: baseURL, providerName: providerName, secret: secret, models: models, err: err}
	}
}

func (model *Model) applyCatalog(message catalogMsg) {
	model.busy = false
	if message.err != nil {
		model.err, model.apiStep = message.err, apiStepModel
		model.apiKey.Blur()
		model.apiModel.Focus()
		return
	}
	model.addAPICandidates(message.baseURL, message.providerName, message.secret, message.models)
	model.clearAPIForm()
	model.screen, model.cursor = ScreenProviders, 0
}

func (model *Model) addAPICandidates(baseURL, providerName, secret string, models []string) {
	for _, name := range models {
		candidate := host.Candidate{Kind: "openai", Name: providerName, URL: strings.TrimRight(baseURL, "/"), Model: name, State: host.StateReady}
		candidate.ID = host.StableID(candidate)
		if !slices.ContainsFunc(model.candidates, func(existing host.Candidate) bool { return existing.ID == candidate.ID }) {
			model.candidates = append(model.candidates, candidate)
		}
		model.secrets[candidate.ID] = secret
	}
}

func (model *Model) clearAPIForm() {
	model.apiURL.SetValue("")
	model.apiKey.SetValue("")
	model.apiModel.SetValue("")
	model.apiURL.Blur()
	model.apiKey.Blur()
	model.apiModel.Blur()
	model.apiName, model.err, model.busy = "", nil, false
}

func (model Model) startCommand() tea.Cmd {
	selections := model.selections()
	return func() tea.Msg {
		if model.dependencies.Configure != nil {
			if err := model.dependencies.Configure(context.Background(), selections); err != nil {
				return startedMsg{err: err}
			}
		}
		if model.dependencies.Activate != nil {
			if err := model.dependencies.Activate(context.Background(), selections); err != nil {
				return startedMsg{err: err}
			}
		}
		if model.dependencies.Start != nil {
			return startedMsg{err: model.dependencies.Start(context.Background())}
		}
		return startedMsg{}
	}
}

func (model *Model) applyStarted(message startedMsg) {
	model.screen, model.busy, model.confirmQuit = ScreenStatus, false, false
	model.err = message.err
	if message.err == nil {
		model.running, model.status = true, "Selected providers are connected."
		for id := range model.secrets {
			delete(model.secrets, id)
		}
	} else {
		model.running, model.status = false, "Hosting could not start."
	}
}

func (model Model) snapshotCommand() tea.Cmd {
	return func() tea.Msg {
		if model.dependencies.Snapshot == nil {
			return snapshotMsg{}
		}
		snapshot, err := model.dependencies.Snapshot()
		return snapshotMsg{snapshot: snapshot, err: err}
	}
}

func (model *Model) applySnapshot(message snapshotMsg) {
	if message.err != nil {
		model.status = "Live status is temporarily unavailable."
		return
	}
	model.snapshot = message.snapshot
	if message.snapshot.Connected {
		model.status = "Provider relay connected."
	} else {
		model.status = "Connecting to provider relay…"
	}
}

func (model Model) selections() []host.Selection {
	result := make([]host.Selection, 0)
	for _, candidate := range model.candidates {
		if model.selected[candidate.ID] {
			result = append(result, host.Selection{Candidate: candidate, Secret: model.secrets[candidate.ID]})
		}
	}
	return result
}

func (model Model) Selected(id string) bool { return model.selected[id] }

func (model Model) Resize(width, height int) Model {
	model.width, model.height = width, height
	inputWidth := width - 8
	if inputWidth < 20 {
		inputWidth = 20
	}
	model.apiURL.SetWidth(inputWidth)
	model.apiKey.SetWidth(inputWidth)
	model.apiModel.SetWidth(inputWidth)
	return model
}

func move(cursor int, key string, length int) int {
	if length == 0 {
		return 0
	}
	switch key {
	case "up", "k":
		return (cursor - 1 + length) % length
	case "down", "j":
		return (cursor + 1) % length
	default:
		return cursor
	}
}

func menuRow(active bool, label string) string {
	prefix := "  "
	if active {
		prefix = "> "
	}
	return prefix + label + "\n"
}

func short(value string, maximum int) string {
	if len(value) <= maximum {
		return value
	}
	return value[:maximum-1] + "…"
}

func errorsNew(message string) error { return fmt.Errorf("%s", message) }
