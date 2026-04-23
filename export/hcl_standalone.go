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
	"github.com/elementumltd/elementum-cli/discovery"
)

// GenerateTableHCL generates HCL for a standalone table export.
func GenerateTableHCL(table *discovery.Table) string {
	gen := &AppHCLGenerator{
		uuidMap: make(map[string]string),
	}
	block := gen.generateTableIR(table)
	if block == nil {
		return ""
	}
	blocks := []*HCLBlock{block}
	StripNulls(blocks)
	return SerializeBlocks(blocks)
}

// GenerateGroupHCL generates HCL for a standalone group export.
func GenerateGroupHCL(group *discovery.Group) string {
	resourceName := SanitizeName(group.Name)
	b := NewResourceBlock("elementum_group", resourceName)

	b.SetAttr("name", Str(group.Name))

	if group.Approver {
		b.SetAttr("approver", Bool(true))
	}
	if group.Assignable {
		b.SetAttr("assignable", Bool(true))
	}
	if group.Mentionable {
		b.SetAttr("mentionable", Bool(true))
	}
	if group.Watchable {
		b.SetAttr("watchable", Bool(true))
	}
	if group.Dynamic {
		b.SetAttr("dynamic", Bool(true))
	}
	if len(group.Tags) > 0 {
		var tagVals []HCLValue
		for _, tag := range group.Tags {
			tagVals = append(tagVals, Str(tag))
		}
		b.SetAttr("tags", HCLList{Values: tagVals})
	}
	if len(group.Members) > 0 {
		var memberVals []HCLValue
		for _, m := range group.Members {
			memberVals = append(memberVals, Str(m))
		}
		b.SetAttr("members", HCLList{Values: memberVals})
	}

	blocks := []*HCLBlock{b}
	StripNulls(blocks)
	return SerializeBlocks(blocks)
}

// GenerateElementHCL generates HCL for a standalone element export.
func GenerateElementHCL(element *discovery.Element) string {
	gen := &AppHCLGenerator{
		uuidMap: make(map[string]string),
	}
	block := gen.generateElementIR(element)
	if block == nil {
		return ""
	}
	blocks := []*HCLBlock{block}
	StripNulls(blocks)
	return SerializeBlocks(blocks)
}

// GenerateAspectTaskHCL generates HCL for a standalone aspect task export.
func GenerateAspectTaskHCL(task *discovery.AspectTask) string {
	gen := &AppHCLGenerator{
		uuidMap: make(map[string]string),
	}
	block := gen.generateAspectTaskIR(task)
	if block == nil {
		return ""
	}
	blocks := []*HCLBlock{block}
	StripNulls(blocks)
	return SerializeBlocks(blocks)
}

// GenerateCloudLinkHCL generates HCL for a standalone cloudlink export.
func GenerateCloudLinkHCL(cloudlink *discovery.CloudLink) string {
	resourceName := SanitizeName(cloudlink.Name)
	b := NewResourceBlock("elementum_cloudlink", resourceName)

	b.SetAttr("name", Str(cloudlink.Name))
	if cloudlink.Type != "" {
		b.SetAttr("type", Str(cloudlink.Type))
	}

	blocks := []*HCLBlock{b}
	StripNulls(blocks)
	return SerializeBlocks(blocks)
}

// GenerateDatamineHCL generates HCL for a standalone datamine export
// with optional beautification using related resources.
func GenerateDatamineHCL(dm *discovery.Datamine, table *discovery.Table, relatedResources []discovery.RelatedResource) string {
	uuidMap := make(map[string]string)

	// Build UUID map from related resources
	if table != nil {
		tableName := SanitizeName(table.Name)
		uuidMap[table.ID] = "elementum_table." + tableName + ".id"
		for _, f := range table.Fields {
			if f.ID != "" && f.Name != "" {
				sanitizedFieldName := SanitizeName(f.Name)
				uuidMap[f.ID] = "data.elementum_field." + tableName + "_" + sanitizedFieldName + ".id"
			}
		}
	}
	for _, r := range relatedResources {
		sanitizedName := SanitizeName(r.Name)
		if r.ResourceType == "elementum_cloudlink" {
			uuidMap[r.ID] = "data.elementum_cloudlink." + sanitizedName + ".id"
		} else {
			uuidMap[r.ID] = r.ResourceType + "." + sanitizedName + ".id"
		}
	}

	gen := &AppHCLGenerator{
		uuidMap: uuidMap,
	}
	block := gen.generateDatamineIR(dm)
	if block == nil {
		return ""
	}

	blocks := []*HCLBlock{block}
	StripNulls(blocks)
	ResolveUUIDs(blocks, uuidMap)
	return SerializeBlocks(blocks)
}
