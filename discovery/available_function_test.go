// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package discovery

import (
	"testing"
)

func TestAvailableFunction_TypeDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		funcType string
		want     string
	}{
		{
			name:     "procedure",
			funcType: "PROCEDURE",
			want:     "Procedure",
		},
		{
			name:     "udf",
			funcType: "USER_DEFINED_FUNCTION",
			want:     "UDF",
		},
		{
			name:     "unknown",
			funcType: "UNKNOWN",
			want:     "UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			af := AvailableFunction{Type: tt.funcType}
			if got := af.TypeDisplayName(); got != tt.want {
				t.Errorf("TypeDisplayName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAvailableFunction_FullyQualifiedName(t *testing.T) {
	tests := []struct {
		name         string
		databaseName string
		schemaName   string
		funcName     string
		want         string
	}{
		{
			name:         "full path",
			databaseName: "MYDB",
			schemaName:   "PUBLIC",
			funcName:     "MY_FUNC",
			want:         "MYDB.PUBLIC.MY_FUNC",
		},
		{
			name:         "database only",
			databaseName: "MYDB",
			schemaName:   "",
			funcName:     "MY_FUNC",
			want:         "MYDB.MY_FUNC",
		},
		{
			name:         "schema only",
			databaseName: "",
			schemaName:   "PUBLIC",
			funcName:     "MY_FUNC",
			want:         "PUBLIC.MY_FUNC",
		},
		{
			name:         "name only",
			databaseName: "",
			schemaName:   "",
			funcName:     "MY_FUNC",
			want:         "MY_FUNC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			af := AvailableFunction{
				Name:         tt.funcName,
				DatabaseName: tt.databaseName,
				SchemaName:   tt.schemaName,
			}
			if got := af.FullyQualifiedName(); got != tt.want {
				t.Errorf("FullyQualifiedName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAvailableFunction_ArgsDisplay(t *testing.T) {
	tests := []struct {
		name    string
		minArgs int
		maxArgs int
		want    string
	}{
		{
			name:    "no args",
			minArgs: 0,
			maxArgs: 0,
			want:    "0",
		},
		{
			name:    "fixed args",
			minArgs: 2,
			maxArgs: 2,
			want:    "2",
		},
		{
			name:    "variable args",
			minArgs: 1,
			maxArgs: 3,
			want:    "1-3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			af := AvailableFunction{MinArgs: tt.minArgs, MaxArgs: tt.maxArgs}
			if got := af.ArgsDisplay(); got != tt.want {
				t.Errorf("ArgsDisplay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAvailableFunction_ParametersDisplay(t *testing.T) {
	tests := []struct {
		name   string
		params []AvailableFunctionParameter
		want   string
	}{
		{
			name:   "no params",
			params: nil,
			want:   "()",
		},
		{
			name:   "empty params",
			params: []AvailableFunctionParameter{},
			want:   "()",
		},
		{
			name: "single param with name",
			params: []AvailableFunctionParameter{
				{Name: "input", Type: "VARCHAR"},
			},
			want: "(input: VARCHAR)",
		},
		{
			name: "single param without name",
			params: []AvailableFunctionParameter{
				{Name: "", Type: "VARCHAR"},
			},
			want: "(VARCHAR)",
		},
		{
			name: "multiple params",
			params: []AvailableFunctionParameter{
				{Name: "id", Type: "NUMBER"},
				{Name: "name", Type: "VARCHAR"},
				{Name: "", Type: "BOOLEAN"},
			},
			want: "(id: NUMBER, name: VARCHAR, BOOLEAN)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			af := AvailableFunction{Parameters: tt.params}
			if got := af.ParametersDisplay(); got != tt.want {
				t.Errorf("ParametersDisplay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertAvailableFunctionData(t *testing.T) {
	// Import the client package types would be needed here
	// For now, test the discovery types directly
	af := AvailableFunction{
		Name:          "TEST_PROC",
		DatabaseName:  "TESTDB",
		SchemaName:    "PUBLIC",
		Type:          "PROCEDURE",
		Description:   "A test procedure",
		MinArgs:       1,
		MaxArgs:       3,
		BuiltIn:       false,
		Secure:        true,
		TableFunction: false,
		Parameters: []AvailableFunctionParameter{
			{Name: "input", Type: "VARCHAR"},
		},
		ReturnType: "VARCHAR",
	}

	// Verify all fields are accessible
	if af.Name != "TEST_PROC" {
		t.Errorf("Name = %v, want TEST_PROC", af.Name)
	}
	if af.TypeDisplayName() != "Procedure" {
		t.Errorf("TypeDisplayName() = %v, want Procedure", af.TypeDisplayName())
	}
	if af.FullyQualifiedName() != "TESTDB.PUBLIC.TEST_PROC" {
		t.Errorf("FullyQualifiedName() = %v, want TESTDB.PUBLIC.TEST_PROC", af.FullyQualifiedName())
	}
	if af.ArgsDisplay() != "1-3" {
		t.Errorf("ArgsDisplay() = %v, want 1-3", af.ArgsDisplay())
	}
	if af.ParametersDisplay() != "(input: VARCHAR)" {
		t.Errorf("ParametersDisplay() = %v, want (input: VARCHAR)", af.ParametersDisplay())
	}
}
