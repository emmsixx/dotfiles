package ui

import (
	"fmt"
	"os"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
)

type Choice[T comparable] struct {
	Label string
	Value T
}

func Interactive() bool {
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func Heading(text string) { fmt.Fprintln(os.Stderr, titleStyle.Render(text)) }
func OK(text string)      { fmt.Fprintln(os.Stderr, okStyle.Render("✓ "+text)) }
func Warn(text string)    { fmt.Fprintln(os.Stderr, warnStyle.Render("! "+text)) }

func Select[T comparable](title, description string, choices []Choice[T], value *T) error {
	options := make([]huh.Option[T], 0, len(choices))
	for _, choice := range choices {
		options = append(options, huh.NewOption(choice.Label, choice.Value))
	}
	return huh.NewForm(huh.NewGroup(
		huh.NewSelect[T]().Title(title).Description(description).Options(options...).Value(value),
	)).Run()
}

func MultiSelect[T comparable](title, description string, choices []Choice[T], value *[]T) error {
	options := make([]huh.Option[T], 0, len(choices))
	for _, choice := range choices {
		options = append(options, huh.NewOption(choice.Label, choice.Value))
	}
	return huh.NewForm(huh.NewGroup(
		huh.NewMultiSelect[T]().Title(title).Description(description).Options(options...).Value(value),
	)).Run()
}

func Confirm(title, description string, value *bool) error {
	return huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title(title).Description(description).Value(value),
	)).Run()
}

type Input struct {
	Title       string
	Description string
	Value       *string
	Secret      bool
}

func Inputs(title string, inputs []Input) error {
	fields := make([]huh.Field, 0, len(inputs))
	for _, input := range inputs {
		field := huh.NewInput().Title(input.Title).Description(input.Description).Value(input.Value)
		if input.Secret {
			field.EchoMode(huh.EchoModePassword)
		}
		fields = append(fields, field)
	}
	return huh.NewForm(huh.NewGroup(fields...).Title(title)).Run()
}
