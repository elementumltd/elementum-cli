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

package export

import (
	"fmt"
	"strings"

	"github.com/elementumltd/elementum-cli/discovery"
)

// StripNulls removes attributes with null values from all blocks.
// Replaces the regex-based stripNullAttributes() in beautify.go.
func StripNulls(blocks []*HCLBlock) {
	for _, block := range blocks {
		stripNullsFromBody(block.Body)
		// Recurse into nested blocks
		for _, nested := range block.Body.Blocks {
			StripNulls([]*HCLBlock{nested})
		}
	}
}

func stripNullsFromBody(body *HCLBody) {
	if body == nil {
		return
	}
	filtered := make([]*HCLAttribute, 0, len(body.Attributes))
	for _, attr := range body.Attributes {
		if _, isNull := attr.Value.(HCLNull); isNull {
			continue
		}
		// Also strip nulls inside objects
		if obj, ok := attr.Value.(HCLObject); ok {
			stripNullsFromObject(&obj)
			attr.Value = obj
		}
		// Strip nulls inside lists
		if list, ok := attr.Value.(HCLList); ok {
			stripNullsFromList(&list)
			attr.Value = list
		}
		filtered = append(filtered, attr)
	}
	body.Attributes = filtered
}

func stripNullsFromObject(obj *HCLObject) {
	filtered := make([]*HCLAttribute, 0, len(obj.Attributes))
	for _, attr := range obj.Attributes {
		if _, isNull := attr.Value.(HCLNull); isNull {
			continue
		}
		filtered = append(filtered, attr)
	}
	obj.Attributes = filtered
}

func stripNullsFromList(list *HCLList) {
	for i, val := range list.Values {
		if obj, ok := val.(HCLObject); ok {
			stripNullsFromObject(&obj)
			list.Values[i] = obj
		}
	}
}

// StripEmptyObjects removes attributes whose values are empty objects ({}).
// Replaces the regex-based stripEmptyFlowActions() for empty action = {} blocks.
func StripEmptyObjects(blocks []*HCLBlock) {
	for _, block := range blocks {
		stripEmptyObjectsFromBody(block.Body)
		for _, nested := range block.Body.Blocks {
			StripEmptyObjects([]*HCLBlock{nested})
		}
	}
}

func stripEmptyObjectsFromBody(body *HCLBody) {
	if body == nil {
		return
	}
	filtered := make([]*HCLAttribute, 0, len(body.Attributes))
	for _, attr := range body.Attributes {
		if obj, ok := attr.Value.(HCLObject); ok && len(obj.Attributes) == 0 {
			continue
		}
		filtered = append(filtered, attr)
	}
	body.Attributes = filtered
}

// ResolveUUIDs walks all blocks and replaces string values that match UUIDs
// with their corresponding Terraform references. Handles:
//   - Simple string attributes: "uuid" -> resource.name.id
//   - Strings inside lists: ["uuid1", "uuid2"] -> [ref1, ref2]
//   - Strings inside objects: { field_id = "uuid" } -> { field_id = ref }
//
// Replaces the regex-based replaceUUIDs() and beautifyFieldIDArrays() in beautify.go.
func ResolveUUIDs(blocks []*HCLBlock, uuidMap map[string]string) {
	for _, block := range blocks {
		resolveUUIDsInBody(block.Body, uuidMap)
		for _, nested := range block.Body.Blocks {
			ResolveUUIDs([]*HCLBlock{nested}, uuidMap)
		}
	}
}

func resolveUUIDsInBody(body *HCLBody, uuidMap map[string]string) {
	if body == nil {
		return
	}
	for _, attr := range body.Attributes {
		attr.Value = resolveUUIDsInValue(attr.Value, uuidMap)
	}
}

func resolveUUIDsInValue(val HCLValue, uuidMap map[string]string) HCLValue {
	switch v := val.(type) {
	case HCLString:
		if ref, ok := uuidMap[v.Value]; ok {
			return Ref(ref)
		}
		return v
	case HCLList:
		for i, elem := range v.Values {
			v.Values[i] = resolveUUIDsInValue(elem, uuidMap)
		}
		return v
	case HCLObject:
		for _, attr := range v.Attributes {
			attr.Value = resolveUUIDsInValue(attr.Value, uuidMap)
		}
		return v
	default:
		return val
	}
}

// FixSelfReferences removes or fixes attributes that reference the block's own resource.
// For example, source_id = elementum_table.foo.id inside resource "elementum_table" "foo"
// should be stripped or set to "" since it creates a self-dependency cycle.
//
// Replaces the regex-based fixSelfReferences() in beautify.go.
func FixSelfReferences(blocks []*HCLBlock) {
	for _, block := range blocks {
		if block.Type != "resource" || len(block.Labels) < 2 {
			continue
		}

		selfRef := block.Labels[0] + "." + block.Labels[1] + ".id"
		fixSelfRefsInBody(block.Body, selfRef)
	}
}

func fixSelfRefsInBody(body *HCLBody, selfRef string) {
	if body == nil {
		return
	}
	filtered := make([]*HCLAttribute, 0, len(body.Attributes))
	for _, attr := range body.Attributes {
		if isSelfRef(attr.Value, selfRef) {
			// For source_id, strip entirely
			if attr.Name == "source_id" {
				continue
			}
			// For other fields, set to empty string
			attr.Value = Str("")
		}
		filtered = append(filtered, attr)
	}
	body.Attributes = filtered
}

func isSelfRef(val HCLValue, selfRef string) bool {
	switch v := val.(type) {
	case HCLReference:
		return v.Ref == selfRef
	case HCLRaw:
		return v.Text == selfRef
	default:
		return false
	}
}

// InjectWarnings walks blocks and adds comments for attributes that may need attention.
// Handles AI provider connector IDs and search table IDs that are hardcoded UUIDs.
func InjectWarnings(blocks []*HCLBlock) {
	for _, block := range blocks {
		injectWarningsInBody(block)
		for _, nested := range block.Body.Blocks {
			InjectWarnings([]*HCLBlock{nested})
		}
	}
}

func injectWarningsInBody(block *HCLBlock) {
	if block.Body == nil {
		return
	}
	for _, attr := range block.Body.Attributes {
		switch attr.Name {
		case "ai_provider_connector_id":
			if str, ok := attr.Value.(HCLString); ok && isUUID(str.Value) {
				attr.Comment = "TODO: Replace with data source or variable"
			}
		case "search_table_id":
			if str, ok := attr.Value.(HCLString); ok && isUUID(str.Value) {
				attr.Comment = "TODO: Replace with data source or variable"
			}
		}
	}
}

// MergeBlocks merges two sets of blocks, replacing blocks from `base` with
// blocks from `override` when they have matching resource type + name.
// This replaces InjectHydrated*Configs() brace-counting logic:
// tofu-generated blocks are the base, CLI-generated blocks are the overrides.
func MergeBlocks(base, override []*HCLBlock) []*HCLBlock {
	// Index overrides by key
	overrideMap := make(map[string]*HCLBlock)
	for _, b := range override {
		key := blockKey(b)
		if key != "" {
			overrideMap[key] = b
		}
	}

	// Replace base blocks with overrides where available
	var result []*HCLBlock
	seen := make(map[string]bool)
	for _, b := range base {
		key := blockKey(b)
		if key != "" {
			if overrideBlock, ok := overrideMap[key]; ok {
				result = append(result, overrideBlock)
				seen[key] = true
				continue
			}
		}
		result = append(result, b)
		if key != "" {
			seen[key] = true
		}
	}

	// Append any override blocks not already in base
	for _, b := range override {
		key := blockKey(b)
		if key != "" && !seen[key] {
			result = append(result, b)
		}
	}

	return result
}

func blockKey(b *HCLBlock) string {
	if len(b.Labels) >= 2 {
		return b.Type + "." + b.Labels[0] + "." + b.Labels[1]
	}
	return ""
}

// isUUID checks if a string looks like a UUID
func isUUID(s string) bool {
	// Simple check: 8-4-4-4-12 hex chars
	if len(s) != 36 {
		return false
	}
	for i, ch := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if ch != '-' {
				return false
			}
		} else {
			if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
				return false
			}
		}
	}
	return true
}

// SetStageIDForInitiateLayouts sets stage_id = "" for "Initiate" layouts
// (their stage is platform-managed and should not be imported).
func SetStageIDForInitiateLayouts(blocks []*HCLBlock, app *discovery.App, imports []ImportBlock) {
	if app == nil {
		return
	}

	// Find Initiate layout resource names
	initiateLayouts := make(map[string]bool)
	for _, layout := range app.AllLayouts() {
		if layout.Name == "Initiate" || layout.IsInitiate {
			for _, imp := range imports {
				if imp.ResourceType == "elementum_layout" && strings.Contains(imp.ID, layout.ID) {
					initiateLayouts[imp.ResourceName] = true
					break
				}
			}
		}
	}

	for _, block := range blocks {
		if block.Type != "resource" || len(block.Labels) < 2 {
			continue
		}
		if block.Labels[0] != "elementum_layout" {
			continue
		}
		if !initiateLayouts[block.Labels[1]] {
			continue
		}

		// Set stage_id = ""
		found := false
		for _, attr := range block.Body.Attributes {
			if attr.Name == "stage_id" {
				attr.Value = Str("")
				found = true
				break
			}
		}
		if !found {
			block.SetAttr("stage_id", Str(""))
		}
	}
}

// RemoveLayoutCommentsBlocks removes the "comments" nested blocks from layout resources.
// Comments are platform-managed and should not be exported.
func RemoveLayoutCommentsBlocks(blocks []*HCLBlock) {
	for _, block := range blocks {
		if block.Type != "resource" || len(block.Labels) < 2 {
			continue
		}
		if block.Labels[0] != "elementum_layout" {
			continue
		}

		// Remove comments attribute
		filtered := make([]*HCLAttribute, 0, len(block.Body.Attributes))
		for _, attr := range block.Body.Attributes {
			if attr.Name == "comments" {
				continue
			}
			filtered = append(filtered, attr)
		}
		block.Body.Attributes = filtered

		// Remove comments nested blocks
		filteredBlocks := make([]*HCLBlock, 0, len(block.Body.Blocks))
		for _, nested := range block.Body.Blocks {
			if nested.Type == "comments" {
				continue
			}
			filteredBlocks = append(filteredBlocks, nested)
		}
		block.Body.Blocks = filteredBlocks
	}
}

// ApplyAllTransforms runs the standard set of IR transforms in the correct order.
// This is the IR equivalent of BeautifyConfig().
func ApplyAllTransforms(blocks []*HCLBlock, uuidMap map[string]string, app *discovery.App, imports []ImportBlock) {
	StripNulls(blocks)
	StripEmptyObjects(blocks)

	if app != nil {
		StripBaseTableReferenceFieldIDs(blocks, app, imports)
	}

	ResolveUUIDs(blocks, uuidMap)
	FixSelfReferences(blocks)
	InjectWarnings(blocks)

	if app != nil {
		SetStageIDForInitiateLayouts(blocks, app, imports)
		RemoveLayoutCommentsBlocks(blocks)
	}
}

// LowercaseFieldValue lowercases the string value of a specific attribute across all blocks.
// Used for fields like "status", "length" that need to be lowercase.
func LowercaseFieldValue(blocks []*HCLBlock, fieldName string) {
	for _, block := range blocks {
		for _, attr := range block.Body.Attributes {
			if attr.Name == fieldName {
				if str, ok := attr.Value.(HCLString); ok {
					attr.Value = Str(strings.ToLower(str.Value))
				}
			}
		}
	}
}

// UppercaseFieldValue uppercases the string value of a specific attribute across all blocks.
// Used for fields like "field_type", "variable_type" that need to be uppercase.
func UppercaseFieldValue(blocks []*HCLBlock, fieldName string) {
	for _, block := range blocks {
		for _, attr := range block.Body.Attributes {
			if attr.Name == fieldName {
				if str, ok := attr.Value.(HCLString); ok {
					attr.Value = Str(strings.ToUpper(str.Value))
				}
			}
		}
	}
}

// StripBaseTableReferenceFieldIDs removes reference_field_id from base table blocks.
// A base table is one that is not a derived/view table.
func StripBaseTableReferenceFieldIDs(blocks []*HCLBlock, app *discovery.App, imports []ImportBlock) {
	if app == nil {
		return
	}

	// Build set of derived table resource names (tables with a valid source table)
	discoveredTableIDs := make(map[string]bool)
	for _, t := range app.DiscoveredTables {
		discoveredTableIDs[t.ID] = true
	}

	baseTableNames := make(map[string]bool)
	for _, table := range app.DiscoveredTables {
		// A table is a "base" table if it has no source or its source is itself or not a discovered table
		isBase := table.SourceID == "" || table.SourceID == table.ID || !discoveredTableIDs[table.SourceID]
		if isBase {
			resourceName := findResourceName(imports, "elementum_table", table.ID)
			if resourceName != "" {
				baseTableNames[resourceName] = true
			}
		}
	}

	for _, block := range blocks {
		if block.Type != "resource" || len(block.Labels) < 2 {
			continue
		}
		if block.Labels[0] != "elementum_table" {
			continue
		}
		if !baseTableNames[block.Labels[1]] {
			continue
		}

		// Strip reference_field_id
		filtered := make([]*HCLAttribute, 0, len(block.Body.Attributes))
		for _, attr := range block.Body.Attributes {
			if attr.Name == "reference_field_id" {
				continue
			}
			filtered = append(filtered, attr)
		}
		block.Body.Attributes = filtered
	}
}

// AddLocalsBlock creates a locals block with app_name and namespace for convenience.
func AddLocalsBlock(blocks []*HCLBlock, appResourceName string) []*HCLBlock {
	if appResourceName == "" {
		return blocks
	}

	locals := &HCLBlock{
		Type: "locals",
		Body: &HCLBody{
			Attributes: []*HCLAttribute{
				{Name: "app_name", Value: Ref(fmt.Sprintf("elementum_app.%s.name", appResourceName))},
				{Name: "namespace", Value: Ref(fmt.Sprintf("elementum_app.%s.namespace", appResourceName))},
			},
		},
	}

	return append([]*HCLBlock{locals}, blocks...)
}
