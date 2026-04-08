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
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/internal/client"
)

// BulkCreateInput holds the configuration for a bulk create operation
type BulkCreateInput struct {
	AspectID string
	Fields   []AspectFieldInfo
	Records  []map[string]interface{} // field name -> value
}

// BulkCreateResult holds the results of a bulk create operation
type BulkCreateResult struct {
	CreatedIDs []string
	Total      int
	Errors     []string
}

// BulkUpdateInput holds a single record update
type BulkUpdateInput struct {
	RecordID string
	Fields   map[string]interface{} // field ID -> value
}

// BulkUpdateResult holds the results of a bulk update operation
type BulkUpdateResult struct {
	UpdatedCount int
	Records      []BulkUpdatedRecord
}

// BulkUpdatedRecord is the result for a single updated record
type BulkUpdatedRecord struct {
	ID        string
	Handle    string
	Title     string
	UpdatedAt string
}

// BulkCreateRecords creates multiple records using the bulk API.
// Records are provided as maps of field name -> value. Field names are resolved to IDs internally.
func BulkCreateRecords(ctx context.Context, c *client.Client, input BulkCreateInput) (*BulkCreateResult, error) {
	if len(input.Records) == 0 {
		return &BulkCreateResult{}, nil
	}

	logger.Debug("bulk creating records", "aspectID", input.AspectID, "count", len(input.Records))

	fieldNameToID := buildFieldNameMap(input.Fields)

	// Build the field IDs list from the first record's keys (all records must have same fields)
	allFieldIDs := collectFieldIDs(input.Records, fieldNameToID)

	const batchSize = 100
	result := &BulkCreateResult{}

	for i := 0; i < len(input.Records); i += batchSize {
		end := i + batchSize
		if end > len(input.Records) {
			end = len(input.Records)
		}
		batch := input.Records[i:end]

		batchResult, err := executeBulkCreateBatch(ctx, c, input.AspectID, allFieldIDs, batch, fieldNameToID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("batch %d-%d: %v", i, end-1, err))
			continue
		}
		result.CreatedIDs = append(result.CreatedIDs, batchResult...)
	}

	result.Total = len(result.CreatedIDs)
	logger.Debug("bulk create completed", "created", result.Total, "errors", len(result.Errors))
	return result, nil
}

// BulkCreateWithFallback tries the bulk API first, then automatically falls back to
// parallel individual creates if the bulk API fails. This provides a seamless experience
// for users - they don't need to know about API differences.
func BulkCreateWithFallback(ctx context.Context, c *client.Client, input BulkCreateInput) (*BulkCreateResult, error) {
	if len(input.Records) == 0 {
		return &BulkCreateResult{}, nil
	}

	logger.Debug("attempting bulk create with fallback", "aspectID", input.AspectID, "count", len(input.Records))

	// Try bulk API first
	result, err := BulkCreateRecords(ctx, c, input)

	// If bulk succeeded with no errors, we're done
	if err == nil && len(result.Errors) == 0 {
		return result, nil
	}

	// Log the fallback
	if err != nil {
		logger.Info("bulk API failed, falling back to individual creates", "error", err)
	} else if len(result.Errors) > 0 {
		logger.Info("bulk API had errors, falling back to individual creates", "errorCount", len(result.Errors), "firstError", result.Errors[0])
	}

	// Fallback to parallel single creates
	return parallelSingleCreates(ctx, c, input, 5) // 5 parallel workers
}

// parallelSingleCreates creates records one at a time using parallel workers.
// This is used as a fallback when the bulk API fails.
func parallelSingleCreates(ctx context.Context, c *client.Client, input BulkCreateInput, workers int) (*BulkCreateResult, error) {
	logger.Debug("starting parallel single creates", "count", len(input.Records), "workers", workers)

	result := &BulkCreateResult{
		CreatedIDs: make([]string, 0, len(input.Records)),
	}

	// Build field name to ID map
	fieldNameToID := buildFieldNameMap(input.Fields)

	// Channel for work items
	type workItem struct {
		index  int
		record map[string]interface{}
	}
	workChan := make(chan workItem, len(input.Records))

	// Channel for results
	type workResult struct {
		index int
		id    string
		err   error
	}
	resultChan := make(chan workResult, len(input.Records))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for work := range workChan {
				// Convert field names to IDs in the record
				resolvedFields := make(map[string]interface{})
				for key, val := range work.record {
					id := resolveFieldKey(key, fieldNameToID)
					resolvedFields[id] = val
				}

				createResult, err := CreateRecord(ctx, c, RecordCreateInput{
					AspectID: input.AspectID,
					Fields:   resolvedFields,
				})

				if err != nil {
					resultChan <- workResult{index: work.index, err: err}
				} else {
					resultChan <- workResult{index: work.index, id: createResult.ID}
				}
			}
		}()
	}

	// Send work items
	for i, record := range input.Records {
		workChan <- workItem{index: i, record: record}
	}
	close(workChan)

	// Wait for workers to finish in a goroutine, then close results channel
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for res := range resultChan {
		if res.err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("record %d: %v", res.index+1, res.err))
		} else {
			result.CreatedIDs = append(result.CreatedIDs, res.id)
		}
	}

	result.Total = len(result.CreatedIDs)
	logger.Debug("parallel single creates completed", "created", result.Total, "errors", len(result.Errors))
	return result, nil
}

func buildFieldNameMap(fields []AspectFieldInfo) map[string]string {
	m := make(map[string]string, len(fields))
	for _, f := range fields {
		m[strings.ToLower(f.Name)] = f.ID
	}
	return m
}

func collectFieldIDs(records []map[string]interface{}, nameToID map[string]string) []string {
	seen := make(map[string]bool)
	var ids []string
	for _, rec := range records {
		for key := range rec {
			id := resolveFieldKey(key, nameToID)
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	return ids
}

func resolveFieldKey(key string, nameToID map[string]string) string {
	if uuidRegex.MatchString(key) {
		return key
	}
	if id, ok := nameToID[strings.ToLower(key)]; ok {
		return id
	}
	return key
}

func executeBulkCreateBatch(ctx context.Context, c *client.Client, aspectID string, fieldIDs []string, records []map[string]interface{}, nameToID map[string]string) ([]string, error) {
	values := make([]*json.RawMessage, 0, len(records))
	for _, rec := range records {
		row := make(map[string]interface{})
		for key, val := range rec {
			id := resolveFieldKey(key, nameToID)
			row[id] = val
		}
		raw, err := json.Marshal(row)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal record: %w", err)
		}
		rawMsg := json.RawMessage(raw)
		values = append(values, &rawMsg)
	}

	bulkInput := client.AspectRecordBulkCreateInput{
		Fields: fieldIDs,
		Values: values,
	}

	resp, err := client.BulkCreateRecords(ctx, c.Genqlient(), aspectID, bulkInput)
	if err != nil {
		return nil, fmt.Errorf("bulk create API error: %w", err)
	}

	return resp.AspectRecordBulkCreate, nil
}

// BulkUpdateRecords updates multiple records using the bulk API
func BulkUpdateRecords(ctx context.Context, c *client.Client, updates []BulkUpdateInput) (*BulkUpdateResult, error) {
	if len(updates) == 0 {
		return &BulkUpdateResult{}, nil
	}

	logger.Debug("bulk updating records", "count", len(updates))

	const batchSize = 50
	result := &BulkUpdateResult{}

	for i := 0; i < len(updates); i += batchSize {
		end := i + batchSize
		if end > len(updates) {
			end = len(updates)
		}
		batch := updates[i:end]

		inputs := make([]client.AspectRecordUpdateInput, 0, len(batch))
		for _, u := range batch {
			dataJSON, err := json.Marshal(u.Fields)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal update data for %s: %w", u.RecordID, err)
			}
			inputs = append(inputs, client.AspectRecordUpdateInput{
				RecordId: u.RecordID,
				Data:     json.RawMessage(dataJSON),
			})
		}

		resp, err := client.BulkUpdateRecords(ctx, c.Genqlient(), inputs)
		if err != nil {
			return nil, fmt.Errorf("bulk update API error: %w", err)
		}

		for _, rec := range resp.AspectRecordsUpdate {
			result.Records = append(result.Records, BulkUpdatedRecord{
				ID:        rec.GetId(),
				Handle:    rec.GetHandle(),
				Title:     safeStringFromPtr(rec.GetTitle()),
				UpdatedAt: safeStringFromPtr(rec.GetUpdatedAt()),
			})
		}
	}

	result.UpdatedCount = len(result.Records)
	logger.Debug("bulk update completed", "updated", result.UpdatedCount)
	return result, nil
}

// FileFormat represents a supported input file format
type FileFormat string

const (
	FormatCSV   FileFormat = "csv"
	FormatJSON  FileFormat = "json"
	FormatJSONL FileFormat = "jsonl"
)

// DetectFileFormat determines the file format from extension
func DetectFileFormat(path string) (FileFormat, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".csv":
		return FormatCSV, nil
	case ".tsv":
		return FormatCSV, nil // handled as CSV with tab delimiter
	case ".json":
		return FormatJSON, nil
	case ".jsonl", ".ndjson":
		return FormatJSONL, nil
	default:
		return "", fmt.Errorf("unsupported file format: %s (supported: .csv, .tsv, .json, .jsonl)", ext)
	}
}

// LoadRecordsFromFile reads records from a file and returns them as maps of field name -> value
func LoadRecordsFromFile(path string, format FileFormat) ([]map[string]interface{}, error) {
	switch format {
	case FormatCSV:
		return loadRecordsFromCSV(path)
	case FormatJSON:
		return loadRecordsFromJSON(path)
	case FormatJSONL:
		return loadRecordsFromJSONL(path)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

func loadRecordsFromCSV(path string) ([]map[string]interface{}, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = f.Close() }()

	reader := csv.NewReader(f)

	// Auto-detect tab-separated files
	if strings.HasSuffix(strings.ToLower(path), ".tsv") {
		reader.Comma = '\t'
	}

	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Trim BOM and whitespace from headers
	for i, h := range headers {
		headers[i] = strings.TrimSpace(strings.TrimLeft(h, "\xef\xbb\xbf"))
	}

	var records []map[string]interface{}
	lineNum := 1

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV row %d: %w", lineNum+1, err)
		}
		lineNum++

		record := make(map[string]interface{})
		for i, value := range row {
			if i < len(headers) && strings.TrimSpace(value) != "" {
				record[headers[i]] = value
			}
		}

		if len(record) > 0 {
			records = append(records, record)
		}
	}

	return records, nil
}

func loadRecordsFromJSON(path string) ([]map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Try parsing as array first
	var records []map[string]interface{}
	if err := json.Unmarshal(data, &records); err == nil {
		return records, nil
	}

	// Try as object with a "records" key
	var wrapper struct {
		Records []map[string]interface{} `json:"records"`
	}
	if err := json.Unmarshal(data, &wrapper); err == nil && wrapper.Records != nil {
		return wrapper.Records, nil
	}

	// Try as single object
	var single map[string]interface{}
	if err := json.Unmarshal(data, &single); err == nil {
		return []map[string]interface{}{single}, nil
	}

	return nil, fmt.Errorf("failed to parse JSON: expected array of records, object with 'records' key, or single record object")
}

func loadRecordsFromJSONL(path string) ([]map[string]interface{}, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = f.Close() }()

	var records []map[string]interface{}
	decoder := json.NewDecoder(f)
	lineNum := 0

	for {
		var record map[string]interface{}
		if err := decoder.Decode(&record); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to parse JSONL line %d: %w", lineNum+1, err)
		}
		lineNum++
		records = append(records, record)
	}

	return records, nil
}

// ResolveRecordFieldValues resolves field names to IDs and converts values to appropriate types
// for a batch of records. Returns records keyed by field ID.
func ResolveRecordFieldValues(ctx context.Context, c *client.Client, fields []AspectFieldInfo, records []map[string]interface{}) ([]map[string]interface{}, error) {
	resolved := make([]map[string]interface{}, 0, len(records))

	for i, rec := range records {
		resolvedRec := make(map[string]interface{})
		for key, val := range rec {
			fieldID, fieldInfo, err := ResolveFieldNameToID(fields, key)
			if err != nil {
				return nil, fmt.Errorf("record %d: %w", i+1, err)
			}

			var resolvedVal interface{}
			switch v := val.(type) {
			case string:
				if fieldInfo != nil {
					resolvedVal, err = ResolveFieldValue(fieldInfo, v)
					if err != nil {
						// For user/group fields that need lookup, try with the client
						if fieldInfo.Type == "AspectUserField" && strings.Contains(v, "@") {
							resolvedVal, err = LookupUserByEmail(ctx, c, v)
							if err != nil {
								return nil, fmt.Errorf("record %d, field %q: %w", i+1, key, err)
							}
						} else if fieldInfo.Type == "AspectGroupField" {
							resolvedVal, err = LookupGroupByName(ctx, c, v)
							if err != nil {
								return nil, fmt.Errorf("record %d, field %q: %w", i+1, key, err)
							}
						} else {
							return nil, fmt.Errorf("record %d, field %q: %w", i+1, key, err)
						}
					}
				} else {
					resolvedVal = v
				}
			default:
				resolvedVal = val
			}

			resolvedRec[fieldID] = resolvedVal
		}
		resolved = append(resolved, resolvedRec)
	}

	return resolved, nil
}

// ExportRecordsToFile writes records to a file in the specified format
func ExportRecordsToFile(records []Record, path string, format FileFormat) error {
	switch format {
	case FormatCSV:
		return exportRecordsToCSV(records, path)
	case FormatJSON:
		return exportRecordsToJSON(records, path)
	case FormatJSONL:
		return exportRecordsToJSONL(records, path)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func exportRecordsToCSV(records []Record, path string) error {
	if len(records) == 0 {
		return fmt.Errorf("no records to export")
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = f.Close() }()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// Collect all unique field names across records
	headers := collectAllFieldNames(records)

	// Write header
	fullHeaders := append([]string{"Handle", "Title", "Status", "Created At"}, headers...)
	if err := writer.Write(fullHeaders); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write rows
	for _, rec := range records {
		row := []string{rec.Handle, rec.Title, rec.Status, rec.CreatedAt}
		for _, h := range headers {
			row = append(row, ExtractDisplayValue(rec.Data[h]))
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}

	return nil
}

func exportRecordsToJSON(records []Record, path string) error {
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal records: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func exportRecordsToJSONL(records []Record, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = f.Close() }()

	encoder := json.NewEncoder(f)
	for _, rec := range records {
		if err := encoder.Encode(rec); err != nil {
			return fmt.Errorf("failed to write record: %w", err)
		}
	}
	return nil
}

func collectAllFieldNames(records []Record) []string {
	seen := make(map[string]bool)
	var names []string
	for _, rec := range records {
		for key := range rec.Data {
			if !seen[key] {
				seen[key] = true
				names = append(names, key)
			}
		}
	}
	return names
}
