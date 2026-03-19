// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"testing"
)

func TestExtractEIFlags(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantFlags    EIFlags
		wantFiltered []string
	}{
		{
			name:         "no ei flags",
			args:         []string{"-out=tfplan", "-var=foo=bar"},
			wantFlags:    EIFlags{},
			wantFiltered: []string{"-out=tfplan", "-var=foo=bar"},
		},
		{
			name:         "profile with equals",
			args:         []string{"--profile=staging", "-out=tfplan"},
			wantFlags:    EIFlags{Profile: "staging"},
			wantFiltered: []string{"-out=tfplan"},
		},
		{
			name:         "profile with space",
			args:         []string{"--profile", "staging", "-out=tfplan"},
			wantFlags:    EIFlags{Profile: "staging"},
			wantFiltered: []string{"-out=tfplan"},
		},
		{
			name:         "log-level with equals",
			args:         []string{"--log-level=debug", "-out=tfplan"},
			wantFlags:    EIFlags{LogLevel: "debug"},
			wantFiltered: []string{"-out=tfplan"},
		},
		{
			name:         "log-level with space",
			args:         []string{"--log-level", "trace", "-out=tfplan"},
			wantFlags:    EIFlags{LogLevel: "trace"},
			wantFiltered: []string{"-out=tfplan"},
		},
		{
			name:         "debug flag",
			args:         []string{"--debug", "-out=tfplan"},
			wantFlags:    EIFlags{Debug: true},
			wantFiltered: []string{"-out=tfplan"},
		},
		{
			name:         "quiet flag long",
			args:         []string{"--quiet", "-out=tfplan"},
			wantFlags:    EIFlags{Quiet: true},
			wantFiltered: []string{"-out=tfplan"},
		},
		{
			name:         "quiet flag short",
			args:         []string{"-q", "-out=tfplan"},
			wantFlags:    EIFlags{Quiet: true},
			wantFiltered: []string{"-out=tfplan"},
		},
		{
			name: "multiple ei flags",
			args: []string{"--profile", "prod", "--debug", "-var=a=b"},
			wantFlags: EIFlags{
				Profile: "prod",
				Debug:   true,
			},
			wantFiltered: []string{"-var=a=b"},
		},
		{
			name: "all ei flags",
			args: []string{"--profile=staging", "--log-level=trace", "--debug", "-q", "-out=plan"},
			wantFlags: EIFlags{
				Profile:  "staging",
				LogLevel: "trace",
				Debug:    true,
				Quiet:    true,
			},
			wantFiltered: []string{"-out=plan"},
		},
		{
			name:         "ei flags mixed with terraform flags",
			args:         []string{"-var=a=b", "--profile", "test", "-var=c=d", "--debug"},
			wantFlags:    EIFlags{Profile: "test", Debug: true},
			wantFiltered: []string{"-var=a=b", "-var=c=d"},
		},
		{
			name:         "empty args",
			args:         []string{},
			wantFlags:    EIFlags{},
			wantFiltered: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFlags, gotFiltered := extractEIFlags(tt.args)

			if gotFlags.Profile != tt.wantFlags.Profile {
				t.Errorf("extractEIFlags() Profile = %q, want %q", gotFlags.Profile, tt.wantFlags.Profile)
			}
			if gotFlags.LogLevel != tt.wantFlags.LogLevel {
				t.Errorf("extractEIFlags() LogLevel = %q, want %q", gotFlags.LogLevel, tt.wantFlags.LogLevel)
			}
			if gotFlags.Debug != tt.wantFlags.Debug {
				t.Errorf("extractEIFlags() Debug = %v, want %v", gotFlags.Debug, tt.wantFlags.Debug)
			}
			if gotFlags.Quiet != tt.wantFlags.Quiet {
				t.Errorf("extractEIFlags() Quiet = %v, want %v", gotFlags.Quiet, tt.wantFlags.Quiet)
			}

			if len(gotFiltered) != len(tt.wantFiltered) {
				t.Errorf("extractEIFlags() filtered len = %d, want %d", len(gotFiltered), len(tt.wantFiltered))
				return
			}

			for i, arg := range gotFiltered {
				if arg != tt.wantFiltered[i] {
					t.Errorf("extractEIFlags() filtered[%d] = %q, want %q", i, arg, tt.wantFiltered[i])
				}
			}
		})
	}
}
