// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package discovery

import (
	"fmt"
	"sort"
	"sync"
)

// idGetter is implemented by types that have an ID for sorting
type idGetter interface {
	GetID() string
}

// sortByID sorts a slice of items by their ID for deterministic output
func sortByID[T idGetter](items []T) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetID() < items[j].GetID()
	})
}

// sortPtrByID sorts a slice of pointer items by their ID
func sortPtrByID[T idGetter](items []T) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetID() < items[j].GetID()
	})
}

// Beautifiable is implemented by types that can contribute UUID mappings
type Beautifiable interface {
	GetUUIDMappings(resourceName string) map[string]string
}

// App represents an discovered Elementum app with all its resources
type App struct {
	ID              string
	Name            string
	Namespace       string
	CategoryID      string // Category ID for the app
	CategoryName    string // Category name for data source generation
	CloudLinkID     string // CloudLink ID (optional, may be empty)
	CloudLinkName   string // CloudLink name for data source generation
	CommentsEnabled bool   // True if comments are enabled on this app
	LockStages      bool   // True if the stage field is locked during record creation
	Fields          []Field
	Layouts         []Layout
	Flows           []Flow
	Automations     []Automation
	Agents          []Agent
	Approvals       []ApprovalProcess
	Widgets         []Widget
	Views           []View
	AIFileReaders   []AIFileReader
	Relationships   []Relationship
	RelatedObjects  []RelatedObject // Objects discovered through relationships
	AccessPolicies  []AccessPolicy  // Access policies for row-level security
	Roles           []Role          // Roles for this app (managed + custom)

	// Referenced resources discovered from automation dependencies
	ReferencedDatamines []*Datamine
	ReferencedTables    []*Table

	// Discovered resources from unified recursive discovery (--recursive flag)
	DiscoveredApps            []*App
	DiscoveredElements        []*Element
	DiscoveredTasks           []*AspectTask // AspectTask - standalone tasks (distinct from automation tasks)
	DiscoveredTables          []*Table
	DiscoveredDatamines       []*Datamine
	DiscoveredCloudLinks      []*CloudLink
	DiscoveredStoredFunctions []*StoredFunction

	// Discovered AI provider connectors from automation tasks
	DiscoveredAiProviderConnectors []AiProviderConnector

	// Phone services on this app
	PhoneServices []PhoneService

	// AI Search Tables on this app
	AISearchTables []AISearchTable

	// Managed view order (singleton per aspect, nil if no views)
	ManagedViewOrder *ManagedViewOrder

	// Dashboards on this app
	Dashboards []Dashboard

	// Charts in this app's dashboards
	Charts []Chart
}

// Field represents a custom field in an app
type Field struct {
	ID           string
	Name         string
	Type         string // text, longtext, number, dropdown, user, calculated, etc.
	Required     bool
	System       bool          // True for system-managed fields
	SemanticTags []string      // System tags like "title", "stage", "status"
	Options      []FieldOption // For dropdown/multiselect fields
	Calculation  string        // For calculated fields - the formula text
	NeedsValues  bool          // Internal: marks fields that need values fetched in parallel mode
}

// FieldOption represents an option in a dropdown/multiselect field
type FieldOption struct {
	ID    string
	Label string
	Color string
	Tags  []string // Tags like "CLOSED" for status field options
}

// Layout represents a layout configuration
type Layout struct {
	ID            string
	Name          string
	Element       string         // Element ID if layout is for an element
	DisplayBlocks []DisplayBlock // Display blocks in this layout/stage
	IsInitiate    bool           // True if this is the platform-managed Initiate stage layout
}

// DisplayBlock represents a display block in a layout/stage
type DisplayBlock struct {
	ID       string
	Type     string // group, field, widget, activity_log, approvals, attachments, etc.
	Name     string // For group blocks, otherwise empty
	StageID  string // The stage this block belongs to
	AspectID string // The app/element this block belongs to
}

// Flow represents a flow diagram
type Flow struct {
	ID   string
	Name string
}

// Automation represents an automation with its workflow
type Automation struct {
	ID           string
	Name         string
	Status       string // ACTIVE, INACTIVE, UNPUBLISHED
	WorkflowID   string // Current workflow ID (published or draft)
	HasDraft     bool
	HasPublished bool
	Triggers     []Trigger
	Tasks        []Task
	Outputs      []WorkflowOutput // Outputs exposed by on-demand workflows
}

// WorkflowOutput represents an output property of a workflow
type WorkflowOutput struct {
	Name  string                 // Human-readable name
	Value map[string]interface{} // Value reference JSON (taskReference, etc.)
}

// Trigger represents a workflow trigger
type Trigger struct {
	ID         string
	Type       string // record_created, record_updated, webhook, datamine, etc.
	Name       string
	DatamineID string // For datamine triggers - the referenced datamine ID
	// FieldRefs maps field UUIDs to human-readable field names for beautification
	// Key: "record.{field_uuid}" or "{field_uuid}", Value: "Field Name"
	FieldRefs map[string]string
	// RawData holds the full trigger data from GraphQL for type-specific field extraction
	RawData map[string]interface{}
}

// Task represents a workflow task
type Task struct {
	ID                      string
	Type                    string // message, update_field, api, etc.
	Name                    string
	WorkflowID              string   // The workflow this task belongs to
	ParentID                string   // The previous task or trigger ID
	ObjectID                string   // Target app/element for cross-app tasks (from "aspect" field)
	RelatedObjectID         string   // For find_related_records task (from "relatedAspect" field)
	DocumentModelID         string   // For ai_file_read and bulk_excel tasks (from "documentModel" field)
	DynamicCategoryAspectID string   // For ai_classify task - the aspect from dynamic_category_source
	FieldIDs                []string // Fields referenced by the task
	// FieldRefs maps property paths to human-readable names for beautification
	// Key: property path (e.g., "Records[0].ID"), Value: "First found record id"
	FieldRefs map[string]string
	// RawData holds the full task data from GraphQL for type-specific field extraction
	RawData map[string]interface{}

	// AI Provider Connector info for AI tasks (ai_file_read, ai_classify, ai_summarize, etc.)
	AiProviderConnectorID        string // UUID of the AI provider connector
	AiProviderConnectorModelName string // Model name (e.g., "GPT-4o", "Claude Sonnet 4")
	AiProviderConnectorProvider  string // Provider name (e.g., "OpenAI", "Anthropic")

	// Stored Function info for procedure tasks
	StoredFunctionID   string // UUID of the stored function (procedure or UDF)
	StoredFunctionName string // Display name of the stored function
	CloudLinkID        string // UUID of the CloudLink containing the stored function

	// BrokenFields tracks GraphQL fields that returned errors (partial data).
	// Key: GraphQL field name (e.g., "agent", "aspect"), Value: error message.
	// Tasks with broken required fields will get TODO comments in exported HCL.
	BrokenFields map[string]string
}

// Agent represents an AI agent
type Agent struct {
	ID          string
	Name        string
	Description string
	Type        string // AgentElementum, AgentSnowflake, AgentBedrock, AgentBrowserUse

	// Configuration
	Instructions string
	FirstMessage string

	// AI Provider Connector (for data source generation)
	AiProviderConnectorID        string
	AiProviderConnectorModelName string
	AiProviderConnectorProvider  string

	// Type-specific configs
	SnowflakeDatabase        string // For AgentSnowflake
	SnowflakeSchema          string // For AgentSnowflake
	SnowflakeName            string // For AgentSnowflake
	BedrockAgentArn          string // For AgentBedrock
	BrowserAgentInstructions string // For AgentBrowserUse

	Tools           []AgentTool
	StartingActions []StartingAction
}

// StartingAction represents a predefined prompt users can select when interacting with an agent
type StartingAction struct {
	ID        string
	Name      string
	Prompt    string
	Order     int
	Enabled   bool
	Icon      string
	Variables []StartingActionVariable
}

// StartingActionVariable represents a variable input in a starting action
type StartingActionVariable struct {
	Key         string
	Label       string
	Type        string // TEXT, NUMBER, SELECT
	Required    bool
	Placeholder string
	Options     []StartingActionSelectOption
	Validation  *StartingActionValidation
}

// StartingActionSelectOption represents an option for SELECT type variables
type StartingActionSelectOption struct {
	Label string
	Value string
}

// StartingActionValidation represents validation rules for a variable
type StartingActionValidation struct {
	MinLength    *int
	MaxLength    *int
	Pattern      string
	ErrorMessage string
}

// AgentTool represents a tool available to an agent
type AgentTool struct {
	ID           string
	Name         string
	Description  string
	Type         string // AgentCreateRecordTool, AgentSearchAspectTool, etc.
	StartMessage string

	// Target references for recursive discovery
	AspectID              string // Target app/element for CreateRecord, SearchAspect, UpdateRecord, SearchTable tools
	AspectName            string // For beautification - data source generation
	RelatedAspectID       string // For RelateRecordTool - the second aspect
	RelatedAspectName     string
	SearchTableID         string // For SearchTableTool - the AI search table ID (NOT a global Table)
	SearchTableAspectID   string // For SearchTableTool - the aspect (app/element) the AI search table belongs to
	SearchTableAspectType string // "App" or "Element" - used for discovery queue
	AutomationID          string // For ExecuteWorkflowTool
	AutomationName        string
	TargetAgentID         string // For RunAgentTool
	TargetAgentName       string

	// Tool-specific configuration (stored for HCL generation)
	RawConfig map[string]interface{}
}

// TerraformResourceType returns the Terraform resource type for an AgentTool
func (at *AgentTool) TerraformResourceType() string {
	switch at.Type {
	case "AgentCreateRecordTool":
		return "elementum_agent_create_record_tool"
	case "AgentSearchAspectTool":
		return "elementum_agent_search_records_tool"
	case "AgentUpdateRecordTool":
		return "elementum_agent_update_record_tool"
	case "AgentSearchTableTool":
		return "elementum_agent_ai_search_tool"
	case "AgentRelateRecordTool":
		return "elementum_agent_relate_record_tool"
	case "AgentExecuteWorkflowTool":
		return "elementum_agent_run_automation_tool"
	case "AgentRunAgentTool":
		return "elementum_agent_run_agent_tool"
	case "AgentSelectBotRouteTool":
		return "elementum_agent_select_bot_route_tool"
	case "AgentMcpTool":
		return "elementum_agent_mcp_tool"
	default:
		return "elementum_agent_create_record_tool" // fallback
	}
}

// ApprovalProcess represents an approval process
type ApprovalProcess struct {
	ID   string
	Name string
}

// Widget represents a widget
type Widget struct {
	ID   string
	Name string
	Type string // __typename: DisplayWidgetRelatedAspect, DisplayWidgetRelatedLinkAction, DisplayWidgetRelatedCreateAction

	// Type-specific fields
	AspectID   string   // Target aspect ID (required for all types)
	Columns    []string // For RelatedAspect, RelatedLinkAction
	Rows       int      // For RelatedAspect, RelatedLinkAction
	ButtonType string   // For RelatedLinkAction, RelatedCreateAction (INLINE, PRIMARY, SECONDARY)
	Color      string   // For RelatedLinkAction, RelatedCreateAction (hex code)
	Icon       string   // For RelatedLinkAction, RelatedCreateAction
	FullWidth  bool     // For RelatedLinkAction, RelatedCreateAction
}

// View represents a managed view on an aspect (App, Element, Task)
type View struct {
	ID                  string
	Name                string
	Type                string // AspectViewList, AspectViewKanban, AspectViewCalendar, AspectViewDashboard, AspectViewAgent
	AllowRecordCreation bool
	DisplayOrder        int

	// Type-specific fields
	Columns             []string // List, Kanban, Calendar
	Density             string   // List only: COMFORTABLE, COMPACT, STANDARD
	Rows                int      // List only
	DisplayByField      string   // Kanban, Calendar
	AllowedFilterFields []string // Kanban only
	DashboardID         string   // Dashboard only
	AgentID             string   // Agent only
}

// TerraformResourceType returns the terraform resource type for this view
func (v *View) TerraformResourceType() string {
	switch v.Type {
	case "AspectViewList":
		return "elementum_list_view"
	case "AspectViewKanban":
		return "elementum_kanban_view"
	case "AspectViewCalendar":
		return "elementum_calendar_view"
	case "AspectViewDashboard":
		return "elementum_dashboard_view"
	case "AspectViewAgent":
		return "elementum_agent_view"
	default:
		return "elementum_list_view"
	}
}

// GetUUIDMappings returns UUID to Terraform reference mappings for a View
func (v *View) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	m[v.ID] = v.TerraformResourceType() + "." + resourceName + ".id"
	return m
}

// ObjectSummary represents an app or element for listing
type ObjectSummary struct {
	ID        string
	Name      string
	Type      string // "App" or "Element"
	Namespace string
}

// ObjectDetails represents detailed information about an object (App, Element, or Task)
type ObjectDetails struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`      // "App", "Element", "Task"
	Namespace     string `json:"namespace"` // lowercase URL-safe identifier
	Handle        string `json:"handle"`    // uppercase short code
	Description   string `json:"description,omitempty"`
	Icon          string `json:"icon,omitempty"`
	Color         string `json:"color,omitempty"`
	SourceType    string `json:"source_type,omitempty"`  // "Elementum", "Snowflake", "BigQuery", "Databricks"
	StorageType   string `json:"storage_type,omitempty"` // "General", "Hybrid"
	CategoryID    string `json:"category_id,omitempty"`
	CategoryName  string `json:"category_name,omitempty"`
	CloudLinkID   string `json:"cloud_link_id,omitempty"`
	CloudLinkName string `json:"cloud_link_name,omitempty"`
	CloudLinkType string `json:"cloud_link_type,omitempty"` // "CloudLinkSnowflake", "CloudLinkApi", etc.
	// Snowflake connection details (only for external tables)
	DatabaseName string `json:"database_name,omitempty"`
	SchemaName   string `json:"schema_name,omitempty"`
	TableName    string `json:"table_name,omitempty"`
	ReadOnly     bool   `json:"read_only,omitempty"`
}

// Category represents an Elementum category for organizing apps, elements, and tasks
type Category struct {
	ID   string
	Name string
}

// Element represents a standalone Element (AspectElement) with all its resources
type Element struct {
	ID              string
	Name            string
	Handle          string
	Namespace       string
	Description     string
	Icon            string
	Color           string
	CategoryID      string
	CategoryName    string // For data source generation
	CloudLinkID     string // CloudLink ID (optional, may be empty)
	CloudLinkName   string // CloudLink name for data source generation
	CommentsEnabled bool   // True if comments are enabled on this element

	// Cloud connection details (from cloudConnection)
	SnowflakeDatabaseName string
	SnowflakeSchemaName   string
	SnowflakeTableName    string
	BigQueryDatasetName   string
	BigQueryTableName     string
	DatabricksDatabase    string
	DatabricksSchema      string
	DatabricksTable       string

	// Resource collections (same as App)
	Fields         []Field
	Layouts        []Layout
	Flows          []Flow
	Automations    []Automation
	Approvals      []ApprovalProcess
	Widgets        []Widget
	Views          []View
	AIFileReaders  []AIFileReader
	Relationships  []Relationship
	AccessPolicies []AccessPolicy  // Access policies for row-level security
	Roles          []Role          // Roles for this element (managed + custom)
	AISearchTables []AISearchTable // AI Search Tables (only Elements have these, not Apps)

	// Dashboards on this element (for dashboard views)
	Dashboards []Dashboard

	// Charts in this element's dashboards
	Charts []Chart

	// Note: Elements do NOT support agents (only Apps do per schema)
}

// GetAspectID implements DiscoverableAspect
func (e *Element) GetAspectID() string { return e.ID }

// GetRelationships implements DiscoverableAspect
func (e *Element) GetRelationships() []Relationship { return e.Relationships }

// GetAutomations implements DiscoverableAspect
func (e *Element) GetAutomations() []Automation { return e.Automations }

// GetAgents implements DiscoverableAspect -- Elements do NOT support agents
func (e *Element) GetAgents() []Agent { return nil }

// AspectTask represents a standalone Task (AspectTask) - distinct from automation Task
type AspectTask struct {
	// Basic info
	ID              string
	Name            string
	Handle          string
	Namespace       string
	Description     string
	Icon            string
	Color           string
	CategoryID      string
	CategoryName    string // For data source generation
	CloudLinkID     string // CloudLink ID (optional, may be empty)
	CloudLinkName   string // CloudLink name for data source generation
	CommentsEnabled bool   // True if comments are enabled on this task

	// Resource collections (tasks support most resources except Flows, Approvals, FileReaders)
	Fields         []Field
	Layouts        []Layout
	Automations    []Automation
	Agents         []Agent // Tasks support agents (unlike Elements)
	Widgets        []Widget
	Views          []View
	Relationships  []Relationship
	AccessPolicies []AccessPolicy // Access policies for row-level security
	Roles          []Role         // Roles for this task (managed + custom)
	// Note: Tasks do NOT support Flows, Approvals, or FileReaders
}

// GetAspectID implements DiscoverableAspect
func (t *AspectTask) GetAspectID() string { return t.ID }

// GetRelationships implements DiscoverableAspect
func (t *AspectTask) GetRelationships() []Relationship { return t.Relationships }

// GetAutomations implements DiscoverableAspect
func (t *AspectTask) GetAutomations() []Automation { return t.Automations }

// GetAgents implements DiscoverableAspect
func (t *AspectTask) GetAgents() []Agent { return t.Agents }

// AllAutomations returns all automations from the app and all discovered resources.
func (a *App) AllAutomations() []Automation {
	all := make([]Automation, 0, len(a.Automations))
	all = append(all, a.Automations...)
	for _, da := range a.DiscoveredApps {
		all = append(all, da.Automations...)
	}
	for _, de := range a.DiscoveredElements {
		all = append(all, de.Automations...)
	}
	for _, dt := range a.DiscoveredTasks {
		all = append(all, dt.Automations...)
	}
	return all
}

// AllLayouts returns all layouts from the app and all discovered resources.
func (a *App) AllLayouts() []Layout {
	all := make([]Layout, 0, len(a.Layouts))
	all = append(all, a.Layouts...)
	for _, da := range a.DiscoveredApps {
		all = append(all, da.Layouts...)
	}
	for _, de := range a.DiscoveredElements {
		all = append(all, de.Layouts...)
	}
	for _, dt := range a.DiscoveredTasks {
		all = append(all, dt.Layouts...)
	}
	return all
}

// AllAgents returns all agents from the app and all discovered resources.
func (a *App) AllAgents() []Agent {
	all := make([]Agent, 0, len(a.Agents))
	all = append(all, a.Agents...)
	for _, da := range a.DiscoveredApps {
		all = append(all, da.Agents...)
	}
	// Elements do NOT support agents
	for _, dt := range a.DiscoveredTasks {
		all = append(all, dt.Agents...)
	}
	return all
}

// Group represents an organization group
type Group struct {
	ID          string
	Name        string
	Approver    bool
	Assignable  bool
	Mentionable bool
	Watchable   bool
	Dynamic     bool
	Tags        []string
	Members     []string // User IDs
}

// CloudLink represents a data connector
type CloudLink struct {
	ID     string
	Name   string
	Type   string // api, bigquery, databricks, snowflake, etc.
	System bool
	Valid  bool
}

// StoredFunctionParameter represents an input parameter of a stored function
type StoredFunctionParameter struct {
	Name string
	Type string
}

// StoredFunction represents a stored function (procedure or UDF) in a CloudLink
type StoredFunction struct {
	ID            string
	Name          string
	DisplayName   string
	Description   string
	DatabaseName  string
	SchemaName    string
	Type          string // PROCEDURE or USER_DEFINED_FUNCTION
	Configured    bool
	CloudLinkID   string
	CloudLinkName string
	Parameters    []StoredFunctionParameter
	ReturnType    string
}

// SnowflakeTableSchema represents a Snowflake table discovered from a CloudLink
type SnowflakeTableSchema struct {
	Name         string
	DatabaseName string
	SchemaName   string
	Type         string // TABLE or VIEW
	Rows         int64
	Bytes        int64
	Columns      []SnowflakeColumn
}

// SnowflakeColumn represents a column in a Snowflake table
type SnowflakeColumn struct {
	Name         string
	DatabaseType string // The raw Snowflake type (VARCHAR, NUMBER, TIMESTAMP_NTZ, etc.)
	Type         string // Normalized Elementum type (TEXT, NUMBER, DATETIME, etc.)
	Nullable     bool
	PrimaryKey   bool
	UniqueKey    bool
	Comment      string
}

// ToElementumCloudType returns the Elementum cloud_type for this column
func (c *SnowflakeColumn) ToElementumCloudType() string {
	// Map Snowflake types to Elementum cloud types
	switch c.Type {
	case "TEXT":
		return "TEXT"
	case "NUMBER", "LONG", "DECIMAL":
		return "NUMBER"
	case "BOOLEAN":
		return "BOOLEAN"
	case "DATETIME":
		return "DATETIME"
	case "DATE":
		return "DATE"
	default:
		return "TEXT" // Default to TEXT for unknown types
	}
}

// Table represents a table (CloudLink table or App)
type Table struct {
	ID            string
	Name          string
	Handle        string
	Description   string
	CloudLinkID   string
	CloudLinkName string // CloudLink name for data source generation
	SourceID      string
	CategoryID    string
	Type          string
	Configured    bool
	CloudManaged  bool // True if table schema is managed by cloud (Snowflake/BigQuery/Databricks)
	// Snowflake connection fields (from cloudConnection)
	SnowflakeDatabaseName string
	SnowflakeSchemaName   string
	SnowflakeTableName    string
	// Table fields (columns)
	Fields []TableField
	// AI Search Tables on this table
	SearchTables []TableSearchTable
}

// TableField represents a field/column in a table
type TableField struct {
	ID   string
	Name string
	Type string
}

// RelatedResource represents a resource related to the main export target
type RelatedResource struct {
	ID           string
	Name         string
	ResourceType string
}

// FileReader represents a file reader for extracting data from files
// Supports multiple types: DocumentModelAi, DocumentModelOCR, DocumentModelJson, DocumentModelXml
type FileReader struct {
	ID           string
	Name         string
	Type         string // __typename: DocumentModelAi, DocumentModelOCR, DocumentModelJson, DocumentModelXml
	Instructions string // AI instructions for extraction (DocumentModelAi only)
	Fields       []FileReaderField
	// For JSON/XML file readers
	Structure string // JSON-encoded structure/values for JSON and XML parsers
}

// FileReaderField represents a field in an AI file reader
type FileReaderField struct {
	ID          string
	Name        string
	Description string
	Type        string // TEXT, NUMBER, DECIMAL, BOOLEAN, DATE, DATETIME
	Required    bool
}

// AIFileReader is an alias for FileReader for backward compatibility
type AIFileReader = FileReader

// AIFileReaderField is an alias for FileReaderField for backward compatibility
type AIFileReaderField = FileReaderField

// FileReaderType constants for the different document model types
const (
	FileReaderTypeAI   = "DocumentModelAi"
	FileReaderTypeOCR  = "DocumentModelOCR"
	FileReaderTypeJSON = "DocumentModelJson"
	FileReaderTypeXML  = "DocumentModelXml"
)

// TerraformResourceType returns the terraform resource type for this file reader
func (fr *FileReader) TerraformResourceType() string {
	switch fr.Type {
	case FileReaderTypeAI:
		return "elementum_ai_file_reader"
	case FileReaderTypeOCR:
		return "elementum_text_file_reader"
	case FileReaderTypeJSON:
		return "elementum_json_file_reader"
	case FileReaderTypeXML:
		return "elementum_xml_file_reader"
	default:
		return "elementum_ai_file_reader" // fallback
	}
}

// Relationship represents an AutoRelation between two objects
type Relationship struct {
	ID                string
	RelatedObjectID   string                 // ID of the target object (App/Element/Task/Table)
	RelatedObjectType string                 // Type: "App", "Element", "Task", "Table"
	RelatedObjectName string                 // Name of the related object
	Columns           []RelationshipColumn   // Field mappings
	Filter            map[string]interface{} // Optional filter
}

// RelationshipColumn represents a field mapping in a relationship
type RelationshipColumn struct {
	FieldID          string // Source field ID
	FieldName        string // Source field name
	RelatedFieldID   string // Target field ID
	RelatedFieldName string // Target field name
}

// RelatedObject represents a discovered related object (app/element) from relationships
type RelatedObject struct {
	ID        string
	Name      string
	Type      string   // "App", "Element", "Task", "Table"
	Namespace string   // For apps/elements
	Fields    []Field  // Discovered fields
	FieldIDs  []string // All field IDs for this object (for quick lookup)
}

// AiProviderConnector represents an AI provider connector discovered from automation tasks
type AiProviderConnector struct {
	ID           string // UUID of the connector
	ModelName    string // Model name (e.g., "GPT-4o", "Claude Sonnet 4")
	ProviderName string // Provider name (e.g., "OpenAI", "Anthropic")
}

// AiProvider represents an AI provider available in the organization
type AiProvider struct {
	ID                string      // Provider ID
	Name              string      // Provider name (e.g., "OpenAI", "Anthropic", "Google")
	AvailableModels   []AiModel   // Models available from this provider
	AvailableFeatures []AiFeature // Features available from this provider
}

// AiModel represents an AI model from a provider
type AiModel struct {
	Name       string // Model name (e.g., "GPT-4o", "Claude Sonnet 4")
	ModelType  string // Model type (e.g., "CHAT", "COMPLETION")
	MultiModal bool   // Whether the model supports multiple modalities
	Legacy     bool   // Whether the model is legacy/deprecated
}

// AiFeature represents an AI feature available from a provider
type AiFeature struct {
	Feature string // Feature identifier
	Name    string // Human-readable feature name
}

// AiProviderConnectorDetail represents a fully-configured AI provider connector
// This includes more details than AiProviderConnector which is used for discovery
type AiProviderConnectorDetail struct {
	ID                   string      // Connector ID
	Model                AiModel     // The AI model
	ProviderID           string      // Provider ID
	ProviderName         string      // Provider name
	ApiType              string      // API type (e.g., "OPENAI", "ANTHROPIC")
	CostPerMillionTokens float64     // Cost per million tokens
	AvailableFeatures    []AiFeature // Features available on this connector
}

// GetUUIDMappings returns UUID to Terraform reference mappings for an App
func (a *App) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	ref := "elementum_app." + resourceName
	m[a.ID] = ref + ".id"

	// Add system field mappings
	m[a.ID+".title_field_id"] = ref + ".title_field_id"
	m[a.ID+".status_field_id"] = ref + ".status_field_id"

	return m
}

// GetAspectID implements DiscoverableAspect
func (a *App) GetAspectID() string { return a.ID }

// GetRelationships implements DiscoverableAspect
func (a *App) GetRelationships() []Relationship { return a.Relationships }

// GetAutomations implements DiscoverableAspect
func (a *App) GetAutomations() []Automation { return a.Automations }

// GetAgents implements DiscoverableAspect
func (a *App) GetAgents() []Agent { return a.Agents }

// GetUUIDMappings returns UUID to Terraform reference mappings for a Field
func (f *Field) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	resourceType := "elementum_" + f.Type + "_field"
	m[f.ID] = resourceType + "." + resourceName + ".id"

	// Add field option ID mappings for dropdown/multi_select fields
	// This enables beautification of status values and dropdown option IDs
	for _, opt := range f.Options {
		if opt.ID != "" && opt.Label != "" {
			// Map option ID to a local reference using label lookup
			m[opt.ID] = fmt.Sprintf("local.%s_options_by_label[\"%s\"]", resourceName, opt.Label)
		}
	}

	return m
}

// GetOptionIDMappings returns a map of field option IDs to their labels
// This is used to generate locals blocks for option lookups
func (f *Field) GetOptionIDMappings() map[string]string {
	m := make(map[string]string)
	for _, opt := range f.Options {
		if opt.ID != "" && opt.Label != "" {
			m[opt.ID] = opt.Label
		}
	}
	return m
}

// GetUUIDMappings returns UUID to Terraform reference mappings for an Automation
func (a *Automation) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	ref := "elementum_automation." + resourceName
	m[a.ID] = ref + ".id"
	m[a.WorkflowID] = ref + ".workflow_id"
	return m
}

// GetUUIDMappings returns UUID to Terraform reference mappings for a Trigger
func (t *Trigger) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	resourceType := "elementum_" + t.Type + "_trigger"
	ref := resourceType + "." + resourceName
	m[t.ID] = ref + ".id"

	// Add computed refs for record-based triggers
	if isRecordBasedTrigger(t.Type) {
		// These are special computed values available in value references
		m["trigger.record.10000001-2000-4000-a000-800000000000"] = ref + ".record_id"
		m["trigger.record.10000001-2000-4000-a000-800000000005"] = ref + ".record_url"
		m["trigger.record.10000001-2000-4000-a000-800000000003"] = ref + ".attachments"
	}

	// Add parameter_ids mappings for on_demand triggers
	// Maps parameter UUIDs to elementum_on_demand_trigger.<name>.parameter_ids["<param_name>"]
	if t.Type == "on_demand" && t.RawData != nil {
		if params, ok := t.RawData["parameters"].([]interface{}); ok {
			for _, paramInterface := range params {
				if param, ok := paramInterface.(map[string]interface{}); ok {
					paramID, hasID := param["id"].(string)
					paramName, hasName := param["name"].(string)
					if hasID && hasName && paramID != "" && paramName != "" {
						m[paramID] = ref + `.parameter_ids["` + paramName + `"]`
					}
				}
			}
		}
	}

	// NOTE: Intentionally NOT mapping datamine_id to trigger.datamine_id as that creates a self-reference
	// The datamine should be exported separately and referenced as elementum_datamine.<name>.id

	// Add field refs mappings for value reference beautification
	// Maps "trigger.record.{uuid}" -> trigger.refs["Field Name"]
	for fieldRef, fieldName := range t.FieldRefs {
		// fieldRef is like "record.{uuid}" or "{uuid}"
		triggerRef := "trigger." + fieldRef
		if fieldRef[0] != 'r' { // doesn't start with "record."
			triggerRef = "trigger.record." + fieldRef
		}
		m[triggerRef] = ref + `.refs["` + fieldName + `"]`
	}

	return m
}

// GetUUIDMappings returns UUID to Terraform reference mappings for a Task
func (t *Task) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	resourceType := "elementum_" + t.Type + "_task"
	ref := resourceType + "." + resourceName
	m[t.ID] = ref + ".id"

	// Add computed refs based on task type
	if t.Type == "record_search" {
		m[t.ID+".first_record_id"] = ref + ".first_record_id"
		m[t.ID+".record_count"] = ref + ".record_count"
	}

	// Add field refs mappings for value reference beautification
	// Maps "task.{task_id}.{property}" -> task.refs["Property Name"]
	for propPath, propName := range t.FieldRefs {
		taskRef := "task." + t.ID + "." + propPath
		m[taskRef] = ref + `.refs["` + propName + `"]`
	}

	return m
}

// GetUUIDMappings returns UUID to Terraform reference mappings for a Layout
func (l *Layout) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	m[l.ID] = "elementum_layout." + resourceName + ".id"
	return m
}

// GetUUIDMappings returns UUID to Terraform reference mappings for a Flow
func (f *Flow) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	m[f.ID] = "elementum_flow." + resourceName + ".id"
	return m
}

// GetUUIDMappings returns UUID to Terraform reference mappings for an Agent
func (a *Agent) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	m[a.ID] = "elementum_agent." + resourceName + ".id"
	return m
}

// GetUUIDMappings returns UUID to Terraform reference mappings for an AgentTool
func (at *AgentTool) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	m[at.ID] = at.TerraformResourceType() + "." + resourceName + ".id"
	return m
}

// GetUUIDMappings returns UUID to Terraform reference mappings for a Widget
func (w *Widget) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	m[w.ID] = "elementum_widget." + resourceName + ".id"
	return m
}

// GetUUIDMappings returns UUID to Terraform reference mappings for a FileReader
func (fr *FileReader) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	m[fr.ID] = fr.TerraformResourceType() + "." + resourceName + ".id"
	return m
}

// GetUUIDMappings returns UUID to Terraform reference mappings for an ApprovalProcess
func (ap *ApprovalProcess) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	m[ap.ID] = "elementum_approval_process." + resourceName + ".id"
	return m
}

// isRecordBasedTrigger checks if a trigger type is record-based
func isRecordBasedTrigger(triggerType string) bool {
	return triggerType == "record_created" || triggerType == "record_updated"
}

// GetUUIDMappings returns UUID to Terraform reference mappings for a Table
func (t *Table) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	m[t.ID] = "elementum_table." + resourceName + ".id"
	return m
}

// GetUUIDMappings returns UUID to Terraform reference mappings for a RelatedObject
func (ro *RelatedObject) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	resourceType := "data.elementum_app"
	switch ro.Type {
	case "Element":
		resourceType = "data.elementum_element"
	case "Task":
		resourceType = "data.elementum_task"
	}
	m[ro.ID] = resourceType + "." + resourceName + ".id"

	// Map system field IDs to data source attributes
	for _, field := range ro.Fields {
		switch field.Name {
		case "Title", "title":
			m[field.ID] = resourceType + "." + resourceName + ".title_field_id"
		case "Status", "status":
			m[field.ID] = resourceType + "." + resourceName + ".status_field_id"
		case "ID", "id":
			m[field.ID] = resourceType + "." + resourceName + ".id_field_id"
		}
		// Note: Custom fields are NOT automatically mapped since there's no easy
		// way to reference them via data source. Users will need to manually
		// add field data source lookups for custom fields.
	}

	return m
}

// GetUUIDMappings returns UUID to Terraform reference mappings for an AspectTask
func (t *AspectTask) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	ref := "elementum_task." + resourceName
	m[t.ID] = ref + ".id"

	// Add system field mappings
	m[t.ID+".title_field_id"] = ref + ".title_field_id"
	m[t.ID+".status_field_id"] = ref + ".status_field_id"
	m[t.ID+".id_field_id"] = ref + ".id_field_id"

	return m
}

// GetUUIDMappings returns UUID to Terraform reference mappings for a Relationship
func (r *Relationship) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	m[r.ID] = "elementum_relationship." + resourceName + ".id"
	return m
}

// AccessPolicy represents an access policy for row-level security on an App, Element, or Task
type AccessPolicy struct {
	ID       string                 // Access policy ID
	ObjectID string                 // ID of the App/Element/Task this policy belongs to
	Filter   map[string]interface{} // Filter criteria for matching records
	UserIDs  []string               // User IDs who can access records matching the filter
	GroupIDs []string               // Group IDs whose members can access records matching the filter
	// For beautification - map IDs to names
	UserEmails map[string]string // userID -> email (for data source lookup)
	GroupNames map[string]string // groupID -> name (for data source lookup)
}

// GetUUIDMappings returns UUID to Terraform reference mappings for an AccessPolicy
func (ap *AccessPolicy) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	m[ap.ID] = "elementum_access_policy." + resourceName + ".id"
	return m
}

// Role represents a role discovered from an App or Element
type Role struct {
	ID          string
	Name        string
	Description string
	AutoShare   []string // APPROVAL_CHAIN, ASSIGNEE, MENTION, RECORD_SHARE, WATCHER
	Managed     bool     // true = system role (data source), false = custom role (resource)
	Tags        []string

	// Membership - resolved to human-readable references
	Users  []RoleMember // User IDs + emails for data source generation
	Groups []RoleMember // Group IDs + names for data source generation

	// Permissions (populated for all roles)
	Permissions []RolePermission
}

// RoleMember represents a user or group assigned to a role
type RoleMember struct {
	ID   string // UUID
	Name string // Email for users, Name for groups
}

// RolePermission represents a permission configuration for a role
type RolePermission struct {
	Group             string   // RECORDS, AUTOMATIONS, AGENTS, etc.
	Level             string   // ADMIN, EDIT, VIEW, NONE, CUSTOM
	CustomPermissions []string // Specific permissions when level is CUSTOM
}

// GetUUIDMappings returns UUID to Terraform reference mappings for a Role
func (r *Role) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	if r.Managed {
		m[r.ID] = "data.elementum_role." + resourceName + ".id"
	} else {
		m[r.ID] = "elementum_role." + resourceName + ".id"
	}
	return m
}

// ==============================================================================
// Unified Discovery Types
// ==============================================================================

// UnifiedDiscoveryContext tracks all discovered objects across the entire graph
// during recursive discovery from a root app or element
type UnifiedDiscoveryContext struct {
	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Cycle detection - single map since IDs are globally unique UUIDs
	VisitedIDs map[string]bool

	// Queue deduplication - track items already in queue (not yet processed)
	QueuedIDs map[string]bool

	// Root object (don't re-export as "discovered")
	RootID   string
	RootType string // "App", "Element"

	// Fully discovered objects by type (maps for deduplication)
	Apps       map[string]*App
	Elements   map[string]*Element
	Tasks      map[string]*AspectTask // AspectTask - distinct from automation tasks
	Tables     map[string]*Table
	Datamines  map[string]*Datamine
	CloudLinks map[string]*CloudLink // Track discovered CloudLinks
	Charts     map[string]*Chart     // Track discovered Charts

	// Processing queue for breadth-first discovery
	Queue []DiscoveryItem
}

// DiscoveryItem represents an item in the discovery queue
type DiscoveryItem struct {
	ID   string
	Type string // "App", "Element", "Table", "Datamine", "CloudLink"
}

// NewUnifiedDiscoveryContext creates a new context with the root object marked as visited
func NewUnifiedDiscoveryContext(rootID, rootType string) *UnifiedDiscoveryContext {
	return &UnifiedDiscoveryContext{
		VisitedIDs: map[string]bool{rootID: true},
		QueuedIDs:  make(map[string]bool),
		RootID:     rootID,
		RootType:   rootType,
		Apps:       make(map[string]*App),
		Elements:   make(map[string]*Element),
		Tasks:      make(map[string]*AspectTask),
		Tables:     make(map[string]*Table),
		Datamines:  make(map[string]*Datamine),
		CloudLinks: make(map[string]*CloudLink),
		Charts:     make(map[string]*Chart),
		Queue:      []DiscoveryItem{},
	}
}

// Enqueue adds an item to the discovery queue if not already visited or queued.
// Returns true if the item was added, false if it was already visited/queued.
func (ctx *UnifiedDiscoveryContext) Enqueue(id, itemType string) bool {
	if id == "" || ctx.VisitedIDs[id] || ctx.QueuedIDs[id] {
		return false
	}
	ctx.QueuedIDs[id] = true
	ctx.Queue = append(ctx.Queue, DiscoveryItem{ID: id, Type: itemType})
	return true
}

// Dequeue removes and returns the next item from the queue, or nil if empty
func (ctx *UnifiedDiscoveryContext) Dequeue() *DiscoveryItem {
	if len(ctx.Queue) == 0 {
		return nil
	}
	item := ctx.Queue[0]
	ctx.Queue = ctx.Queue[1:]
	return &item
}

// MarkVisited marks an ID as visited
func (ctx *UnifiedDiscoveryContext) MarkVisited(id string) {
	ctx.VisitedIDs[id] = true
}

// IsVisited checks if an ID has been visited
func (ctx *UnifiedDiscoveryContext) IsVisited(id string) bool {
	return ctx.VisitedIDs[id]
}

// ==============================================================================
// Thread-Safe Methods for Concurrent Discovery
// ==============================================================================

// EnqueueSafe is a thread-safe version of Enqueue
func (ctx *UnifiedDiscoveryContext) EnqueueSafe(id, itemType string) bool {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	if id == "" || ctx.VisitedIDs[id] || ctx.QueuedIDs[id] {
		return false
	}
	ctx.QueuedIDs[id] = true
	ctx.Queue = append(ctx.Queue, DiscoveryItem{ID: id, Type: itemType})
	return true
}

// DequeueBatch removes and returns up to n items from the queue
// Returns an empty slice if the queue is empty
func (ctx *UnifiedDiscoveryContext) DequeueBatch(n int) []DiscoveryItem {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	if len(ctx.Queue) == 0 {
		return nil
	}

	// Take at most n items
	count := n
	if count > len(ctx.Queue) {
		count = len(ctx.Queue)
	}

	items := make([]DiscoveryItem, count)
	copy(items, ctx.Queue[:count])
	ctx.Queue = ctx.Queue[count:]
	return items
}

// MarkVisitedSafe is a thread-safe version of MarkVisited
func (ctx *UnifiedDiscoveryContext) MarkVisitedSafe(id string) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.VisitedIDs[id] = true
}

// IsVisitedSafe is a thread-safe version of IsVisited
func (ctx *UnifiedDiscoveryContext) IsVisitedSafe(id string) bool {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.VisitedIDs[id]
}

// StoreAppSafe stores an app in a thread-safe manner
func (ctx *UnifiedDiscoveryContext) StoreAppSafe(id string, app *App) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.Apps[id] = app
}

// StoreElementSafe stores an element in a thread-safe manner
func (ctx *UnifiedDiscoveryContext) StoreElementSafe(id string, element *Element) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.Elements[id] = element
}

// StoreTableSafe stores a table in a thread-safe manner
func (ctx *UnifiedDiscoveryContext) StoreTableSafe(id string, table *Table) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.Tables[id] = table
}

// StoreDatamineSafe stores a datamine in a thread-safe manner
func (ctx *UnifiedDiscoveryContext) StoreDatamineSafe(id string, datamine *Datamine) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.Datamines[id] = datamine
}

// StoreCloudLinkSafe stores a cloudlink in a thread-safe manner
func (ctx *UnifiedDiscoveryContext) StoreCloudLinkSafe(id string, cloudlink *CloudLink) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.CloudLinks[id] = cloudlink
}

// StoreChartSafe stores a chart in a thread-safe manner
func (ctx *UnifiedDiscoveryContext) StoreChartSafe(id string, chart *Chart) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.Charts[id] = chart
}

// QueueLen returns the current queue length in a thread-safe manner
func (ctx *UnifiedDiscoveryContext) QueueLen() int {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return len(ctx.Queue)
}

// ==============================================================================
// Phone Provider and Phone Service Types
// ==============================================================================

// PhoneProvider represents a phone provider at the organization level
type PhoneProvider struct {
	ID           string // UUID
	Name         string
	ProviderType string // "elementum", "twilio", "elementum_twilio", "unconfigured"
	System       bool   // True for system-managed providers
}

// GetUUIDMappings returns UUID to Terraform reference mappings for a PhoneProvider
func (pp *PhoneProvider) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	if pp.System {
		// System providers are looked up via data source
		m[pp.ID] = "data.elementum_phone_provider." + resourceName + ".id"
	} else {
		m[pp.ID] = "elementum_phone_provider." + resourceName + ".id"
	}
	return m
}

// PhoneService represents a phone service on an app
type PhoneService struct {
	ID                     string // UUID
	PhoneNumber            string // E.164 format (e.g., +14155551234)
	ProviderID             string // Phone provider ID
	ProviderName           string // Provider name for beautification
	AgentID                string // AI agent ID
	AgentName              string // Agent name for beautification
	AspectID               string // App ID
	RegistrationProperties PhoneRegistrationProperties
}

// PhoneRegistrationProperties represents registration properties for a phone service
type PhoneRegistrationProperties struct {
	GenerateFirstMessage  bool
	ExpandLanguages       bool
	DefaultLanguage       string
	EnableLanguageRouting bool
	PreferredLanguages    []string
}

// GetUUIDMappings returns UUID to Terraform reference mappings for a PhoneService
func (ps *PhoneService) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	m[ps.ID] = "elementum_phone_service." + resourceName + ".id"
	return m
}

// ==============================================================================
// AI Search Table Types
// ==============================================================================

// AISearchTable represents an AI search table on an App or Element
type AISearchTable struct {
	ID                      string
	ObjectID                string
	FieldID                 string
	FieldName               string // For beautification
	AIProviderConnectorID   string
	AIProviderConnectorName string // For beautification
	AttributeFieldIDs       []string
	Duration                string // DAYS, HOURS, MINUTES, SECONDS
	TargetLag               int
	Warehouse               string
}

// GetUUIDMappings returns UUID to Terraform reference mappings for an AISearchTable
func (st *AISearchTable) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	m[st.ID] = "elementum_ai_search_table." + resourceName + ".id"
	return m
}

// ==============================================================================
// Table Search Table Types
// ==============================================================================

// TableSearchTable represents an AI search table on a Table (not an Aspect)
type TableSearchTable struct {
	ID                    string
	TableID               string
	FieldID               string
	FieldName             string // For beautification
	AIProviderConnectorID string
	AttributeFieldIDs     []string
	Duration              string // DAYS, HOURS, MINUTES, SECONDS
	TargetLag             int
	Warehouse             string
}

// GetUUIDMappings returns UUID to Terraform reference mappings for a TableSearchTable
func (st *TableSearchTable) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	m[st.ID] = "elementum_table_search_table." + resourceName + ".id"
	return m
}

// ==============================================================================
// Managed View Order Types
// ==============================================================================

// ManagedViewOrder represents the view ordering for an aspect
type ManagedViewOrder struct {
	ObjectID string   // Same as aspect ID (singleton)
	ViewIDs  []string // Ordered list of view IDs
}

// GetUUIDMappings returns UUID to Terraform reference mappings for a ManagedViewOrder
func (mvo *ManagedViewOrder) GetUUIDMappings(resourceName string) map[string]string {
	// ManagedViewOrder doesn't have its own unique ID - it uses object_id (app/element ID)
	// We should NOT map the ObjectID here because it would incorrectly replace
	// app/element ID references with managed_view_order references.
	// The aspect_id for views should reference the app, not the managed_view_order.
	return make(map[string]string)
}

// ==============================================================================
// Dashboard Types
// ==============================================================================

// Dashboard represents a dashboard resource
type Dashboard struct {
	ID        string
	Name      string
	OwnerID   string // ID of the owning aspect or user/group
	OwnerType string // APP_ASPECT, ELEMENT_ASPECT, USER, GROUP, TASK_ASPECT
	Favorite  bool
	Tags      []string
	Widgets   []DashboardWidget
}

// GetUUIDMappings returns UUID to Terraform reference mappings for a Dashboard
func (d *Dashboard) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	m[d.ID] = "elementum_dashboard." + resourceName + ".id"
	return m
}

// DashboardWidget represents a widget on a dashboard
type DashboardWidget struct {
	ID           string
	DashboardID  string
	Name         string
	Size         string
	Color        string
	Icon         string
	DisplayOrder int
	// AspectWidget fields
	AspectID string
	Filter   string // JSON-encoded filter
	Sort     string // JSON-encoded sort
	// ChartWidget fields
	ChartID string
}

// GetUUIDMappings returns UUID to Terraform reference mappings for a DashboardWidget
func (dw *DashboardWidget) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	m[dw.ID] = "elementum_dashboard_widget." + resourceName + ".id"
	return m
}

// ==============================================================================
// Chart Types
// ==============================================================================

// Chart represents a chart resource
type Chart struct {
	ID            string
	Name          string
	Type          string   // BAR, LINE, PIE, DOUGHNUT, RADAR, POLAR, BUBBLE, SCATTER, SINGLEBAR, SINGLELINE, SINGLEVALUE
	Target        *float64 // Optional target value
	Properties    ChartProperties
	Subscriptions []ChartSubscription
}

// ChartProperties represents chart display properties
type ChartProperties struct {
	BeginAtZero *bool
	Fill        *bool
	Horizontal  *bool
	Legend      *bool
	Stacked     *bool
	Target      *float64
}

// ChartSubscription represents a data subscription in a chart
type ChartSubscription struct {
	DatasourceID string
	Columns      []ChartColumn
	GroupBys     []ChartGroupBy
	Joins        []ChartJoin
	Filter       string // JSON-encoded filter
	Sort         string // JSON-encoded sort
	Limit        string
	Tags         []string
	With         string // JSON-encoded with config
}

// ChartColumn represents a column in a chart subscription
type ChartColumn struct {
	ColumnID    string
	Aggregation string
	Name        string
	Value       string
	ValueType   string
	Expression  string // JSON-encoded expression
}

// ChartGroupBy represents a group by in a chart subscription
type ChartGroupBy struct {
	ColumnID   string
	Expression string // JSON-encoded expression
}

// ChartJoin represents a join in a chart subscription
type ChartJoin struct {
	DatasourceID    string
	LeftColumnID    string
	RightColumnID   string
	LeftExpression  string // JSON-encoded expression
	RightExpression string // JSON-encoded expression
	Type            string // INNER, LEFT, RIGHT, FULL
}

// GetUUIDMappings returns UUID to Terraform reference mappings for a Chart
func (c *Chart) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	m[c.ID] = "elementum_chart." + resourceName + ".id"
	return m
}

// ==============================================================================
// Feature Flag Types
// ==============================================================================

// FeatureFlag represents a feature flag in the organization
type FeatureFlag struct {
	Feature string // Feature name (e.g., "AI_CHAT", "AGENT_AUTOMATIONS")
	Enabled bool   // Whether the feature is enabled for this organization
}

// ==============================================================================
// User Types
// ==============================================================================

// User represents an organization user
type User struct {
	ID        string
	Email     string
	Name      string
	FirstName string
	LastName  string
	Status    string // ACTIVE, INACTIVE, ANONYMOUS
	JobTitle  string
}

// ==============================================================================
// Record Types (for ei records command)
// ==============================================================================

// Record represents a single record fetched from an Aspect
type Record struct {
	ID        string                 // Record UUID
	Handle    string                 // Human-readable handle (e.g., "TKT-123")
	Title     string                 // Record title
	Status    string                 // Status label
	StatusID  string                 // Status option ID
	URL       string                 // Full URL to the record
	CreatedAt string                 // ISO timestamp
	UpdatedAt string                 // ISO timestamp
	CreatedBy string                 // Creator's name
	Data      map[string]interface{} // All field data keyed by field name
}

// RecordListResult contains records and pagination info
type RecordListResult struct {
	Records     []Record
	Total       int
	HasNextPage bool
	EndCursor   string
	AspectID    string
	AspectName  string
	AspectType  string // "App", "Element", "Task"
	Namespace   string
}

// RecordListOptions configures the records query
type RecordListOptions struct {
	Limit   int      // Max records per request (default 25)
	After   string   // Pagination cursor
	All     bool     // Auto-paginate to fetch all records
	Columns []string // Columns to display (for table output)
}

// ==============================================================================
// GetID methods for deterministic sorting
// ==============================================================================

func (f Field) GetID() string               { return f.ID }
func (l Layout) GetID() string              { return l.ID }
func (f Flow) GetID() string                { return f.ID }
func (a Automation) GetID() string          { return a.ID }
func (a Agent) GetID() string               { return a.ID }
func (ap ApprovalProcess) GetID() string    { return ap.ID }
func (w Widget) GetID() string              { return w.ID }
func (v View) GetID() string                { return v.ID }
func (r AIFileReader) GetID() string        { return r.ID }
func (r Relationship) GetID() string        { return r.ID }
func (ro RelatedObject) GetID() string      { return ro.ID }
func (ap AccessPolicy) GetID() string       { return ap.ID }
func (r Role) GetID() string                { return r.ID }
func (d *Datamine) GetID() string           { return d.ID }
func (t *Table) GetID() string              { return t.ID }
func (a *App) GetID() string                { return a.ID }
func (e *Element) GetID() string            { return e.ID }
func (t *AspectTask) GetID() string         { return t.ID }
func (c *CloudLink) GetID() string          { return c.ID }
func (s *StoredFunction) GetID() string     { return s.ID }
func (p PhoneService) GetID() string        { return p.ID }
func (s AISearchTable) GetID() string       { return s.ID }
func (d Dashboard) GetID() string           { return d.ID }
func (c Chart) GetID() string               { return c.ID }
func (t Trigger) GetID() string             { return t.ID }
func (t Task) GetID() string                { return t.ID }
func (t AgentTool) GetID() string           { return t.ID }
func (a AiProviderConnector) GetID() string { return a.ID }

// SortForDeterministicOutput sorts all slices in the App by ID for consistent output
func (app *App) SortForDeterministicOutput() {
	sortByID(app.Fields)
	sortByID(app.Layouts)
	sortByID(app.Flows)
	sortByID(app.Automations)
	sortByID(app.Agents)
	sortByID(app.Approvals)
	sortByID(app.Widgets)
	sortByID(app.Views)
	sortByID(app.AIFileReaders)
	sortByID(app.Relationships)
	sortByID(app.RelatedObjects)
	sortByID(app.AccessPolicies)
	sortByID(app.Roles)
	sortByID(app.PhoneServices)
	sortByID(app.AISearchTables)
	sortByID(app.Dashboards)
	sortByID(app.Charts)
	sortByID(app.DiscoveredAiProviderConnectors)

	// Sort pointer slices
	sortPtrByID(app.ReferencedDatamines)
	sortPtrByID(app.ReferencedTables)
	sortPtrByID(app.DiscoveredApps)
	sortPtrByID(app.DiscoveredElements)
	sortPtrByID(app.DiscoveredTasks)
	sortPtrByID(app.DiscoveredTables)
	sortPtrByID(app.DiscoveredDatamines)
	sortPtrByID(app.DiscoveredCloudLinks)
	sortPtrByID(app.DiscoveredStoredFunctions)

	// Sort nested items within automations
	for i := range app.Automations {
		sortByID(app.Automations[i].Triggers)
		sortByID(app.Automations[i].Tasks)
	}

	// Sort nested items within agents
	for i := range app.Agents {
		sortByID(app.Agents[i].Tools)
	}

	// Recursively sort discovered apps and elements
	for _, a := range app.DiscoveredApps {
		a.SortForDeterministicOutput()
	}
	for _, e := range app.DiscoveredElements {
		e.SortForDeterministicOutput()
	}
}

// SortForDeterministicOutput sorts all slices in the Element by ID
func (elem *Element) SortForDeterministicOutput() {
	sortByID(elem.Fields)
	sortByID(elem.Layouts)
	sortByID(elem.Flows)
	sortByID(elem.Automations)
	sortByID(elem.Widgets)
	sortByID(elem.Views)
	sortByID(elem.AccessPolicies)
	sortByID(elem.Roles)
	sortByID(elem.AISearchTables)
	sortByID(elem.Dashboards)

	for i := range elem.Automations {
		sortByID(elem.Automations[i].Triggers)
		sortByID(elem.Automations[i].Tasks)
	}
}
