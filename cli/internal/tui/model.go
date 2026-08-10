package tui

import (
	"context"
	"fmt"
	"math/big"
	"net/url"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/kunalshah017/myference/cli/internal/account"
	"github.com/kunalshah017/myference/cli/internal/config"
	"github.com/kunalshah017/myference/cli/internal/host"
	"github.com/kunalshah017/myference/cli/internal/provider"
	"github.com/kunalshah017/myference/cli/internal/providerops"
)

type Screen uint8

const (
	ScreenHome Screen = iota
	ScreenProviders
	ScreenAPI
	ScreenReview
	ScreenStatus
	ScreenOffers
	ScreenOfferAttach
	ScreenPricing
	ScreenCollateral
	ScreenCollateralDeposit
)

type priceStep uint8

const (
	priceStepInput priceStep = iota
	priceStepOutput
	priceStepCompute
	priceStepReview
)

type apiStep uint8

const (
	apiStepURL apiStep = iota
	apiStepKey
	apiStepModel
)

type Dependencies struct {
	ListModels   func(context.Context, string, string) ([]string, error)
	Configure    func(context.Context, []host.Selection) error
	Start        func(context.Context) error
	Stop         func()
	Snapshot     func() (provider.StatusSnapshot, error)
	Backends     []config.Backend
	LoadBackends func() []config.Backend
	Account      func(context.Context) (account.ProviderAccount, error)
	Attach       func(context.Context, string, string) error
	Publish      func(context.Context, config.Backend, providerops.Rates) error
	Deposit      func(context.Context, string) error
	RequestExit  func(context.Context) error
	FinalizeExit func(context.Context) error
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

	pricingBackend config.Backend
	priceStep      priceStep
	priceInput     textinput.Model
	priceOutput    textinput.Model
	priceCompute   textinput.Model
	depositAmount  textinput.Model
	account        account.ProviderAccount
	accountErr     error
	attachBackend  config.Backend
	attachOffers   []account.EditableOffer
}

type catalogMsg struct {
	baseURL, providerName, secret string
	models                        []string
	err                           error
}

type startedMsg struct{ err error }
type configuredMsg struct{ err error }
type snapshotMsg struct {
	snapshot provider.StatusSnapshot
	err      error
}
type pollStatusMsg struct{}
type pollAccountMsg struct{}
type accountMsg struct {
	account account.ProviderAccount
	err     error
}
type providerOperationMsg struct {
	status string
	err    error
}

type offerRow struct {
	backend config.Backend
	offer   account.EditableOffer
	wallet  bool
}

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
	priceInput := textinput.New()
	priceInput.Prompt = "Input / million tokens (MON): "
	priceInput.SetWidth(64)
	priceOutput := textinput.New()
	priceOutput.Prompt = "Output / million tokens (MON): "
	priceOutput.SetWidth(64)
	priceCompute := textinput.New()
	priceCompute.Prompt = "Compute / second (MON): "
	priceCompute.SetWidth(64)
	depositAmount := textinput.New()
	depositAmount.Prompt = "Deposit (MON): "
	depositAmount.SetWidth(64)
	model := Model{
		dependencies:  dependencies,
		screen:        ScreenHome,
		candidates:    append([]host.Candidate(nil), candidates...),
		selected:      make(map[string]bool),
		secrets:       make(map[string]string),
		apiURL:        urlInput,
		apiKey:        keyInput,
		apiModel:      modelInput,
		priceInput:    priceInput,
		priceOutput:   priceOutput,
		priceCompute:  priceCompute,
		depositAmount: depositAmount,
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
			return model, tea.Batch(model.snapshotCommand(), model.accountCommand())
		}
		return model, nil
	case configuredMsg:
		model.busy, model.err = false, message.err
		if message.err == nil {
			model.screen, model.cursor, model.status, model.busy = ScreenOffers, 0, "Providers saved. Publish pricing before starting.", true
			return model, model.accountCommand()
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
	case accountMsg:
		model.applyAccount(message)
		if model.screen == ScreenStatus {
			return model, tea.Tick(15*time.Second, func(time.Time) tea.Msg { return pollAccountMsg{} })
		}
		return model, nil
	case pollAccountMsg:
		if model.screen == ScreenStatus {
			return model, model.accountCommand()
		}
		return model, nil
	case providerOperationMsg:
		model.busy = false
		model.err = message.err
		if message.err == nil {
			model.status = message.status
			if model.status == "" {
				model.status = "Provider action confirmed."
			}
			if model.screen == ScreenOfferAttach {
				model.screen, model.cursor = ScreenOffers, 0
				model.attachOffers = nil
			}
			if model.screen == ScreenPricing {
				model.screen = ScreenOffers
				model.clearPricing()
			}
			if model.screen == ScreenCollateralDeposit {
				model.screen = ScreenCollateral
				model.depositAmount.SetValue("")
				model.depositAmount.Blur()
			}
		}
		if message.err == nil {
			return model, model.accountCommand()
		}
		return model, nil
	case tea.PasteMsg:
		var command tea.Cmd
		switch model.screen {
		case ScreenAPI:
			switch model.apiStep {
			case apiStepURL:
				model.apiURL, command = model.apiURL.Update(message)
			case apiStepKey:
				model.apiKey, command = model.apiKey.Update(message)
			case apiStepModel:
				model.apiModel, command = model.apiModel.Update(message)
			}
		case ScreenPricing:
			switch model.priceStep {
			case priceStepInput:
				model.priceInput, command = model.priceInput.Update(message)
			case priceStepOutput:
				model.priceOutput, command = model.priceOutput.Update(message)
			case priceStepCompute:
				model.priceCompute, command = model.priceCompute.Update(message)
			}
		case ScreenCollateralDeposit:
			model.depositAmount, command = model.depositAmount.Update(message)
		}
		return model, command
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
		if model.screen == ScreenPricing && key != "enter" && key != "esc" && key != "ctrl+c" {
			var command tea.Cmd
			switch model.priceStep {
			case priceStepInput:
				model.priceInput, command = model.priceInput.Update(message)
			case priceStepOutput:
				model.priceOutput, command = model.priceOutput.Update(message)
			case priceStepCompute:
				model.priceCompute, command = model.priceCompute.Update(message)
			}
			return model, command
		}
		if model.screen == ScreenCollateralDeposit && key != "enter" && key != "esc" && key != "ctrl+c" {
			var command tea.Cmd
			model.depositAmount, command = model.depositAmount.Update(message)
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
		items := []string{"Providers", "Offers & Pricing", "Collateral", "Live Status", "Quit"}
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
		output.WriteString("\nEnter save providers • Esc back\n")
	case ScreenStatus:
		state := "Stopped"
		if model.running {
			state = "Starting"
			if model.snapshot.Connected {
				state = "Connected"
			}
		}
		output.WriteString("Hosting status: " + state + "\n\n")
		if model.account.ProviderEarningsWei != "" {
			fmt.Fprintf(&output, "Lifetime earnings: %s MON\nClaimable: %s MON\n", formatWei(model.account.ProviderEarningsWei), formatWei(model.account.ClaimableWei))
		} else if model.accountErr != nil {
			output.WriteString("Lifetime earnings unavailable: " + model.accountErr.Error() + "\n")
		}
		if model.running {
			fmt.Fprintf(&output, "This run: %d completed · %s MON earned\n", model.snapshot.Requests, formatWei(model.snapshot.RunEarningsWei))
			requests := visibleRequests(model.snapshot.RecentRequests, 8)
			if len(requests) > 0 {
				output.WriteString("\nRequests\n")
				for _, request := range requests {
					fmt.Fprintf(&output, "• %-14s %-18s %-9s", short(request.RequestID, 14), short(request.Model, 18), request.State)
					switch request.State {
					case "completed":
						fmt.Fprintf(&output, " %d in · %d out · %s MON earned", request.InputTokens, request.OutputTokens, formatWei(request.EarningsWei))
					case "active":
						output.WriteString(" earning pending")
					case "settling":
						output.WriteString(" settlement pending")
					case "failed":
						if request.Error != "" {
							output.WriteString(" " + short(request.Error, 32))
						}
					}
					output.WriteByte('\n')
				}
				if len(model.snapshot.RecentRequests) > len(requests) {
					fmt.Fprintf(&output, "  … %d older requests hidden\n", len(model.snapshot.RecentRequests)-len(requests))
				}
			}
			output.WriteString("\nProviders\n")
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
	case ScreenOffers:
		output.WriteString("Offers & Pricing\n\n")
		rows := model.offerRows()
		backends := model.currentBackends()
		output.WriteString("This machine\n")
		if len(backends) == 0 {
			output.WriteString("  Configure a provider first.\n")
		}
		rowIndex := 0
		for index, item := range backends {
			state := "Not published"
			if item.PriceVersion > 0 {
				state = fmt.Sprintf("Public as %s v%d", item.EffectiveOfferID(), item.PriceVersion)
			} else if matches := model.compatibleOffers(item); len(matches) == 1 {
				state = fmt.Sprintf("Attach %s v%d", matches[0].OfferID, matches[0].Version)
			} else if len(matches) > 1 {
				state = fmt.Sprintf("%d compatible offers", len(matches))
			}
			output.WriteString(menuRow(index == model.cursor, fmt.Sprintf("%-18s %-20s %s", item.Name, item.Model, state)))
			rowIndex++
		}
		output.WriteString("\nWallet offers\n")
		if len(model.account.Offers) == 0 && !model.busy {
			output.WriteString("  No wallet offers found.\n")
		}
		for _, row := range rows[len(backends):] {
			offer := row.offer
			output.WriteString(menuRow(rowIndex == model.cursor, fmt.Sprintf("%-18s %-20s v%d · %s", offer.OfferID, offer.Model, offer.Version, model.walletOfferState(offer))))
			rowIndex++
		}
		if model.status != "" {
			output.WriteString("\n" + model.status + "\n")
		}
		if model.err != nil {
			output.WriteString("\nAction required: " + model.err.Error() + "\n")
		}
		if model.busy {
			output.WriteString("\nLoading wallet offers…\n")
		}
		output.WriteString("\nEnter reuse/select • e create/edit offer • s start hosting • Esc back\n")
	case ScreenOfferAttach:
		fmt.Fprintf(&output, "Attach wallet offer · %s (%s)\n\n", model.attachBackend.Name, model.attachBackend.Model)
		for index, offer := range model.attachOffers {
			output.WriteString(menuRow(index == model.cursor, fmt.Sprintf("%-20s v%d", offer.OfferID, offer.Version)))
		}
		if model.busy {
			output.WriteString("\nSaving attachment…")
		}
		if model.err != nil {
			output.WriteString("\nAction required: " + model.err.Error())
		}
		output.WriteString("\n\nEnter attach • Esc cancel\n")
	case ScreenPricing:
		title := "Create new offer"
		if model.pricingBackend.PriceVersion > 0 {
			title = "Edit offer pricing"
		}
		fmt.Fprintf(&output, "%s · %s (%s)\n\n", title, model.pricingBackend.Name, model.pricingBackend.Model)
		computeOnly := backendComputeOnly(model.pricingBackend)
		switch model.priceStep {
		case priceStepInput:
			output.WriteString(model.priceInput.View())
		case priceStepOutput:
			output.WriteString(model.priceOutput.View())
		case priceStepCompute:
			output.WriteString(model.priceCompute.View())
		case priceStepReview:
			if !computeOnly {
				fmt.Fprintf(&output, "Input / million tokens: %s MON\nOutput / million tokens: %s MON\n", model.priceInput.Value(), model.priceOutput.Value())
			}
			fmt.Fprintf(&output, "Compute / second: %s MON\n\nEnter open wallet approval", model.priceCompute.Value())
		}
		if model.busy {
			output.WriteString("\nWaiting for wallet and chain confirmation…")
		}
		if model.err != nil {
			output.WriteString("\nAction required: " + model.err.Error())
		}
		output.WriteString("\n\nEnter continue • Esc cancel\n")
	case ScreenCollateral:
		output.WriteString("Collateral\n\n")
		if model.busy {
			output.WriteString("Loading provider account…\n")
		} else {
			fmt.Fprintf(&output, "Bond: %s MON\nClaimable: %s MON\n", formatWei(model.account.ProviderBondWei), formatWei(model.account.ClaimableWei))
			if model.account.BondExitAvailableAt == 0 {
				output.WriteString("\nd Deposit • x Request exit")
			} else {
				output.WriteString("\nf Finalize exit")
			}
		}
		if model.status != "" {
			output.WriteString("\n" + model.status)
		}
		if model.err != nil {
			output.WriteString("\nAction required: " + model.err.Error())
		}
		output.WriteString("\nEsc back\n")
	case ScreenCollateralDeposit:
		output.WriteString("Deposit collateral\n\n")
		output.WriteString(model.depositAmount.View())
		if model.busy {
			output.WriteString("\nWaiting for wallet and chain confirmation…")
		}
		if model.err != nil {
			output.WriteString("\nAction required: " + model.err.Error())
		}
		output.WriteString("\n\nEnter open wallet approval • Esc cancel\n")
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
		model.cursor = move(model.cursor, key, 5)
		if key == "q" {
			return model, tea.Quit
		}
		if key == "enter" {
			switch model.cursor {
			case 0:
				model.screen, model.cursor = ScreenProviders, 0
			case 1:
				model.screen, model.cursor, model.busy, model.err = ScreenOffers, 0, true, nil
				return model, model.accountCommand()
			case 2:
				model.screen, model.cursor, model.busy, model.err = ScreenCollateral, 0, true, nil
				return model, model.accountCommand()
			case 3:
				model.screen, model.cursor = ScreenStatus, 0
				return model, model.accountCommand()
			case 4:
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
			model.busy, model.status, model.err = true, "Saving provider configuration…", nil
			return model, model.configureCommand()
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
	case ScreenOffers:
		rows := model.offerRows()
		model.cursor = move(model.cursor, key, len(rows))
		if key == "esc" {
			model.screen, model.cursor = ScreenHome, 0
		}
		if key == "e" && len(rows) > 0 && !model.busy {
			if rows[model.cursor].wallet {
				model.status = "Select a provider under This machine to create or edit an offer."
			} else {
				model.openPricing(rows[model.cursor].backend)
			}
		}
		if key == "enter" && len(rows) > 0 && !model.busy {
			row := rows[model.cursor]
			if row.wallet {
				if attached, ok := model.attachedBackend(row.offer.OfferID); ok {
					model.status = fmt.Sprintf("Already attached to %s.", attached.Name)
					return model, nil
				}
				matches := model.attachableBackends(row.offer)
				switch len(matches) {
				case 0:
					model.status = "Configure a matching provider first."
				case 1:
					model.busy, model.err = true, nil
					return model, model.attachCommand(matches[0], row.offer)
				default:
					model.status = "Multiple matching local providers; select one under This machine."
				}
				return model, nil
			}
			item := row.backend
			if item.PriceVersion > 0 {
				model.status = fmt.Sprintf("Already attached as %s v%d. Press e to edit pricing.", item.EffectiveOfferID(), item.PriceVersion)
				return model, nil
			}
			matches := model.compatibleOffers(item)
			switch len(matches) {
			case 0:
				model.openPricing(item)
			case 1:
				model.busy, model.err = true, nil
				return model, model.attachCommand(item, matches[0])
			default:
				model.screen, model.cursor, model.attachBackend, model.attachOffers = ScreenOfferAttach, 0, item, matches
			}
		}
		if key == "s" && !model.busy {
			backends := model.currentBackends()
			if slices.ContainsFunc(backends, func(item config.Backend) bool { return item.Enabled && item.PriceVersion == 0 }) {
				model.err = errorsNew("Publish pricing for every enabled provider before starting")
				return model, nil
			}
			model.busy, model.err = true, nil
			return model, model.startConfiguredCommand()
		}
	case ScreenOfferAttach:
		model.cursor = move(model.cursor, key, len(model.attachOffers))
		if key == "esc" && !model.busy {
			model.screen, model.cursor, model.attachOffers, model.err = ScreenOffers, 0, nil, nil
		}
		if key == "enter" && len(model.attachOffers) > 0 && !model.busy {
			model.busy, model.err = true, nil
			return model, model.attachCommand(model.attachBackend, model.attachOffers[model.cursor])
		}
	case ScreenPricing:
		if key == "esc" && !model.busy {
			model.screen, model.cursor, model.err = ScreenOffers, 0, nil
			model.clearPricing()
		}
		if key == "enter" && !model.busy {
			return model.advancePricing()
		}
	case ScreenCollateral:
		switch key {
		case "esc":
			model.screen, model.cursor = ScreenHome, 0
		case "d":
			if model.account.BondExitAvailableAt == 0 {
				model.screen, model.err = ScreenCollateralDeposit, nil
				model.depositAmount.Focus()
			}
		case "x":
			if model.account.BondExitAvailableAt == 0 && model.dependencies.RequestExit != nil {
				model.busy = true
				return model, func() tea.Msg { return providerOperationMsg{err: model.dependencies.RequestExit(context.Background())} }
			}
		case "f":
			if model.account.BondExitAvailableAt != 0 && model.dependencies.FinalizeExit != nil {
				model.busy = true
				return model, func() tea.Msg {
					return providerOperationMsg{err: model.dependencies.FinalizeExit(context.Background())}
				}
			}
		}
	case ScreenCollateralDeposit:
		if key == "esc" && !model.busy {
			model.screen, model.err = ScreenCollateral, nil
			model.depositAmount.SetValue("")
			model.depositAmount.Blur()
		}
		if key == "enter" && !model.busy {
			if strings.TrimSpace(model.depositAmount.Value()) == "" {
				model.err = errorsNew("Enter a deposit amount")
				return model, nil
			}
			if model.dependencies.Deposit == nil {
				model.err = errorsNew("Collateral operations are unavailable")
				return model, nil
			}
			amount := model.depositAmount.Value()
			model.busy, model.err = true, nil
			return model, func() tea.Msg {
				return providerOperationMsg{err: model.dependencies.Deposit(context.Background(), amount)}
			}
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
		if model.dependencies.Start != nil {
			return startedMsg{err: model.dependencies.Start(context.Background())}
		}
		return startedMsg{}
	}
}

func (model Model) configureCommand() tea.Cmd {
	selections := model.selections()
	return func() tea.Msg {
		if model.dependencies.Configure == nil {
			return configuredMsg{}
		}
		return configuredMsg{err: model.dependencies.Configure(context.Background(), selections)}
	}
}

func (model Model) startConfiguredCommand() tea.Cmd {
	return func() tea.Msg {
		if model.dependencies.Start == nil {
			return startedMsg{}
		}
		return startedMsg{err: model.dependencies.Start(context.Background())}
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

func (model Model) Screen() Screen { return model.screen }

func (model Model) currentBackends() []config.Backend {
	if model.dependencies.LoadBackends != nil {
		return model.dependencies.LoadBackends()
	}
	return model.dependencies.Backends
}

func (model Model) offerRows() []offerRow {
	backends := model.currentBackends()
	rows := make([]offerRow, 0, len(backends)+len(model.account.Offers))
	for _, item := range backends {
		rows = append(rows, offerRow{backend: item})
	}
	for _, offer := range model.account.Offers {
		rows = append(rows, offerRow{offer: offer, wallet: true})
	}
	return rows
}

func (model Model) walletOfferState(offer account.EditableOffer) string {
	if attached, ok := model.attachedBackend(offer.OfferID); ok {
		return "Attached here · " + attached.Name
	}
	matches := len(model.attachableBackends(offer))
	if matches == 1 {
		return "Ready to attach"
	}
	if matches > 1 {
		return fmt.Sprintf("%d matching local providers", matches)
	}
	return "No matching local provider"
}

func (model Model) attachedBackend(offerID string) (config.Backend, bool) {
	backends := model.currentBackends()
	index := slices.IndexFunc(backends, func(item config.Backend) bool {
		return item.PriceVersion > 0 && item.EffectiveOfferID() == offerID
	})
	if index < 0 {
		return config.Backend{}, false
	}
	return backends[index], true
}

func (model Model) attachableBackends(offer account.EditableOffer) []config.Backend {
	result := make([]config.Backend, 0)
	for _, item := range model.currentBackends() {
		if item.PriceVersion == 0 && providerops.Compatible(item, offer) {
			result = append(result, item)
		}
	}
	return result
}

func (model Model) compatibleOffers(item config.Backend) []account.EditableOffer {
	result := make([]account.EditableOffer, 0)
	for _, offer := range model.account.Offers {
		if _, attached := model.attachedBackend(offer.OfferID); !attached && providerops.Compatible(item, offer) {
			result = append(result, offer)
		}
	}
	return result
}

func (model Model) attachCommand(item config.Backend, offer account.EditableOffer) tea.Cmd {
	return func() tea.Msg {
		if model.dependencies.Attach == nil {
			return providerOperationMsg{err: errorsNew("Offer attachment is unavailable")}
		}
		return providerOperationMsg{status: "Offer attached.", err: model.dependencies.Attach(context.Background(), item.Name, offer.OfferID)}
	}
}

func (model *Model) openPricing(item config.Backend) {
	model.screen, model.pricingBackend, model.err, model.busy = ScreenPricing, item, nil, false
	if backendComputeOnly(item) {
		model.priceStep = priceStepCompute
		model.priceInput.SetValue("0")
		model.priceOutput.SetValue("0")
		model.priceCompute.Focus()
	} else {
		model.priceStep = priceStepInput
		model.priceInput.Focus()
	}
}

func (model Model) advancePricing() (Model, tea.Cmd) {
	switch model.priceStep {
	case priceStepInput:
		if strings.TrimSpace(model.priceInput.Value()) == "" {
			model.err = errorsNew("Enter an input token price")
			return model, nil
		}
		model.priceInput.Blur()
		model.priceOutput.Focus()
		model.priceStep, model.err = priceStepOutput, nil
	case priceStepOutput:
		if strings.TrimSpace(model.priceOutput.Value()) == "" {
			model.err = errorsNew("Enter an output token price")
			return model, nil
		}
		model.priceOutput.Blur()
		model.priceCompute.Focus()
		model.priceStep, model.err = priceStepCompute, nil
	case priceStepCompute:
		if strings.TrimSpace(model.priceCompute.Value()) == "" {
			model.err = errorsNew("Enter a compute price")
			return model, nil
		}
		model.priceCompute.Blur()
		model.priceStep, model.err = priceStepReview, nil
	case priceStepReview:
		if model.dependencies.Publish == nil {
			model.err = errorsNew("Offer publishing is unavailable")
			return model, nil
		}
		backend := model.pricingBackend
		rates := providerops.Rates{InputPerMillionMON: model.priceInput.Value(), OutputPerMillionMON: model.priceOutput.Value(), ComputePerSecondMON: model.priceCompute.Value()}
		model.busy, model.err = true, nil
		return model, func() tea.Msg {
			return providerOperationMsg{err: model.dependencies.Publish(context.Background(), backend, rates)}
		}
	}
	return model, nil
}

func (model *Model) clearPricing() {
	model.priceInput.SetValue("")
	model.priceOutput.SetValue("")
	model.priceCompute.SetValue("")
	model.priceInput.Blur()
	model.priceOutput.Blur()
	model.priceCompute.Blur()
}

func (model Model) accountCommand() tea.Cmd {
	if model.dependencies.Account == nil {
		return nil
	}
	return func() tea.Msg {
		value, err := model.dependencies.Account(context.Background())
		return accountMsg{account: value, err: err}
	}
}

func (model *Model) applyAccount(message accountMsg) {
	model.accountErr = message.err
	if model.screen == ScreenStatus {
		if message.err == nil {
			model.account = message.account
		}
		return
	}
	model.busy = false
	model.err = message.err
	if message.err == nil {
		model.account = message.account
	}
}

func visibleRequests(requests []provider.RequestStatus, limit int) []provider.RequestStatus {
	visible := make([]provider.RequestStatus, 0, min(limit, len(requests)))
	for _, request := range requests {
		if (request.State == "active" || request.State == "settling") && len(visible) < limit {
			visible = append(visible, request)
		}
	}
	for _, request := range requests {
		if request.State != "active" && request.State != "settling" && len(visible) < limit {
			visible = append(visible, request)
		}
	}
	return visible
}

func backendComputeOnly(item config.Backend) bool {
	return item.Kind == "kimi" || ((item.Kind == "codex" || item.Kind == "claude") && item.Image != "")
}

func formatWei(value string) string {
	integer, ok := new(big.Int).SetString(value, 10)
	if !ok || integer.Sign() < 0 {
		return "0"
	}
	whole, fraction := new(big.Int), new(big.Int)
	whole.QuoRem(integer, new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil), fraction)
	if fraction.Sign() == 0 {
		return whole.String()
	}
	fractionText := fraction.String()
	fractionText = strings.Repeat("0", 18-len(fractionText)) + fractionText
	fractionText = strings.TrimRight(fractionText, "0")
	return whole.String() + "." + fractionText
}

func (model Model) Resize(width, height int) Model {
	model.width, model.height = width, height
	inputWidth := width - 8
	if inputWidth < 20 {
		inputWidth = 20
	}
	model.apiURL.SetWidth(inputWidth)
	model.apiKey.SetWidth(inputWidth)
	model.apiModel.SetWidth(inputWidth)
	model.priceInput.SetWidth(inputWidth)
	model.priceOutput.SetWidth(inputWidth)
	model.priceCompute.SetWidth(inputWidth)
	model.depositAmount.SetWidth(inputWidth)
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
