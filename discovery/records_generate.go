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

package discovery

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

// GenerateRecords creates synthetic records based on field schema
// Deprecated: Use GenerateRecordsForAspect instead to properly handle Elements
func GenerateRecords(fields []AspectFieldInfo, count int, seed int64) []map[string]interface{} {
	return GenerateRecordsForAspect(fields, count, seed, AspectTypeApp)
}

// GenerateRecordsForAspect creates synthetic records with aspect type awareness
// For Elements, this ensures the HANDLE field (ID) is always generated with a unique value
func GenerateRecordsForAspect(fields []AspectFieldInfo, count int, seed int64, aspectType AspectType) []map[string]interface{} {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	rng := rand.New(rand.NewSource(seed))

	records := make([]map[string]interface{}, 0, count)
	for i := 0; i < count; i++ {
		rec := generateSingleRecordForAspect(fields, i, rng, aspectType)
		if len(rec) > 0 {
			records = append(records, rec)
		}
	}

	return records
}

func generateSingleRecordForAspect(fields []AspectFieldInfo, index int, rng *rand.Rand, aspectType AspectType) map[string]interface{} {
	rec := make(map[string]interface{})

	for _, field := range fields {
		// Check if this field has HANDLE semantic tag
		hasHandleTag := false
		for _, tag := range field.SemanticTags {
			if tag == "HANDLE" {
				hasHandleTag = true
				break
			}
		}

		// HANDLE fields: system:true auto-populates, system:false requires user input
		if hasHandleTag {
			if !field.System {
				// Non-system HANDLE field must be generated with a unique value
				rec[field.ID] = generateUniqueID(field, index, rng)
			}
			// Skip system HANDLE fields - they auto-populate
			continue
		}

		// Skip other system fields
		if field.System {
			continue
		}

		// Skip fields that can't be set directly
		switch field.Type {
		case "AspectHandleField", "AspectAttachmentField",
			"AspectCalculationField", "AspectReferenceField",
			"AspectCreatedAtField", "AspectUpdatedAtField",
			"AspectCreatedByField":
			continue
		}

		val := generateFieldValue(field, index, rng)
		if val != nil {
			rec[field.ID] = val
		}
	}

	return rec
}

// generateUniqueID generates a unique identifier for Element records
func generateUniqueID(field AspectFieldInfo, index int, rng *rand.Rand) string {
	name := strings.ToLower(field.Name)

	// Generate context-aware IDs based on field name
	switch {
	case strings.Contains(name, "sku"):
		return fmt.Sprintf("SKU-%06d", index+1)
	case strings.Contains(name, "code"):
		return fmt.Sprintf("CODE-%06d", index+1)
	case strings.Contains(name, "sys") || strings.Contains(name, "sys_id"):
		// Generate UUID-like sys_id
		return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
			rng.Uint32(), rng.Intn(65536), rng.Intn(65536), rng.Intn(65536),
			rng.Int63n(281474976710656))
	case strings.Contains(name, "skill"):
		return fmt.Sprintf("SKILL-%06d", index+1)
	default:
		// Default unique ID format
		return fmt.Sprintf("ID-%06d", index+1)
	}
}

func generateFieldValue(field AspectFieldInfo, index int, rng *rand.Rand) interface{} {
	// For non-required fields, skip ~30% of the time
	if !field.Required && rng.Float64() < 0.3 {
		return nil
	}

	switch field.Type {
	case "AspectTextField":
		return generateTextValue(field, index, rng)
	case "AspectHtmlField":
		return generateHTMLValue(field, index, rng)
	case "AspectNumberField":
		return generateNumberValue(rng)
	case "AspectDecimalField":
		return generateDecimalValue(rng)
	case "AspectBooleanField":
		return rng.Intn(2) == 0
	case "AspectDateField":
		return generateDateValue(rng)
	case "AspectDateTimeField":
		return generateDateTimeValue(rng)
	case "AspectPicklistField":
		return generatePicklistValue(field, rng)
	case "AspectMultiPicklistField":
		return generateMultiPicklistValue(field, rng)
	case "AspectUserField", "AspectGroupField":
		return nil // can't generate valid user/group IDs
	default:
		return nil
	}
}

var sampleTitles = []string{
	"Quarterly Review", "Bug Fix", "Feature Request", "Maintenance Task",
	"Customer Inquiry", "Design Proposal", "Sprint Planning", "Code Review",
	"Infrastructure Update", "Security Audit", "Performance Test", "Data Migration",
	"API Integration", "UI Enhancement", "Documentation Update", "Release Notes",
	"Incident Report", "Compliance Check", "Training Session", "Budget Review",
}

var sampleDescriptions = []string{
	"Requires immediate attention from the team.",
	"Follow up with stakeholders before proceeding.",
	"This has been on the backlog for a while.",
	"High priority item that affects multiple teams.",
	"Standard process, no special handling needed.",
	"Pending review from the technical lead.",
	"Part of the Q2 roadmap initiative.",
	"Needs additional input from the client.",
}

var sampleParagraphs = []string{
	"This item requires coordination across multiple departments to ensure successful delivery.",
	"The proposed changes will improve system reliability and reduce manual intervention.",
	"After careful analysis, we recommend proceeding with the phased approach outlined below.",
	"Key metrics should be monitored throughout the implementation to catch regressions early.",
}

func generateTextValue(field AspectFieldInfo, index int, rng *rand.Rand) string {
	name := strings.ToLower(field.Name)
	for _, tag := range field.SemanticTags {
		if strings.EqualFold(tag, "TITLE") {
			base := sampleTitles[rng.Intn(len(sampleTitles))]
			return fmt.Sprintf("%s #%d", base, index+1)
		}
	}
	switch {
	case strings.Contains(name, "title") || strings.Contains(name, "subject") || strings.Contains(name, "name"):
		base := sampleTitles[rng.Intn(len(sampleTitles))]
		return fmt.Sprintf("%s #%d", base, index+1)
	case strings.Contains(name, "description") || strings.Contains(name, "summary") || strings.Contains(name, "note"):
		return sampleDescriptions[rng.Intn(len(sampleDescriptions))]
	case strings.Contains(name, "email"):
		return fmt.Sprintf("user%d@example.com", rng.Intn(100))
	case strings.Contains(name, "phone"):
		return fmt.Sprintf("+1-%03d-%03d-%04d", rng.Intn(999), rng.Intn(999), rng.Intn(9999))
	case strings.Contains(name, "url") || strings.Contains(name, "link") || strings.Contains(name, "website"):
		return fmt.Sprintf("https://example.com/item/%d", index+1)
	default:
		return fmt.Sprintf("Sample %s %d", field.Name, index+1)
	}
}

func generateHTMLValue(field AspectFieldInfo, index int, rng *rand.Rand) string {
	para := sampleParagraphs[rng.Intn(len(sampleParagraphs))]
	return fmt.Sprintf("<p>%s</p><p>Reference: item-%d</p>", para, index+1)
}

func generateNumberValue(rng *rand.Rand) int64 {
	return int64(rng.Intn(1000))
}

func generateDecimalValue(rng *rand.Rand) float64 {
	v := rng.Float64() * 10000
	f, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", v), 64)
	return f
}

func generateDateValue(rng *rand.Rand) string {
	daysOffset := rng.Intn(365) - 30
	t := time.Now().AddDate(0, 0, daysOffset)
	return t.Format("2006-01-02")
}

func generateDateTimeValue(rng *rand.Rand) string {
	daysOffset := rng.Intn(365) - 30
	hours := rng.Intn(24)
	mins := rng.Intn(60)
	t := time.Now().AddDate(0, 0, daysOffset).
		Truncate(24 * time.Hour).
		Add(time.Duration(hours)*time.Hour + time.Duration(mins)*time.Minute)
	return t.Format(time.RFC3339)
}

func generatePicklistValue(field AspectFieldInfo, rng *rand.Rand) interface{} {
	if len(field.Options) == 0 {
		return nil
	}
	return field.Options[rng.Intn(len(field.Options))].ID
}

func generateMultiPicklistValue(field AspectFieldInfo, rng *rand.Rand) interface{} {
	if len(field.Options) == 0 {
		return nil
	}
	// Select 1-3 random options
	numSelections := rng.Intn(3) + 1
	if numSelections > len(field.Options) {
		numSelections = len(field.Options)
	}

	perm := rng.Perm(len(field.Options))
	selected := make([]string, numSelections)
	for i := 0; i < numSelections; i++ {
		selected[i] = field.Options[perm[i]].ID
	}
	return selected
}
