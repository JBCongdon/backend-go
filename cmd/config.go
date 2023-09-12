/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	profileName = iota
	clientID
	clientSecret
	oidcEndpoint
)

const (
	hotPink  = lipgloss.Color("#FF06B7")
	darkGray = lipgloss.Color("#767676")
)

var (
	inputStyle    = lipgloss.NewStyle().Foreground(hotPink)
	continueStyle = lipgloss.NewStyle().Foreground(darkGray)
)

type model struct {
	inputs  []textinput.Model
	focused int
	err     error
}

type (
	errMsg error
)

type Config struct {
	Profiles map[string]OpenTDFConfig `toml:"profiles"`
}

type OpenTDFConfig struct {
	OidcEndpoint string `toml:"oidcEndpoint"`
	ClientID     string `toml:"clientID"`
	ClientSecret string `toml:"clientSecret,omitempty"`
}

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure the TDF CLI",
	Run: func(cmd *cobra.Command, args []string) {
		p := tea.NewProgram(initialModel())
		m, err := p.Run()
		if err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}

		// Assert the final tea.Model to our local model and print the choice.
		if _, ok := m.(model); !ok {
			fmt.Print("can't assert model")
			os.Exit(1)
		}

		otdfConfig := OpenTDFConfig{
			OidcEndpoint: m.(model).inputs[oidcEndpoint].Value(),
			ClientID:     m.(model).inputs[clientID].Value(),
		}
		if m.(model).inputs[clientSecret].Value() != "" {
			otdfConfig.ClientSecret = m.(model).inputs[clientSecret].Value()
		}
		config := Config{
			Profiles: map[string]OpenTDFConfig{
				m.(model).inputs[profileName].Value(): otdfConfig,
			},
		}

		tomlConfig, err := toml.Marshal(&config)
		if err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}
		tomlReader := bytes.NewReader(tomlConfig)
		fmt.Println(string(tomlConfig))
		viper.ReadConfig(tomlReader)
		homedir, err := os.UserHomeDir()
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		if _, err := os.Stat(fmt.Sprintf("%s/.opentdf", homedir)); errors.Is(err, os.ErrNotExist) {
			err := os.Mkdir(fmt.Sprintf("%s/.opentdf", homedir), os.ModePerm)
			if err != nil {
				log.Println(err)
				os.Exit(1)
			}
		}
		if err := viper.WriteConfig(); err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}

	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// configCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// configCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func initialModel() model {
	var inputs []textinput.Model = make([]textinput.Model, 4)
	inputs[profileName] = textinput.New()
	inputs[profileName].Placeholder = "Profile Name"
	inputs[profileName].Focus()
	inputs[profileName].CharLimit = 156
	inputs[profileName].Width = 20
	inputs[oidcEndpoint] = textinput.New()
	inputs[oidcEndpoint].Placeholder = "OIDC Endpoint"
	inputs[oidcEndpoint].CharLimit = 156
	inputs[oidcEndpoint].Width = 100
	inputs[clientID] = textinput.New()
	inputs[clientID].Placeholder = "Client ID"
	inputs[clientID].CharLimit = 156
	inputs[clientID].Width = 20
	inputs[clientSecret] = textinput.New()
	inputs[clientSecret].Placeholder = "Client Secret"
	inputs[clientSecret].CharLimit = 156
	inputs[clientSecret].Width = 40

	return model{
		inputs: inputs,
	}
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd = make([]tea.Cmd, len(m.inputs))
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.focused == len(m.inputs)-1 {
				return m, tea.Quit
			}
			m.nextInput()
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyShiftTab, tea.KeyCtrlP:
			m.prevInput()
		case tea.KeyTab, tea.KeyCtrlN:
			m.nextInput()
		}
		for i := range m.inputs {
			m.inputs[i].Blur()
		}
		m.inputs[m.focused].Focus()

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	// The header
	return fmt.Sprintf(
		`
 %s  %s
 %s  %s
 %s  %s
 %s  %s

 %s
`,
		inputStyle.Width(24).Render("Profile Name"),
		m.inputs[profileName].View(),
		inputStyle.Width(24).Render("OIDC Endpoint"),
		m.inputs[oidcEndpoint].View(),
		inputStyle.Width(24).Render("Client ID"),
		m.inputs[clientID].View(),
		inputStyle.Width(24).Render("Client Secret"),
		m.inputs[clientSecret].View(),
		continueStyle.Render("Submit ->"),
	) + "\n"
}

// nextInput focuses the next input field
func (m *model) nextInput() {
	m.focused = (m.focused + 1) % len(m.inputs)
}

// prevInput focuses the previous input field
func (m *model) prevInput() {
	m.focused--
	// Wrap around
	if m.focused < 0 {
		m.focused = len(m.inputs) - 1
	}
}
