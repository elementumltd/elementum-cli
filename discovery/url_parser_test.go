// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package discovery

import (
	"testing"
)

func TestParseResourceFromURL(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantType       string
		wantIdentifier string
		wantErr        bool
	}{
		// App URL patterns
		{
			name:           "app full URL with path",
			input:          "https://org.elementum.io/app/accountassignment/records",
			wantType:       "App",
			wantIdentifier: "accountassignment",
		},
		{
			name:           "apps plural URL",
			input:          "https://org.elementum.io/apps/myapp/settings",
			wantType:       "App",
			wantIdentifier: "myapp",
		},
		{
			name:           "admin apps URL",
			input:          "https://org.elementum.io/admin/apps/testapp/fields",
			wantType:       "App",
			wantIdentifier: "testapp",
		},
		{
			name:           "app path only",
			input:          "/app/namespace/records",
			wantType:       "App",
			wantIdentifier: "namespace",
		},

		// Element URL patterns
		{
			name:           "element full URL",
			input:          "https://org.elementum.io/element/products/records",
			wantType:       "Element",
			wantIdentifier: "products",
		},
		{
			name:           "elements plural URL",
			input:          "https://org.elementum.io/elements/customers/settings",
			wantType:       "Element",
			wantIdentifier: "customers",
		},
		{
			name:           "admin elements URL",
			input:          "https://org.elementum.io/admin/elements/vendors/fields",
			wantType:       "Element",
			wantIdentifier: "vendors",
		},

		// Task URL patterns
		{
			name:           "task full URL",
			input:          "https://org.elementum.io/task/action_items/records",
			wantType:       "Task",
			wantIdentifier: "action_items",
		},
		{
			name:           "tasks plural URL",
			input:          "https://org.elementum.io/tasks/todos/settings",
			wantType:       "Task",
			wantIdentifier: "todos",
		},
		{
			name:           "admin tasks URL",
			input:          "https://org.elementum.io/admin/tasks/workflows/fields",
			wantType:       "Task",
			wantIdentifier: "workflows",
		},

		// Table URL patterns
		{
			name:           "admin tables URL",
			input:          "https://org.elementum.io/admin/tables/customers",
			wantType:       "Table",
			wantIdentifier: "customers",
		},
		{
			name:           "tables URL",
			input:          "https://org.elementum.io/tables/sales_data/columns",
			wantType:       "Table",
			wantIdentifier: "sales_data",
		},

		// CloudLink URL patterns
		{
			name:           "admin cloudlinks URL",
			input:          "https://org.elementum.io/admin/cloudlinks/production_api",
			wantType:       "CloudLink",
			wantIdentifier: "production_api",
		},
		{
			name:           "cloudlinks URL",
			input:          "https://org.elementum.io/cloudlinks/snowflake_prod/settings",
			wantType:       "CloudLink",
			wantIdentifier: "snowflake_prod",
		},

		// Different environments
		{
			name:           "stage environment URL",
			input:          "https://org.stage.elementum.io/app/myapp/records",
			wantType:       "App",
			wantIdentifier: "myapp",
		},
		{
			name:           "dev environment URL",
			input:          "https://org.dev.elementum.io/elements/products/records",
			wantType:       "Element",
			wantIdentifier: "products",
		},

		// Edge cases
		{
			name:           "URL with query parameters",
			input:          "https://org.elementum.io/app/myapp/records?filter=active",
			wantType:       "App",
			wantIdentifier: "myapp",
		},
		{
			name:           "URL with fragment",
			input:          "https://org.elementum.io/element/products/records#section",
			wantType:       "Element",
			wantIdentifier: "products",
		},
		{
			name:           "path without leading slash",
			input:          "app/namespace/records",
			wantType:       "App",
			wantIdentifier: "namespace",
		},

		// Error cases
		{
			name:    "plain text without slash",
			input:   "justanidentifier",
			wantErr: true,
		},
		{
			name:    "URL without resource type",
			input:   "https://org.elementum.io/unknown/path",
			wantErr: true,
		},
		{
			name:    "URL with resource type but no identifier",
			input:   "https://org.elementum.io/app/",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseResourceFromURL(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseResourceFromURL(%q) expected error but got none", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseResourceFromURL(%q) unexpected error: %v", tt.input, err)
				return
			}

			if result.Type != tt.wantType {
				t.Errorf("ParseResourceFromURL(%q) type = %q, want %q", tt.input, result.Type, tt.wantType)
			}

			if result.Identifier != tt.wantIdentifier {
				t.Errorf("ParseResourceFromURL(%q) identifier = %q, want %q", tt.input, result.Identifier, tt.wantIdentifier)
			}
		})
	}
}

func TestParseResourceFromURL_CaseInsensitivity(t *testing.T) {
	// Test that resource type markers are case-insensitive
	tests := []struct {
		input    string
		wantType string
	}{
		{"https://org.elementum.io/APP/myapp/records", "App"},
		{"https://org.elementum.io/App/myapp/records", "App"},
		{"https://org.elementum.io/ELEMENT/products/records", "Element"},
		{"https://org.elementum.io/TASK/todos/records", "Task"},
		{"https://org.elementum.io/TABLE/data/columns", "Table"},
		{"https://org.elementum.io/CLOUDLINK/api/settings", "CloudLink"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseResourceFromURL(tt.input)
			if err != nil {
				t.Errorf("ParseResourceFromURL(%q) unexpected error: %v", tt.input, err)
				return
			}
			if result.Type != tt.wantType {
				t.Errorf("ParseResourceFromURL(%q) type = %q, want %q", tt.input, result.Type, tt.wantType)
			}
		})
	}
}
