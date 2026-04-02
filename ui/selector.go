// Copyright 2026 Elementum Ltd. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
