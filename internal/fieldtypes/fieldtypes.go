// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package fieldtypes

// MapGraphQLFieldType maps GraphQL typename to field type identifier.
func MapGraphQLFieldType(typename string) string {
	typeMap := map[string]string{
		"AspectTextField":          "text",
		"AspectHtmlField":          "longtext",
		"AspectNumberField":        "number",
		"AspectDecimalField":       "decimal",
		"AspectBooleanField":       "bool",
		"AspectDateField":          "date",
		"AspectDateTimeField":      "datetime",
		"AspectPicklistField":      "dropdown",
		"AspectMultiPicklistField": "multi_select",
		"AspectUserField":          "user",
		"AspectGroupField":         "group",
		"AspectAttachmentField":    "attachment",
		"AspectJsonField":          "json",
		"AspectCalculatedField":    "calculated",
		"AspectHandleField":        "handle",
		"AspectRelationField":      "relation",
	}
	if mappedType, ok := typeMap[typename]; ok {
		return mappedType
	}
	return typename
}
