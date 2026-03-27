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

package fieldtypes

import "testing"

func TestMapGraphQLFieldType(t *testing.T) {
	tests := []struct {
		name     string
		typename string
		want     string
	}{
		{
			name:     "text field",
			typename: "AspectTextField",
			want:     "text",
		},
		{
			name:     "html/longtext field",
			typename: "AspectHtmlField",
			want:     "longtext",
		},
		{
			name:     "number field",
			typename: "AspectNumberField",
			want:     "number",
		},
		{
			name:     "decimal field",
			typename: "AspectDecimalField",
			want:     "decimal",
		},
		{
			name:     "boolean field",
			typename: "AspectBooleanField",
			want:     "bool",
		},
		{
			name:     "date field",
			typename: "AspectDateField",
			want:     "date",
		},
		{
			name:     "datetime field",
			typename: "AspectDateTimeField",
			want:     "datetime",
		},
		{
			name:     "dropdown/picklist field",
			typename: "AspectPicklistField",
			want:     "dropdown",
		},
		{
			name:     "multiselect field",
			typename: "AspectMultiPicklistField",
			want:     "multi_select",
		},
		{
			name:     "user field",
			typename: "AspectUserField",
			want:     "user",
		},
		{
			name:     "group field",
			typename: "AspectGroupField",
			want:     "group",
		},
		{
			name:     "attachment field",
			typename: "AspectAttachmentField",
			want:     "attachment",
		},
		{
			name:     "json field",
			typename: "AspectJsonField",
			want:     "json",
		},
		{
			name:     "calculated field",
			typename: "AspectCalculatedField",
			want:     "calculated",
		},
		{
			name:     "handle field (system ID)",
			typename: "AspectHandleField",
			want:     "handle",
		},
		{
			name:     "relation field",
			typename: "AspectRelationField",
			want:     "relation",
		},
		{
			name:     "unknown field type returns typename",
			typename: "AspectUnknownField",
			want:     "AspectUnknownField",
		},
		{
			name:     "empty typename",
			typename: "",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapGraphQLFieldType(tt.typename)
			if got != tt.want {
				t.Errorf("MapGraphQLFieldType(%q) = %q, want %q", tt.typename, got, tt.want)
			}
		})
	}
}
