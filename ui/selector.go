// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package ui

import (
	"github.com/charmbracelet/huh"
)

// SelectOption represents a selectable option
type SelectOption struct {
	Label string
	Value string
}

// MultiSelectOptions shows a multi-select prompt and returns selected values
func MultiSelectOptions(title string, options []SelectOption) ([]string, error) {
	var selected []string

	// Convert options to huh options
	huhOptions := make([]huh.Option[string], len(options))
	for i, opt := range options {
		huhOptions[i] = huh.NewOption(opt.Label, opt.Value)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title(title).
				Options(huhOptions...).
				Value(&selected),
		),
	)

	err := form.Run()
	if err != nil {
		return nil, err
	}

	return selected, nil
}

// SelectSingle shows a single-select prompt and returns the selected value
func SelectSingle(title string, options []SelectOption) (string, error) {
	var selected string

	// Convert options to huh options
	huhOptions := make([]huh.Option[string], len(options))
	for i, opt := range options {
		huhOptions[i] = huh.NewOption(opt.Label, opt.Value)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(title).
				Options(huhOptions...).
				Value(&selected),
		),
	)

	err := form.Run()
	if err != nil {
		return "", err
	}

	return selected, nil
}

// Confirm shows a confirmation prompt
func Confirm(title, description string) (bool, error) {
	var confirmed bool

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Description(description).
				Value(&confirmed),
		),
	)

	err := form.Run()
	if err != nil {
		return false, err
	}

	return confirmed, nil
}

// TextInput shows a text input prompt
func TextInput(title, placeholder string) (string, error) {
	var value string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(title).
				Placeholder(placeholder).
				Value(&value),
		),
	)

	err := form.Run()
	if err != nil {
		return "", err
	}

	return value, nil
}
