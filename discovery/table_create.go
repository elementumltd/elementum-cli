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
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/internal/client"
)

// TableCreateConfig represents the YAML configuration for creating a table
type TableCreateConfig struct {
	Name        string       `yaml:"name" json:"name"`
	Handle      string       `yaml:"handle" json:"handle"`
	Category    string       `yaml:"category" json:"category"`
	Description string       `yaml:"description,omitempty" json:"description,omitempty"`
	Source      string       `yaml:"source" json:"source"`
	Fields      []FieldGroup `yaml:"fields" json:"fields"`
	Joins       []JoinConfig `yaml:"joins,omitempty" json:"joins,omitempty"`
}

// FieldGroup represents fields from a single source
type FieldGroup struct {
	Source  string         `yaml:"source" json:"source"`
	Columns []ColumnConfig `yaml:"columns" json:"columns"`
}

// ColumnConfig represents a single column/field selection
type ColumnConfig struct {
	Name  string `yaml:"name" json:"name"`
	Alias string `yaml:"alias,omitempty" json:"alias,omitempty"`
}

// JoinConfig represents a join definition
type JoinConfig struct {
	Source string         `yaml:"source" json:"source"`
	Type   string         `yaml:"type" json:"type"` // INNER, LEFT, RIGHT, FULL
	On     []JoinOnConfig `yaml:"on" json:"on"`
}

// JoinOnConfig represents a single join condition
type JoinOnConfig struct {
	LeftSource string `yaml:"left_source,omitempty" json:"left_source,omitempty"`
	LeftField  string `yaml:"left_field" json:"left_field"`
	RightField string `yaml:"right_field" json:"right_field"`
}

// CloudLinkPath represents a parsed CloudLink reference
type CloudLinkPath struct {
	CloudLinkName string
	Database      string
	Schema        string
	Table         string
}

// ResolvedSource represents a source that has been resolved to IDs
type ResolvedSource struct {
	Name            string // Original name from config
	ID              string // Resolved ID
	Type            string // "app", "element", "table", "cloudlink"
	CloudLinkPath   *CloudLinkPath
	Fields          []ResolvedField
	NeedsGeneration bool   // True if this CloudLink table needs to be generated
	TerraformName   string // Sanitized name for Terraform resource
}

// ResolvedField represents a field that has been resolved
type ResolvedField struct {
	ID           string
	Name         string
	Alias        string
	CloudType    string // For CloudLink fields
	DatabaseType string // Raw database type
	SourceName   string // Which source this field belongs to
	TerraformRef string // Terraform reference like "elementum_table.foo.id"
}

// ResolvedConfig represents a fully resolved table configuration
type ResolvedConfig struct {
	Config         *TableCreateConfig
	CategoryID     string
	Sources        map[string]*ResolvedSource // keyed by original source name
	PrimarySource  *ResolvedSource
	JoinSources    []*ResolvedSource
	FieldsToCreate []ResolvedField
	JoinsToCreate  []ResolvedJoin
}

// ResolvedJoin represents a resolved join
type ResolvedJoin struct {
	SourceID     string
	SourceName   string
	Type         string
	LeftFieldID  string
	RightFieldID string
	TerraformRef string
}

// ParseCloudLinkPath parses a dot-notation CloudLink path
// Format: "CloudLink Name.DATABASE.SCHEMA.TABLE"
// Returns nil if not a CloudLink path (no dots or wrong format)
func ParseCloudLinkPath(path string) *CloudLinkPath {
	// Split by dots, but CloudLink name can contain dots so we need to be smart
	// We expect at least 4 parts: cloudlink.db.schema.table
	// The last 3 are always DB.SCHEMA.TABLE (uppercase typically)
	// Everything before is the CloudLink name

	parts := strings.Split(path, ".")
	if len(parts) < 4 {
		return nil
	}

	// Last 3 parts are database, schema, table
	table := parts[len(parts)-1]
	schema := parts[len(parts)-2]
	database := parts[len(parts)-3]
	cloudLinkName := strings.Join(parts[:len(parts)-3], ".")

	// Validate - database names are typically uppercase
	if cloudLinkName == "" {
		return nil
	}

	return &CloudLinkPath{
		CloudLinkName: cloudLinkName,
		Database:      database,
		Schema:        schema,
		Table:         table,
	}
}

// IsCloudLinkPath checks if a source string is a CloudLink path
func IsCloudLinkPath(source string) bool {
	return ParseCloudLinkPath(source) != nil
}

// SanitizeTerraformName converts a name to a valid Terraform resource name
func SanitizeTerraformName(name string) string {
	// Replace spaces and special chars with underscores
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	result := re.ReplaceAllString(name, "_")
	// Remove consecutive underscores
	re2 := regexp.MustCompile(`_+`)
	result = re2.ReplaceAllString(result, "_")
	// Trim underscores
	result = strings.Trim(result, "_")
	// Lowercase
	result = strings.ToLower(result)
	// Ensure it doesn't start with a number
	if len(result) > 0 && result[0] >= '0' && result[0] <= '9' {
		result = "_" + result
	}
	if result == "" {
		result = "table"
	}
	return result
}

// TableResolver handles resolution of table configuration to IDs
type TableResolver struct {
	client     *client.Client
	ctx        context.Context
	tables     []Table         // Cached list of all tables
	objects    []ObjectSummary // Cached list of all objects (apps/elements)
	categories []Category      // Cached list of categories
	cloudlinks []CloudLink     // Cached list of cloudlinks
}

// NewTableResolver creates a new resolver
func NewTableResolver(ctx context.Context, c *client.Client) *TableResolver {
	return &TableResolver{
		client: c,
		ctx:    ctx,
	}
}

// loadCaches loads all required data for resolution
func (r *TableResolver) loadCaches() error {
	var err error

	// Load tables
	r.tables, err = ListTables(r.ctx, r.client)
	if err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}
	logger.Debug("loaded tables", "count", len(r.tables))

	// Load objects (apps/elements)
	r.objects, err = ListObjects(r.ctx, r.client)
	if err != nil {
		return fmt.Errorf("failed to list objects: %w", err)
	}
	logger.Debug("loaded objects", "count", len(r.objects))

	// Load categories
	r.categories, err = ListCategories(r.ctx, r.client)
	if err != nil {
		return fmt.Errorf("failed to list categories: %w", err)
	}
	logger.Debug("loaded categories", "count", len(r.categories))

	// Load cloudlinks
	r.cloudlinks, err = ListCloudLinks(r.ctx, r.client)
	if err != nil {
		return fmt.Errorf("failed to list cloudlinks: %w", err)
	}
	logger.Debug("loaded cloudlinks", "count", len(r.cloudlinks))

	return nil
}

// ResolveConfig resolves all names in the config to IDs
func (r *TableResolver) ResolveConfig(config *TableCreateConfig) (*ResolvedConfig, error) {
	if err := r.loadCaches(); err != nil {
		return nil, err
	}

	resolved := &ResolvedConfig{
		Config:  config,
		Sources: make(map[string]*ResolvedSource),
	}

	// Resolve category
	categoryID, err := r.resolveCategory(config.Category)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve category %q: %w", config.Category, err)
	}
	resolved.CategoryID = categoryID

	// Collect all unique sources from fields and joins
	sourceNames := make(map[string]bool)
	sourceNames[config.Source] = true
	for _, fg := range config.Fields {
		sourceNames[fg.Source] = true
	}
	for _, j := range config.Joins {
		sourceNames[j.Source] = true
		for _, on := range j.On {
			if on.LeftSource != "" {
				sourceNames[on.LeftSource] = true
			}
		}
	}

	// Resolve each source
	for sourceName := range sourceNames {
		source, err := r.resolveSource(sourceName)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve source %q: %w", sourceName, err)
		}
		resolved.Sources[sourceName] = source
	}

	// Set primary source
	resolved.PrimarySource = resolved.Sources[config.Source]

	// Build join sources list
	for _, j := range config.Joins {
		resolved.JoinSources = append(resolved.JoinSources, resolved.Sources[j.Source])
	}

	// Resolve fields
	for _, fg := range config.Fields {
		source := resolved.Sources[fg.Source]
		for _, col := range fg.Columns {
			field, err := r.resolveField(source, col)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve field %q from %q: %w", col.Name, fg.Source, err)
			}
			resolved.FieldsToCreate = append(resolved.FieldsToCreate, *field)
		}
	}

	// Resolve joins
	for _, j := range config.Joins {
		joinSource := resolved.Sources[j.Source]
		for _, on := range j.On {
			// Determine left source (default to primary if not specified)
			leftSourceName := on.LeftSource
			if leftSourceName == "" {
				leftSourceName = config.Source
			}
			leftSource := resolved.Sources[leftSourceName]

			leftField, err := r.findFieldByName(leftSource, on.LeftField)
			if err != nil {
				return nil, fmt.Errorf("failed to find left join field %q in %q: %w", on.LeftField, leftSourceName, err)
			}

			rightField, err := r.findFieldByName(joinSource, on.RightField)
			if err != nil {
				return nil, fmt.Errorf("failed to find right join field %q in %q: %w", on.RightField, j.Source, err)
			}

			resolved.JoinsToCreate = append(resolved.JoinsToCreate, ResolvedJoin{
				SourceID:     joinSource.ID,
				SourceName:   j.Source,
				Type:         strings.ToUpper(j.Type),
				LeftFieldID:  leftField.ID,
				RightFieldID: rightField.ID,
			})
		}
	}

	return resolved, nil
}

func (r *TableResolver) resolveCategory(name string) (string, error) {
	nameLower := strings.ToLower(name)
	for _, cat := range r.categories {
		if strings.ToLower(cat.Name) == nameLower {
			return cat.ID, nil
		}
	}
	return "", fmt.Errorf("category not found: %s", name)
}

func (r *TableResolver) resolveSource(name string) (*ResolvedSource, error) {
	// Check if it's a CloudLink path
	if clPath := ParseCloudLinkPath(name); clPath != nil {
		return r.resolveCloudLinkSource(name, clPath)
	}

	// Try to find as existing table
	for _, t := range r.tables {
		if strings.EqualFold(t.Name, name) || strings.EqualFold(t.Handle, name) {
			table, err := GetTable(r.ctx, r.client, t.ID)
			if err != nil {
				return nil, err
			}
			return r.tableToResolvedSource(name, table)
		}
	}

	// Try to find as app/element
	for _, obj := range r.objects {
		if strings.EqualFold(obj.Name, name) || strings.EqualFold(obj.Namespace, name) {
			return r.objectToResolvedSource(name, &obj)
		}
	}

	return nil, fmt.Errorf("source not found: %s (not a table, app, element, or valid CloudLink path)", name)
}

func (r *TableResolver) resolveCloudLinkSource(name string, clPath *CloudLinkPath) (*ResolvedSource, error) {
	// Find the CloudLink by name
	var cloudlink *CloudLink
	for _, cl := range r.cloudlinks {
		if strings.EqualFold(cl.Name, clPath.CloudLinkName) {
			cloudlink = &cl
			break
		}
	}
	if cloudlink == nil {
		return nil, fmt.Errorf("cloudlink not found: %s", clPath.CloudLinkName)
	}

	// Check if a table already exists for this CloudLink path
	for _, t := range r.tables {
		table, err := GetTable(r.ctx, r.client, t.ID)
		if err != nil {
			continue
		}
		if table.CloudLinkID == cloudlink.ID &&
			strings.EqualFold(table.SnowflakeDatabaseName, clPath.Database) &&
			strings.EqualFold(table.SnowflakeSchemaName, clPath.Schema) &&
			strings.EqualFold(table.SnowflakeTableName, clPath.Table) {
			logger.Debug("found existing table for CloudLink path",
				"path", name,
				"table_id", table.ID,
				"table_name", table.Name)
			return r.tableToResolvedSource(name, table)
		}
	}

	// No existing table - need to generate one
	// Fetch the schema from CloudLink
	schema, err := GetSnowflakeTableSchema(r.ctx, r.client, cloudlink.ID, clPath.Database, clPath.Schema, clPath.Table)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema for %s: %w", name, err)
	}

	source := &ResolvedSource{
		Name:            name,
		ID:              cloudlink.ID, // Will use CloudLink ID as source
		Type:            "cloudlink",
		CloudLinkPath:   clPath,
		NeedsGeneration: true,
		TerraformName:   SanitizeTerraformName(clPath.Table),
	}

	// Add fields from schema
	for _, col := range schema.Columns {
		source.Fields = append(source.Fields, ResolvedField{
			Name:         col.Name,
			CloudType:    col.ToElementumCloudType(),
			DatabaseType: col.DatabaseType,
			SourceName:   name,
		})
	}

	logger.Debug("need to generate table for CloudLink path",
		"path", name,
		"cloudlink_id", cloudlink.ID,
		"columns", len(source.Fields))

	return source, nil
}

func (r *TableResolver) tableToResolvedSource(name string, table *Table) (*ResolvedSource, error) {
	source := &ResolvedSource{
		Name:          name,
		ID:            table.ID,
		Type:          "table",
		TerraformName: SanitizeTerraformName(table.Name),
	}

	for _, f := range table.Fields {
		source.Fields = append(source.Fields, ResolvedField{
			ID:         f.ID,
			Name:       f.Name,
			SourceName: name,
		})
	}

	return source, nil
}

func (r *TableResolver) objectToResolvedSource(name string, obj *ObjectSummary) (*ResolvedSource, error) {
	source := &ResolvedSource{
		Name:          name,
		ID:            obj.ID,
		Type:          obj.Type, // "app" or "element"
		TerraformName: SanitizeTerraformName(obj.Name),
	}

	// Fetch fields for the object
	fields, err := GetAspectFields(r.ctx, r.client, obj.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get fields for %s: %w", name, err)
	}

	for _, f := range fields {
		source.Fields = append(source.Fields, ResolvedField{
			ID:         f.ID,
			Name:       f.Name,
			SourceName: name,
		})
	}

	return source, nil
}

func (r *TableResolver) resolveField(source *ResolvedSource, col ColumnConfig) (*ResolvedField, error) {
	for _, f := range source.Fields {
		if strings.EqualFold(f.Name, col.Name) {
			alias := col.Alias
			if alias == "" {
				alias = f.Name
			}
			return &ResolvedField{
				ID:           f.ID,
				Name:         f.Name,
				Alias:        alias,
				CloudType:    f.CloudType,
				DatabaseType: f.DatabaseType,
				SourceName:   source.Name,
			}, nil
		}
	}
	return nil, fmt.Errorf("field %q not found in source %q", col.Name, source.Name)
}

func (r *TableResolver) findFieldByName(source *ResolvedSource, fieldName string) (*ResolvedField, error) {
	for _, f := range source.Fields {
		if strings.EqualFold(f.Name, fieldName) {
			return &f, nil
		}
	}
	return nil, fmt.Errorf("field %q not found", fieldName)
}

// GetCategories returns cached categories (for interactive mode)
func (r *TableResolver) GetCategories() ([]Category, error) {
	if r.categories == nil {
		categories, err := ListCategories(r.ctx, r.client)
		if err != nil {
			return nil, err
		}
		r.categories = categories
	}
	return r.categories, nil
}

// GetSources returns all available sources (for interactive mode)
func (r *TableResolver) GetSources() ([]SourceOption, error) {
	if err := r.loadCaches(); err != nil {
		return nil, err
	}

	var sources []SourceOption

	for _, obj := range r.objects {
		sources = append(sources, SourceOption{
			Name: obj.Name,
			ID:   obj.ID,
			Type: obj.Type,
		})
	}

	for _, t := range r.tables {
		sources = append(sources, SourceOption{
			Name: t.Name,
			ID:   t.ID,
			Type: "table",
		})
	}

	return sources, nil
}

// SourceOption represents a source choice for interactive mode
type SourceOption struct {
	Name string
	ID   string
	Type string // "app", "element", "table"
}

// GenerateTerraformHCL generates Terraform HCL from a resolved config
func GenerateTerraformHCL(resolved *ResolvedConfig) string {
	var sb strings.Builder

	// Track which CloudLink data sources we need
	cloudlinkDataSources := make(map[string]string) // cloudlink name -> terraform name

	// Generate base tables for CloudLink sources that need generation
	for _, source := range resolved.Sources {
		if source.NeedsGeneration && source.CloudLinkPath != nil {
			clTfName := SanitizeTerraformName(source.CloudLinkPath.CloudLinkName)
			cloudlinkDataSources[source.CloudLinkPath.CloudLinkName] = clTfName
		}
	}

	// Generate CloudLink data sources
	for clName, tfName := range cloudlinkDataSources {
		fmt.Fprintf(&sb, `data "elementum_cloudlink" "%s" {
  name = %q
}

`, tfName, clName)
	}

	// Generate base tables for CloudLink sources
	for _, source := range resolved.Sources {
		if source.NeedsGeneration && source.CloudLinkPath != nil {
			clTfName := cloudlinkDataSources[source.CloudLinkPath.CloudLinkName]
			sb.WriteString(generateBaseTableHCL(source, resolved.CategoryID, clTfName))
			sb.WriteString("\n")
		}
	}

	// Generate the main join table
	sb.WriteString(generateJoinTableHCL(resolved, cloudlinkDataSources))

	return sb.String()
}

func generateBaseTableHCL(source *ResolvedSource, categoryID string, cloudlinkTfName string) string {
	var sb strings.Builder

	tableName := source.CloudLinkPath.Table
	tfName := source.TerraformName

	fmt.Fprintf(&sb, `resource "elementum_table" "%s" {
  category_id = %q
  source_id   = data.elementum_cloudlink.%s.id
  name        = %q
  handle      = %q

  cloud_mapping_cloudlink_id       = data.elementum_cloudlink.%s.id
  cloud_mapping_snowflake_database = %q
  cloud_mapping_snowflake_schema   = %q
  cloud_mapping_snowflake_table    = %q

  fields = [
`, tfName, categoryID, cloudlinkTfName, tableName, strings.ToLower(tableName),
		cloudlinkTfName, source.CloudLinkPath.Database, source.CloudLinkPath.Schema, source.CloudLinkPath.Table)

	for i, f := range source.Fields {
		comma := ","
		if i == len(source.Fields)-1 {
			comma = ""
		}
		fmt.Fprintf(&sb, `    {
      name              = %q
      cloud_column_name = %q
      cloud_type        = %q
    }%s
`, f.Name, f.Name, f.CloudType, comma)
	}

	sb.WriteString(`  ]
}
`)
	return sb.String()
}

func generateJoinTableHCL(resolved *ResolvedConfig, cloudlinkDataSources map[string]string) string {
	var sb strings.Builder

	config := resolved.Config
	tfName := SanitizeTerraformName(config.Name)

	// Determine source_id reference
	sourceRef := getSourceRef(resolved.PrimarySource, cloudlinkDataSources)

	fmt.Fprintf(&sb, `resource "elementum_table" "%s" {
  category_id = %q
  source_id   = %s
  name        = %q
  handle      = %q
`, tfName, resolved.CategoryID, sourceRef, config.Name, config.Handle)

	if config.Description != "" {
		fmt.Fprintf(&sb, `  description = %q
`, config.Description)
	}

	// Generate fields
	sb.WriteString("\n  fields = [\n")
	for i, f := range resolved.FieldsToCreate {
		comma := ","
		if i == len(resolved.FieldsToCreate)-1 {
			comma = ""
		}

		source := resolved.Sources[f.SourceName]
		fieldRef := getFieldRef(source, f, cloudlinkDataSources)

		fmt.Fprintf(&sb, `    {
      name               = %q
      reference_field_id = %s
    }%s
`, f.Alias, fieldRef, comma)
	}
	sb.WriteString("  ]\n")

	// Generate joins if any
	if len(resolved.JoinsToCreate) > 0 {
		sb.WriteString("\n  joins = [\n")
		for i, j := range resolved.JoinsToCreate {
			comma := ","
			if i == len(resolved.JoinsToCreate)-1 {
				comma = ""
			}

			joinSource := resolved.Sources[j.SourceName]
			joinSourceRef := getSourceRef(joinSource, cloudlinkDataSources)

			// Find left and right field refs
			leftFieldRef := findFieldRefByID(resolved, j.LeftFieldID, cloudlinkDataSources)
			rightFieldRef := findFieldRefByID(resolved, j.RightFieldID, cloudlinkDataSources)

			fmt.Fprintf(&sb, `    {
      type      = %q
      source_id = %s
      join_on = [
        {
          left_field_id  = %s
          right_field_id = %s
        }
      ]
    }%s
`, j.Type, joinSourceRef, leftFieldRef, rightFieldRef, comma)
		}
		sb.WriteString("  ]\n")
	}

	sb.WriteString("}\n")
	return sb.String()
}

func getSourceRef(source *ResolvedSource, cloudlinkDataSources map[string]string) string {
	if source.NeedsGeneration {
		return fmt.Sprintf("elementum_table.%s.id", source.TerraformName)
	}
	// For existing sources, use a data source lookup or direct ID
	// For simplicity, using direct ID here - could be enhanced to use data sources
	return fmt.Sprintf("%q", source.ID)
}

func getFieldRef(source *ResolvedSource, field ResolvedField, cloudlinkDataSources map[string]string) string {
	if source.NeedsGeneration {
		// Find field index in source
		for i, f := range source.Fields {
			if f.Name == field.Name {
				return fmt.Sprintf("elementum_table.%s.fields[%d].id", source.TerraformName, i)
			}
		}
	}
	// Direct ID reference
	return fmt.Sprintf("%q", field.ID)
}

func findFieldRefByID(resolved *ResolvedConfig, fieldID string, cloudlinkDataSources map[string]string) string {
	for _, source := range resolved.Sources {
		for i, f := range source.Fields {
			if f.ID == fieldID {
				if source.NeedsGeneration {
					return fmt.Sprintf("elementum_table.%s.fields[%d].id", source.TerraformName, i)
				}
				return fmt.Sprintf("%q", fieldID)
			}
		}
	}
	return fmt.Sprintf("%q", fieldID)
}

// GenerateCurlCommand generates a curl command for direct API creation
func GenerateCurlCommand(resolved *ResolvedConfig, baseURL, token string) string {
	input := buildTableCreateInput(resolved)

	inputJSON, _ := json.MarshalIndent(input, "", "  ")

	query := `mutation CreateMultiJoinTable($input: TableCreateV2Input!) {
  tableCreateV2(input: $input) {
    id
    name
    handle
    joins { id type source { id name } }
    fields { id name fieldType }
  }
}`

	payload := map[string]interface{}{
		"query": query,
		"variables": map[string]interface{}{
			"input": input,
		},
	}

	payloadJSON, _ := json.Marshal(payload)

	return fmt.Sprintf(`curl -X POST '%s/graphql' \
  -H 'Authorization: Bearer %s' \
  -H 'Content-Type: application/json' \
  -d '%s'

# Input details:
# %s`, baseURL, token, string(payloadJSON), string(inputJSON))
}

func buildTableCreateInput(resolved *ResolvedConfig) map[string]interface{} {
	config := resolved.Config

	input := map[string]interface{}{
		"name":       config.Name,
		"handle":     config.Handle,
		"categoryId": resolved.CategoryID,
		"source":     resolved.PrimarySource.ID,
	}

	// Build fields
	var fields []map[string]interface{}
	for _, f := range resolved.FieldsToCreate {
		field := map[string]interface{}{
			"reference": map[string]interface{}{
				"name":  f.Alias,
				"field": f.ID,
			},
		}
		fields = append(fields, field)
	}
	input["fields"] = fields

	// Build joins
	if len(resolved.JoinsToCreate) > 0 {
		var joins []map[string]interface{}
		for _, j := range resolved.JoinsToCreate {
			join := map[string]interface{}{
				"type":   j.Type,
				"source": j.SourceID,
				"joinOn": []map[string]interface{}{
					{
						"leftFieldId":  j.LeftFieldID,
						"rightFieldId": j.RightFieldID,
					},
				},
			}
			joins = append(joins, join)
		}
		input["joins"] = joins
	}

	return input
}
