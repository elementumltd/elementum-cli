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
	"net/url"
	"sort"
	"strings"
	"sync"

	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/elementumltd/elementum-cli/internal/fieldtypes"
	"github.com/elementumltd/elementum-cli/logger"
)

// CLI-specific queries for discovering app resources
// Note: Trigger and task type registries are now shared via internal/client package

// extractTriggerFromDiscovery converts genqlient DiscoveryTriggerFields to Trigger
// This is a lightweight extraction that only populates fields available in the discovery fragment.
func extractTriggerFromDiscovery(trigger client.DiscoveryTriggerFields) Trigger {
	typename := ""
	if trigger.GetTypename() != nil {
		typename = *trigger.GetTypename()
	}

	result := Trigger{
		ID:   trigger.GetId(),
		Type: mapTriggerType(typename),
		Name: mapTriggerType(typename), // Use type as name since triggers don't have explicit names
	}

	// Extract datamine reference if present
	if dt, ok := trigger.(*client.DiscoveryTriggerFieldsWorkflowDatamineTrigger); ok {
		if dt.Datamine != nil {
			result.DatamineID = dt.Datamine.Id
		}
	}

	return result
}

// extractTaskFromDiscovery converts genqlient DiscoveryTaskFields to Task
// This is a lightweight extraction that only populates fields available in the discovery fragment.
// For full task details (RawData, AI connector info, etc.), use the heavy dynamic query.
func extractTaskFromDiscovery(task client.DiscoveryTaskFields, workflowID string) Task {
	typename := ""
	if task.GetTypename() != nil {
		typename = *task.GetTypename()
	}

	name := ""
	if task.GetName() != nil {
		name = *task.GetName()
	}

	result := Task{
		ID:         task.GetId(),
		Name:       name,
		Type:       mapTaskType(typename),
		WorkflowID: workflowID,
	}

	// Extract previous task ID (GetPrevious returns *interface, need to dereference)
	if prev := task.GetPrevious(); prev != nil {
		result.ParentID = (*prev).GetId()
	}

	// Extract aspect references based on task type.
	//
	// genqlient generates a distinct concrete type per query site even when
	// those types share a fragment, so a `switch t := task.(type)` on the
	// concrete `*client.DiscoveryTaskFields...` names never matches tasks
	// that actually came from a full query (e.g. those prefixed
	// `GetWorkflowDetailsForDiscoveryOrganizationAspectWorkflow...`).
	// Instead we dispatch via the shared getter interfaces declared on the
	// fragment, so every query site flows through the same code.
	switch typename {
	case "WorkflowCreateRecordTask",
		"WorkflowUpdateFieldTask",
		"WorkflowRecordSearchTask",
		"WorkflowAspectRecordFieldLockingTask",
		"WorkflowBulkExcelTask":
		if t, ok := task.(interface {
			GetAspect() *client.DiscoveryTaskFieldsAspect
		}); ok {
			if a := t.GetAspect(); a != nil {
				result.ObjectID = (*a).GetId()
			}
		}
	case "WorkflowFindRelatedRecordsTask":
		if t, ok := task.(interface {
			GetRelatedAspect() client.DiscoveryTaskFieldsRelatedAspect
		}); ok {
			if a := t.GetRelatedAspect(); a != nil {
				result.RelatedObjectID = a.GetId()
			}
		}
	case "WorkflowAiSearchTableTask":
		// An AI search table task targets a SearchTable that lives on a parent
		// aspect (typically an Element). For recursive discovery we need the
		// parent aspect ID so the element gets enqueued and exported.
		if t, ok := task.(interface {
			GetSearchTable() *client.DiscoveryTaskFieldsSearchTableAspectSearchTable
		}); ok {
			if st := t.GetSearchTable(); st != nil {
				if aspect := (*st).GetAspect(); aspect != nil {
					result.ObjectID = aspect.GetId()
				}
			}
		}
	}

	return result
}

// ParseNamespaceFromURL extracts the app namespace from an Elementum URL
// Supports URLs like: https://org.elementum.io/app/namespace/...
func ParseNamespaceFromURL(input string) (string, error) {
	logger.Debug("parsing namespace from URL", "input", input)

	// If it doesn't look like a URL, treat it as a namespace
	if !strings.Contains(input, "://") && !strings.Contains(input, "/") {
		return input, nil
	}

	// Parse as URL
	u, err := url.Parse(input)
	if err != nil {
		// If it fails to parse, might just be a namespace with invalid URL chars
		// Try to extract namespace pattern
		parts := strings.Split(strings.TrimPrefix(input, "/"), "/")
		for i, part := range parts {
			if (part == "app" || part == "apps") && i+1 < len(parts) {
				return parts[i+1], nil
			}
		}
		return "", fmt.Errorf("could not extract namespace from input: %s", input)
	}

	// Extract namespace from path: /app/namespace/... or /apps/namespace/...
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	logger.Debug("URL parsed into parts", "parts", parts, "path", u.Path)
	for i, part := range parts {
		if (part == "app" || part == "apps") && i+1 < len(parts) {
			namespace := parts[i+1]
			logger.Debug("namespace extracted successfully", "namespace", namespace)
			return namespace, nil
		}
	}

	return "", fmt.Errorf("no app namespace found in URL: %s", input)
}

// GetAppByNamespace retrieves an app by its namespace.
// If ec is provided, it is passed through to GetApp for error tracking.
func GetAppByNamespace(ctx context.Context, c *client.Client, namespace string, ec ...*ErrorCollector) (*App, error) {
	// Use genqlient-generated SearchAspects function
	resp, err := client.SearchAspects(ctx, c.Genqlient(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search for app: %w", err)
	}

	// Use helper to extract aspects into simplified data structure
	aspects := client.ExtractAspects(resp)

	// Find app with matching namespace
	for _, aspect := range aspects {
		if aspect.Typename == "AspectApp" && aspect.Namespace == namespace {
			// Found it, now get full details
			var collector *ErrorCollector
			if len(ec) > 0 {
				collector = ec[0]
			}
			return GetApp(ctx, c, aspect.ID, collector)
		}
	}

	return nil, fmt.Errorf("app not found with namespace: %s", namespace)
}

// GetAppByNamespaceParallel is a parallel version of GetAppByNamespace
func GetAppByNamespaceParallel(ctx context.Context, c *client.Client, namespace string, config *ParallelConfig, ec ...*ErrorCollector) (*App, error) {
	resp, err := client.SearchAspects(ctx, c.Genqlient(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search for app: %w", err)
	}

	aspects := client.ExtractAspects(resp)

	for _, aspect := range aspects {
		if aspect.Typename == "AspectApp" && aspect.Namespace == namespace {
			var collector *ErrorCollector
			if len(ec) > 0 {
				collector = ec[0]
			}
			return GetAppParallel(ctx, c, aspect.ID, config, collector)
		}
	}

	return nil, fmt.Errorf("app not found with namespace: %s", namespace)
}

// ListObjects returns a list of all objects (apps and elements) in the organization
func ListObjects(ctx context.Context, c *client.Client) ([]ObjectSummary, error) {
	// Use genqlient-generated SearchAspects function without filter to get all objects
	resp, err := client.SearchAspects(ctx, c.Genqlient(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	// Use helper to extract aspects into simplified data structure
	aspects := client.ExtractAspects(resp)

	objects := make([]ObjectSummary, 0, len(aspects))
	for _, aspect := range aspects {
		objectType := mapObjectType(aspect.Typename)
		objects = append(objects, ObjectSummary{
			ID:        aspect.ID,
			Name:      aspect.Name,
			Type:      objectType,
			Namespace: aspect.Namespace,
		})
	}

	return objects, nil
}

// GetObjectDetails returns detailed information about a single object by namespace
func GetObjectDetails(ctx context.Context, c *client.Client, namespace string) (*ObjectDetails, error) {
	resp, err := client.GetAspectDetails(ctx, c.Genqlient(), namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get object details: %w", err)
	}

	data := client.ExtractAspectDetails(resp)
	if data == nil {
		return nil, fmt.Errorf("object with namespace %q not found", namespace)
	}

	return &ObjectDetails{
		ID:            data.ID,
		Name:          data.Name,
		Type:          data.Type,
		Namespace:     data.Namespace,
		Handle:        data.Handle,
		Description:   data.Description,
		Icon:          data.Icon,
		Color:         data.Color,
		SourceType:    data.SourceType,
		StorageType:   data.StorageType,
		CategoryID:    data.CategoryID,
		CategoryName:  data.CategoryName,
		CloudLinkID:   data.CloudLinkID,
		CloudLinkName: data.CloudLinkName,
		CloudLinkType: data.CloudLinkType,
		DatabaseName:  data.DatabaseName,
		SchemaName:    data.SchemaName,
		TableName:     data.TableName,
		ReadOnly:      data.ReadOnly,
	}, nil
}

// SearchObjects returns a list of objects matching the search query
func SearchObjects(ctx context.Context, c *client.Client, query string) ([]ObjectSummary, error) {
	// Use genqlient-generated SearchAspects function with search filter
	filter := client.BuildSearchFilter(query)
	resp, err := client.SearchAspects(ctx, c.Genqlient(), filter)
	if err != nil {
		return nil, fmt.Errorf("failed to search objects: %w", err)
	}

	// Use helper to extract aspects into simplified data structure
	aspects := client.ExtractAspects(resp)

	objects := make([]ObjectSummary, 0, len(aspects))
	for _, aspect := range aspects {
		objectType := mapObjectType(aspect.Typename)
		objects = append(objects, ObjectSummary{
			ID:        aspect.ID,
			Name:      aspect.Name,
			Type:      objectType,
			Namespace: aspect.Namespace,
		})
	}

	return objects, nil
}

// mapObjectType converts GraphQL typename to display type
func mapObjectType(typename string) string {
	switch typename {
	case "AspectApp":
		return "App"
	case "AspectElement":
		return "Element"
	case "AspectTask":
		return "Task"
	default:
		return typename
	}
}

// checkAspectCommentsEnabled checks if comments are enabled on an aspect by querying for comments display block
func checkAspectCommentsEnabled(ctx context.Context, c *client.Client, aspectID string) bool {
	result, err := client.GetAspectCommentsStatus(ctx, c.Genqlient(), aspectID)
	if err != nil {
		logger.Debug("failed to check comments status", "aspectID", aspectID, "error", err)
		return false
	}

	if result.Organization.Aspect == nil {
		return false
	}

	// Get display blocks from the aspect (interface provides GetDisplayBlocks method)
	displayBlocks := (*result.Organization.Aspect).GetDisplayBlocks()
	for _, edge := range displayBlocks.Edges {
		if edge == nil {
			continue
		}
		typename := edge.Node.GetTypename()
		if typename != nil && *typename == "AspectCommentsDisplayBlock" {
			return true
		}
	}
	return false
}

// GetApp retrieves detailed information about an app and all its resources.
// If ec is non-nil, errors are recorded to the collector. In lenient mode,
// non-fatal errors are recorded as warnings and discovery continues.
// If ec is nil, the legacy behavior is used (fail on critical, warn on optional).
func GetApp(ctx context.Context, c *client.Client, appID string, ec ...*ErrorCollector) (*App, error) {
	app := &App{ID: appID}

	// Use the error collector if provided
	var collector *ErrorCollector
	if len(ec) > 0 && ec[0] != nil {
		collector = ec[0]
	}

	// recordError classifies and records an error. Returns true if the caller should abort.
	// In strict mode (collector present), fatal errors cause abort.
	// The phase and resource are used for the error report.
	recordError := func(phase, resource string, err error) bool {
		if err == nil {
			return false
		}
		severity := ClassifyError(err)
		if collector != nil {
			collector.Add(severity, phase, resource, err)
			// In strict mode, fatal errors cause abort
			return severity == SeverityFatal
		}
		// Legacy mode (no collector): log and continue
		if severity == SeverityFatal {
			logger.Error("failed to get "+resource, "appID", appID, "error", err)
		} else {
			logger.Warn("failed to get "+resource+" (non-fatal)", "appID", appID, "error", err)
		}
		return false
	}

	// Get basic app info - always fatal
	if err := getAppBasicInfo(ctx, c, app); err != nil {
		return nil, err
	}
	logger.Debug("got basic app info", "appID", app.ID, "name", app.Name, "namespace", app.Namespace)

	// Get fields - always fatal
	if err := getAppFields(ctx, c, app); err != nil {
		return nil, fmt.Errorf("failed to get fields: %w", err)
	}
	logger.Debug("got app fields", "fieldCount", len(app.Fields))

	// Get layouts - always fatal
	if err := getAppLayouts(ctx, c, app); err != nil {
		return nil, fmt.Errorf("failed to get layouts: %w", err)
	}
	logger.Debug("got app layouts", "layoutCount", len(app.Layouts))

	// Get automations - record error, may abort in strict mode
	// Pass error collector so GraphQL partial-data warnings are captured
	if err := getAppAutomations(ctx, c, app, collector); err != nil {
		if recordError("discovery", fmt.Sprintf("automations for %s", app.Name), err) {
			return app, fmt.Errorf("failed to get automations for %s: %w", app.Name, err)
		}
	}

	// Get agents - record error, may abort in strict mode
	if err := getAppAgents(ctx, c, app); err != nil {
		if recordError("discovery", fmt.Sprintf("agents for %s", app.Name), err) {
			return app, fmt.Errorf("failed to get agents for %s: %w", app.Name, err)
		}
	}

	// Get flows
	if err := getAppFlows(ctx, c, app); err != nil {
		if recordError("discovery", fmt.Sprintf("flows for %s", app.Name), err) {
			return app, fmt.Errorf("failed to get flows for %s: %w", app.Name, err)
		}
	}
	logger.Debug("checked for flows", "flowCount", len(app.Flows))

	// Get widgets
	if err := getAppWidgets(ctx, c, app); err != nil {
		if recordError("discovery", fmt.Sprintf("widgets for %s", app.Name), err) {
			return app, fmt.Errorf("failed to get widgets for %s: %w", app.Name, err)
		}
	}
	logger.Debug("checked for widgets", "widgetCount", len(app.Widgets))

	// Get views
	if err := getAppViews(ctx, c, app); err != nil {
		if recordError("discovery", fmt.Sprintf("views for %s", app.Name), err) {
			return app, fmt.Errorf("failed to get views for %s: %w", app.Name, err)
		}
	}
	logger.Debug("checked for views", "viewCount", len(app.Views))

	// Get AI File Readers
	if err := getAppAIFileReaders(ctx, c, app); err != nil {
		if recordError("discovery", fmt.Sprintf("AI file readers for %s", app.Name), err) {
			return app, fmt.Errorf("failed to get AI file readers for %s: %w", app.Name, err)
		}
	}

	// Get Approval Processes
	if err := getAppApprovalProcesses(ctx, c, app); err != nil {
		if recordError("discovery", fmt.Sprintf("approval processes for %s", app.Name), err) {
			return app, fmt.Errorf("failed to get approval processes for %s: %w", app.Name, err)
		}
	}

	// Get Relationships
	if err := getAppRelationships(ctx, c, app); err != nil {
		if recordError("discovery", fmt.Sprintf("relationships for %s", app.Name), err) {
			return app, fmt.Errorf("failed to get relationships for %s: %w", app.Name, err)
		}
	}

	// Discover related objects through relationships (recursive)
	if len(app.Relationships) > 0 {
		if err := discoverRelatedObjects(ctx, c, app, nil); err != nil {
			if recordError("discovery", fmt.Sprintf("related objects for %s", app.Name), err) {
				return app, fmt.Errorf("failed to discover related objects for %s: %w", app.Name, err)
			}
		}
	}

	// Discover related objects from automation tasks and agent tools
	if err := discoverTaskAndToolRelatedObjects(ctx, c, app); err != nil {
		if recordError("discovery", fmt.Sprintf("task/tool related objects for %s", app.Name), err) {
			return app, fmt.Errorf("failed to discover task/tool related objects for %s: %w", app.Name, err)
		}
	}

	// Get Access Policies
	if err := getAppAccessPolicies(ctx, c, app); err != nil {
		if recordError("discovery", fmt.Sprintf("access policies for %s", app.Name), err) {
			return app, fmt.Errorf("failed to get access policies for %s: %w", app.Name, err)
		}
	}
	logger.Debug("checked for access policies", "accessPolicyCount", len(app.AccessPolicies))

	// Check comments status
	app.CommentsEnabled = checkAspectCommentsEnabled(ctx, c, appID)
	logger.Debug("checked comments status", "appID", appID, "commentsEnabled", app.CommentsEnabled)

	// Get Roles
	if err := getAppRoles(ctx, c, app); err != nil {
		if recordError("discovery", fmt.Sprintf("roles for %s", app.Name), err) {
			return app, fmt.Errorf("failed to get roles for %s: %w", app.Name, err)
		}
	}
	logger.Debug("checked for roles", "roleCount", len(app.Roles))

	// Get Phone Services
	services, err := GetPhoneServicesForApp(ctx, c, app.ID)
	if err != nil {
		if recordError("discovery", fmt.Sprintf("phone services for %s", app.Name), err) {
			return app, fmt.Errorf("failed to get phone services for %s: %w", app.Name, err)
		}
	} else {
		app.PhoneServices = services
	}
	logger.Debug("checked for phone services", "phoneServiceCount", len(app.PhoneServices))

	// Get AI Search Tables
	searchTables, err := GetAISearchTablesForAspect(ctx, c, app.ID)
	if err != nil {
		if recordError("discovery", fmt.Sprintf("AI search tables for %s", app.Name), err) {
			return app, fmt.Errorf("failed to get AI search tables for %s: %w", app.Name, err)
		}
	} else {
		app.AISearchTables = searchTables
	}
	logger.Debug("checked for AI search tables", "aiSearchTableCount", len(app.AISearchTables))

	// Get Managed View Order (after views are loaded)
	if len(app.Views) > 0 {
		viewOrder, err := GetManagedViewOrderForAspect(ctx, c, app.ID)
		if err != nil {
			if recordError("discovery", fmt.Sprintf("managed view order for %s", app.Name), err) {
				return app, fmt.Errorf("failed to get managed view order for %s: %w", app.Name, err)
			}
		} else {
			app.ManagedViewOrder = viewOrder
		}
		logger.Debug("checked for managed view order", "hasManagedViewOrder", app.ManagedViewOrder != nil)
	}

	// Get Dashboards (and their widgets)
	dashboards, err := GetDashboardsForAspect(ctx, c, app.ID)
	if err != nil {
		if recordError("discovery", fmt.Sprintf("dashboards for %s", app.Name), err) {
			return app, fmt.Errorf("failed to get dashboards for %s: %w", app.Name, err)
		}
	} else {
		app.Dashboards = dashboards
	}
	widgetCount := 0
	for _, d := range app.Dashboards {
		widgetCount += len(d.Widgets)
	}
	logger.Debug("checked for dashboards", "dashboardCount", len(app.Dashboards), "widgetCount", widgetCount)

	// Discover aspects referenced by dashboard widgets (for beautification)
	if err := discoverDashboardWidgetAspects(ctx, c, app); err != nil {
		if recordError("discovery", fmt.Sprintf("dashboard widget aspects for %s", app.Name), err) {
			return app, fmt.Errorf("failed to discover dashboard widget aspects for %s: %w", app.Name, err)
		}
	}

	return app, nil
}

// GetAppParallel is a parallel version of GetApp that fetches app resources concurrently.
// It restructures the sequential calls into dependency phases that can run in parallel.
// The config parameter controls concurrency; if nil, DefaultMaxConcurrency is used.
func GetAppParallel(ctx context.Context, c *client.Client, appID string, config *ParallelConfig, ec ...*ErrorCollector) (*App, error) {
	app := &App{ID: appID}
	executor := NewParallelExecutor(config)

	var collector *ErrorCollector
	if len(ec) > 0 && ec[0] != nil {
		collector = ec[0]
	}

	// Helper to record and classify errors
	recordError := func(resource string, err error) bool {
		if err == nil {
			return false
		}
		severity := ClassifyError(err)
		if collector != nil {
			collector.Add(severity, "discovery", resource, err)
			return severity == SeverityFatal
		}
		if severity == SeverityFatal {
			logger.Error("failed to get "+resource, "appID", appID, "error", err)
		}
		return false
	}

	// ===========================================================================
	// Phase 0: Basic app info (must complete first - need app.Name for error msgs)
	// ===========================================================================
	if err := getAppBasicInfo(ctx, c, app); err != nil {
		return nil, err
	}
	logger.Debug("got basic app info", "appID", app.ID, "name", app.Name)

	// ===========================================================================
	// Phase 1: All independent resources in parallel (only need app.ID)
	// Combines previous Phase 1 + Phase 2 for maximum parallelism
	// ===========================================================================
	var (
		results        = &ParallelResults{}
		fieldsErr      error
		layoutsErr     error
		phoneServices  []PhoneService
		aiSearchTables []AISearchTable
		dashboards     []Dashboard
	)

	phase1Tasks := []func() error{
		// Critical resources (errors are fatal)
		func() error { fieldsErr = getAppFieldsNoValues(ctx, c, app); return nil },
		func() error { layoutsErr = getAppLayouts(ctx, c, app); return nil },

		// Non-critical resources (errors are recorded but don't abort)
		func() error { results.Add("flows", getAppFlows(ctx, c, app)); return nil },
		func() error { results.Add("widgets", getAppWidgets(ctx, c, app)); return nil },
		func() error { results.Add("views", getAppViews(ctx, c, app)); return nil },
		func() error { results.Add("AI file readers", getAppAIFileReaders(ctx, c, app)); return nil },
		func() error { results.Add("approval processes", getAppApprovalProcesses(ctx, c, app)); return nil },
		func() error { results.Add("access policies", getAppAccessPolicies(ctx, c, app)); return nil },
		func() error { results.Add("roles", getAppRoles(ctx, c, app)); return nil },
		func() error { results.Add("agents", getAppAgents(ctx, c, app)); return nil },
		func() error { results.Add("automations list", getAppAutomationsListOnly(ctx, c, app)); return nil },
		func() error { results.Add("relationships", getAppRelationships(ctx, c, app)); return nil },
		func() error {
			var err error
			phoneServices, err = GetPhoneServicesForApp(ctx, c, app.ID)
			results.Add("phone services", err)
			return nil
		},
		func() error {
			var err error
			aiSearchTables, err = GetAISearchTablesForAspect(ctx, c, app.ID)
			results.Add("AI search tables", err)
			return nil
		},
		func() error {
			var err error
			dashboards, err = GetDashboardsForAspect(ctx, c, app.ID)
			results.Add("dashboards", err)
			return nil
		},
		func() error {
			app.CommentsEnabled = checkAspectCommentsEnabled(ctx, c, appID)
			return nil
		},
	}
	_ = executor.RunParallel(ctx, phase1Tasks)

	// Check critical resources
	if fieldsErr != nil {
		return nil, fmt.Errorf("failed to get fields: %w", fieldsErr)
	}
	if layoutsErr != nil {
		return nil, fmt.Errorf("failed to get layouts: %w", layoutsErr)
	}

	// Store successful results
	app.PhoneServices = phoneServices
	app.AISearchTables = aiSearchTables
	app.Dashboards = dashboards

	// Record non-fatal errors
	for _, r := range results.Errors() {
		if recordError(fmt.Sprintf("%s for %s", r.name, app.Name), r.err) {
			return app, fmt.Errorf("failed to get %s: %w", r.name, r.err)
		}
	}

	logger.Debug("phase 1 complete",
		"fields", len(app.Fields), "layouts", len(app.Layouts),
		"flows", len(app.Flows), "views", len(app.Views),
		"agents", len(app.Agents), "automations", len(app.Automations),
	)

	// ===========================================================================
	// Phase 2: Field values (parallel fan-out for static picklists)
	// ===========================================================================
	if err := getAppFieldValuesParallel(ctx, c, app, executor); err != nil {
		logger.Warn("failed to fetch some field values", "error", err)
	}

	// ===========================================================================
	// Phase 3: Automation details + refs (two-stage parallel)
	// ===========================================================================
	if err := fetchAutomationDetailsParallel(ctx, c, app, executor, collector); err != nil {
		if recordError(fmt.Sprintf("automation details for %s", app.Name), err) {
			return app, err
		}
	}
	collectAiProviderConnectors(app)
	collectStoredFunctions(ctx, c, app)

	// ===========================================================================
	// Phase 4: Dependent fetches (need results from earlier phases)
	// ===========================================================================
	var (
		viewOrderErr, relatedErr, taskToolErr, dashboardAspectsErr error
		viewOrder                                                  *ManagedViewOrder
	)

	phase4Tasks := []func() error{}
	if len(app.Views) > 0 {
		phase4Tasks = append(phase4Tasks, func() error {
			viewOrder, viewOrderErr = GetManagedViewOrderForAspect(ctx, c, app.ID)
			return nil
		})
	}
	if len(app.Relationships) > 0 {
		phase4Tasks = append(phase4Tasks, func() error {
			relatedErr = discoverRelatedObjects(ctx, c, app, nil)
			return nil
		})
	}
	phase4Tasks = append(phase4Tasks, func() error {
		taskToolErr = discoverTaskAndToolRelatedObjects(ctx, c, app)
		return nil
	})
	if len(app.Dashboards) > 0 {
		phase4Tasks = append(phase4Tasks, func() error {
			dashboardAspectsErr = discoverDashboardWidgetAspects(ctx, c, app)
			return nil
		})
	}

	if len(phase4Tasks) > 0 {
		_ = executor.RunParallel(ctx, phase4Tasks)
	}

	// Store and record phase 4 results
	if viewOrderErr == nil {
		app.ManagedViewOrder = viewOrder
	}
	recordError(fmt.Sprintf("view order for %s", app.Name), viewOrderErr)
	recordError(fmt.Sprintf("related objects for %s", app.Name), relatedErr)
	recordError(fmt.Sprintf("task/tool objects for %s", app.Name), taskToolErr)
	recordError(fmt.Sprintf("dashboard aspects for %s", app.Name), dashboardAspectsErr)

	// Sort all slices for deterministic output
	app.SortForDeterministicOutput()

	return app, nil
}

// getAppFieldsNoValues is a variant of getAppFields that only fetches field metadata,
// not the picklist values. Values are fetched separately in parallel.
func getAppFieldsNoValues(ctx context.Context, c *client.Client, app *App) error {
	logger.Debug("fetching fields (no values)", "appId", app.ID)

	resp, err := client.GetAspectFieldsNoValues(ctx, c.Genqlient(), app.ID)
	if err != nil {
		return fmt.Errorf("failed to fetch fields: %w", err)
	}

	aspect := resp.Organization.Aspect
	if aspect == nil {
		return fmt.Errorf("fields query failed: aspect is nil (possible timeout)")
	}

	// Extract fields based on aspect type
	var fieldEdges []struct {
		Node client.FieldDetailsNoValues
	}
	switch a := (*aspect).(type) {
	case *client.GetAspectFieldsNoValuesOrganizationAspectAspectApp:
		for _, edge := range a.Fields.Edges {
			fieldEdges = append(fieldEdges, struct{ Node client.FieldDetailsNoValues }{Node: edge.Node})
		}
	case *client.GetAspectFieldsNoValuesOrganizationAspectAspectElement:
		for _, edge := range a.Fields.Edges {
			fieldEdges = append(fieldEdges, struct{ Node client.FieldDetailsNoValues }{Node: edge.Node})
		}
	case *client.GetAspectFieldsNoValuesOrganizationAspectAspectTask:
		for _, edge := range a.Fields.Edges {
			fieldEdges = append(fieldEdges, struct{ Node client.FieldDetailsNoValues }{Node: edge.Node})
		}
	default:
		return nil
	}

	app.Fields = make([]Field, 0, len(fieldEdges))
	for _, edge := range fieldEdges {
		node := edge.Node

		var semanticTags []string
		for _, tag := range node.GetSemanticTags() {
			semanticTags = append(semanticTags, string(tag))
		}

		typename := ""
		if node.GetTypename() != nil {
			typename = *node.GetTypename()
		}

		field := Field{
			ID:           node.GetId(),
			Name:         node.GetName(),
			Type:         mapFieldType(typename),
			Required:     node.GetRequired(),
			System:       node.GetSystem(),
			SemanticTags: semanticTags,
		}

		// Check if this is the stage field and capture lockOnCreate
		for _, tag := range semanticTags {
			if tag == "stage" {
				app.LockStages = node.GetLockOnCreate()
				break
			}
		}

		// Add calculation text for calculated fields
		if node.GetCalculation() != nil {
			field.Calculation = node.GetCalculation().Text
		}

		// Mark fields that need values fetched (static picklists)
		switch f := node.(type) {
		case *client.FieldDetailsNoValuesAspectPicklistField:
			if f.Config != nil {
				if _, ok := (*f.Config).(*client.FieldDetailsNoValuesConfigAspectStaticPicklistConfig); ok {
					field.NeedsValues = true
				}
			}
		case *client.FieldDetailsNoValuesAspectMultiPicklistField:
			if f.Config != nil {
				if _, ok := (*f.Config).(*client.FieldDetailsNoValuesConfigAspectStaticPicklistConfig); ok {
					field.NeedsValues = true
				}
			}
		}

		app.Fields = append(app.Fields, field)
	}

	logger.Debug("fields discovery (no values) complete", "totalFields", len(app.Fields))
	return nil
}

// getAppFieldValuesParallel fetches picklist values for all static picklist fields in parallel
func getAppFieldValuesParallel(ctx context.Context, c *client.Client, app *App, executor *ParallelExecutor) error {
	// Find all fields that need values
	var fieldsNeedingValues []int
	for i, field := range app.Fields {
		if field.NeedsValues {
			fieldsNeedingValues = append(fieldsNeedingValues, i)
		}
	}

	if len(fieldsNeedingValues) == 0 {
		return nil
	}

	logger.Debug("fetching picklist values in parallel", "count", len(fieldsNeedingValues))

	// Create tasks to fetch each field's values
	tasks := make([]func() error, len(fieldsNeedingValues))
	for i, fieldIndex := range fieldsNeedingValues {
		fieldIndex := fieldIndex // capture
		tasks[i] = func() error {
			field := &app.Fields[fieldIndex]
			valuesResp, err := client.GetFieldValues(ctx, c.Genqlient(), app.ID, field.ID)
			if err != nil {
				logger.Warn("failed to fetch values for field", "fieldId", field.ID, "error", err)
				return nil // Non-fatal, continue with other fields
			}

			if valuesResp.Organization.Aspect == nil {
				return nil
			}

			fieldData := (*valuesResp.Organization.Aspect).GetField()
			if fieldData == nil {
				return nil
			}

			switch f := fieldData.(type) {
			case *client.GetFieldValuesOrganizationAspectFieldAspectPicklistField:
				field.Options = make([]FieldOption, 0, len(f.Values.Edges))
				for _, edge := range f.Values.Edges {
					color := ""
					if edge.Node.Color != nil {
						color = *edge.Node.Color
					}
					var tags []string
					for _, tag := range edge.Node.Tags {
						tags = append(tags, string(tag))
					}
					field.Options = append(field.Options, FieldOption{
						ID:    edge.Node.Id,
						Label: edge.Node.Label,
						Color: color,
						Tags:  tags,
					})
				}
			case *client.GetFieldValuesOrganizationAspectFieldAspectMultiPicklistField:
				field.Options = make([]FieldOption, 0, len(f.Values.Edges))
				for _, edge := range f.Values.Edges {
					color := ""
					if edge.Node.Color != nil {
						color = *edge.Node.Color
					}
					var tags []string
					for _, tag := range edge.Node.Tags {
						tags = append(tags, string(tag))
					}
					field.Options = append(field.Options, FieldOption{
						ID:    edge.Node.Id,
						Label: edge.Node.Label,
						Color: color,
						Tags:  tags,
					})
				}
			}
			return nil
		}
	}

	// Run all value fetches in parallel
	_ = executor.RunParallel(ctx, tasks)

	logger.Debug("picklist values fetch complete", "fieldsProcessed", len(fieldsNeedingValues))
	return nil
}

// getAppAutomationsListOnly fetches just the automation list (without workflow details)
// This populates app.Automations with basic info, ready for parallel detail fetching
func getAppAutomationsListOnly(ctx context.Context, c *client.Client, app *App) error {
	logger.Debug("fetching automation list only", "appId", app.ID)

	// Filter out DISABLED automations at the API level
	listResp, err := client.GetAutomationsList(ctx, c.Genqlient(), app.ID, client.BuildNotDisabledFilter())
	if err != nil {
		return fmt.Errorf("failed to fetch automations list: %w", err)
	}

	aspect := listResp.Organization.Aspect
	if aspect == nil {
		return fmt.Errorf("automation query failed: aspect is nil (possible timeout)")
	}

	var automations []automationInfo
	switch a := (*aspect).(type) {
	case *client.GetAutomationsListOrganizationAspectAspectApp:
		if a.Automations != nil {
			for _, edge := range a.Automations.Edges {
				info := automationInfo{
					ID:     edge.Node.Id,
					Name:   edge.Node.Name,
					Status: string(edge.Node.Status),
				}
				if edge.Node.Current != nil {
					info.CurrentWorkflowID = edge.Node.Current.Id
				}
				if edge.Node.Draft != nil {
					info.DraftWorkflowID = edge.Node.Draft.Id
				}
				automations = append(automations, info)
			}
		}
	case *client.GetAutomationsListOrganizationAspectAspectElement:
		if a.Automations != nil {
			for _, edge := range a.Automations.Edges {
				info := automationInfo{
					ID:     edge.Node.Id,
					Name:   edge.Node.Name,
					Status: string(edge.Node.Status),
				}
				if edge.Node.Current != nil {
					info.CurrentWorkflowID = edge.Node.Current.Id
				}
				if edge.Node.Draft != nil {
					info.DraftWorkflowID = edge.Node.Draft.Id
				}
				automations = append(automations, info)
			}
		}
	default:
		return nil
	}

	// Populate app.Automations with basic info
	app.Automations = make([]Automation, 0, len(automations))
	for _, info := range automations {
		// Only export automations that have a published (current) workflow
		// This ensures we query consistently through automation.current
		if info.CurrentWorkflowID == "" {
			logger.Debug("skipping unpublished automation", "name", info.Name, "id", info.ID)
			continue
		}

		automation := Automation{
			ID:           info.ID,
			Name:         info.Name,
			Status:       info.Status,
			HasPublished: true,
			Terminal:     true, // Default: terminal (does NOT trigger other automations)
			WorkflowID:   info.CurrentWorkflowID,
		}
		if info.DraftWorkflowID != "" {
			automation.HasDraft = true
		}
		app.Automations = append(app.Automations, automation)
	}

	logger.Debug("automation list fetched", "count", len(app.Automations))
	return nil
}

// fetchAutomationDetailsParallel fetches workflow details and refs for all automations in parallel
func fetchAutomationDetailsParallel(ctx context.Context, c *client.Client, app *App, executor *ParallelExecutor, collector *ErrorCollector) error {
	if len(app.Automations) == 0 {
		return nil
	}

	logger.Debug("fetching automation details in parallel", "count", len(app.Automations))

	// Stage 1: Fetch workflow details for all automations in parallel
	detailTasks := make([]func() error, len(app.Automations))
	for i := range app.Automations {
		i := i // capture
		automation := &app.Automations[i]
		detailTasks[i] = func() error {
			if automation.WorkflowID == "" {
				return nil
			}

			workflowResp, err := client.GetWorkflowDetailsForDiscovery(ctx, c.Genqlient(), app.ID, automation.WorkflowID)
			if err != nil {
				if collector != nil {
					collector.Add(SeverityFatal, "workflow_query", fmt.Sprintf("workflow %s for automation %s", automation.WorkflowID, automation.Name), err)
				}
				logger.Warn("failed to fetch workflow details", "error", err, "automationId", automation.ID, "workflowId", automation.WorkflowID)
				return nil // Don't fail the whole batch
			}

			if workflowResp.Organization.Aspect != nil {
				workflow := (*workflowResp.Organization.Aspect).GetWorkflow()
				if workflow != nil {
					// Extract terminal flag (non-terminal = triggers other automations)
					automation.Terminal = workflow.GetTerminal()

					automation.Triggers = make([]Trigger, 0, len(workflow.Triggers))
					for _, trigger := range workflow.Triggers {
						automation.Triggers = append(automation.Triggers, extractTriggerFromDiscovery(trigger))
					}

					automation.Tasks = make([]Task, 0, len(workflow.Tasks))
					for _, task := range workflow.Tasks {
						automation.Tasks = append(automation.Tasks, extractTaskFromDiscovery(task, automation.WorkflowID))
					}
				}
			}

			logger.Debug("fetched automation details",
				"id", automation.ID,
				"name", automation.Name,
				"triggerCount", len(automation.Triggers),
				"taskCount", len(automation.Tasks),
			)
			return nil
		}
	}

	_ = executor.RunParallel(ctx, detailTasks)

	// Stage 1.5: Fetch full task details (RawData) for HCL generation
	// This uses a dynamic query with all task type fragments
	fullDetailTasks := make([]func() error, 0, len(app.Automations))
	tasksFragment := client.BuildTasksQueryFragment()
	triggersFragment := buildTriggersQueryFragment()

	for i := range app.Automations {
		i := i // capture
		automation := &app.Automations[i]

		// Skip automations without workflows or unpublished/inactive ones
		if automation.WorkflowID == "" || !automation.HasPublished || automation.Status != "ACTIVE" {
			continue
		}

		fullDetailTasks = append(fullDetailTasks, func() error {
			query := fmt.Sprintf(`
				query GetFullWorkflowDetails($aspectId: ID!, $workflowId: ID!) {
					organization {
						aspect(id: $aspectId) {
							id
							workflow(id: $workflowId) {
								id
								outputs {
									id
									name
									value {
										taskReference { name }
										triggerReference { name }
										variableReference { name }
									}
								}
								triggers {
									id
									__typename
									%s
								}
								%s
							}
						}
					}
				}
			`, triggersFragment, tasksFragment)

			variables := map[string]interface{}{
				"aspectId":   app.ID,
				"workflowId": automation.WorkflowID,
			}

			var result map[string]interface{}
			if err := c.ExecuteInto(ctx, query, variables, &result); err != nil {
				logger.Warn("failed to fetch full workflow details",
					"automationId", automation.ID,
					"workflowId", automation.WorkflowID,
					"error", err)
				return nil // Don't fail the whole batch
			}

			workflowData := extractWorkflowData(result)
			if workflowData == nil {
				return nil
			}

			// Update trigger RawData
			if triggers, ok := workflowData["triggers"].([]interface{}); ok {
				triggerByID := make(map[string]*Trigger)
				for j := range automation.Triggers {
					triggerByID[automation.Triggers[j].ID] = &automation.Triggers[j]
				}
				for _, triggerInterface := range triggers {
					if triggerData, ok := triggerInterface.(map[string]interface{}); ok {
						triggerID := getString(triggerData, "id")
						if trigger, exists := triggerByID[triggerID]; exists {
							trigger.RawData = triggerData
						}
					}
				}
			}

			// Update task RawData
			if tasks, ok := workflowData["tasks"].([]interface{}); ok {
				taskByID := make(map[string]*Task)
				for j := range automation.Tasks {
					taskByID[automation.Tasks[j].ID] = &automation.Tasks[j]
				}
				for _, taskInterface := range tasks {
					if taskData, ok := taskInterface.(map[string]interface{}); ok {
						taskID := getString(taskData, "id")
						if task, exists := taskByID[taskID]; exists {
							task.RawData = taskData
							// Re-determine task type now that we have RawData
							// (needed for variable/update_variable distinction)
							if task.Type == "variable" && isUpdateVariableTask(taskData) {
								task.Type = "update_variable"
							}
							// Extract operator children (switch cases, fork/join branches)
							if childrenArr, ok := taskData["children"].([]interface{}); ok {
								for _, childInterface := range childrenArr {
									if childData, ok := childInterface.(map[string]interface{}); ok {
										task.Children = append(task.Children, childData)
									}
								}
							}
						}
					}
				}
			}

			// Parse workflow outputs (for on-demand automations)
			if outputs, ok := workflowData["outputs"].([]interface{}); ok {
				automation.Outputs = make([]WorkflowOutput, 0, len(outputs))
				for _, outputInterface := range outputs {
					if outputData, ok := outputInterface.(map[string]interface{}); ok {
						name := getString(outputData, "name")
						var valueRef map[string]interface{}
						if v := outputData["value"]; v != nil {
							valueRef, _ = v.(map[string]interface{})
						}
						if name != "" {
							automation.Outputs = append(automation.Outputs, WorkflowOutput{
								Name:  name,
								Value: valueRef,
							})
						}
					}
				}
			}

			logger.Debug("fetched full task details",
				"automationId", automation.ID,
				"taskCount", len(automation.Tasks),
				"outputCount", len(automation.Outputs))
			return nil
		})
	}

	if len(fullDetailTasks) > 0 {
		_ = executor.RunParallel(ctx, fullDetailTasks)
		logger.Debug("full task details fetch complete", "automations", len(fullDetailTasks))
	}

	// Stage 2: Fetch refs for all automations in parallel
	// First, collect all leaf tasks across all automations
	type leafTaskRef struct {
		automationIndex int
		leafTask        *Task
	}
	var allLeafTasks []leafTaskRef

	// Create a mutex per automation to protect FieldRefs map writes
	// (multiple leaf tasks from the same automation may run in parallel)
	automationMutexes := make([]sync.Mutex, len(app.Automations))

	for i := range app.Automations {
		automation := &app.Automations[i]
		if automation.WorkflowID == "" {
			continue
		}

		// Initialize FieldRefs maps
		for j := range automation.Triggers {
			if automation.Triggers[j].FieldRefs == nil {
				automation.Triggers[j].FieldRefs = make(map[string]string)
			}
		}
		for j := range automation.Tasks {
			if automation.Tasks[j].FieldRefs == nil {
				automation.Tasks[j].FieldRefs = make(map[string]string)
			}
		}

		leafTasks := findAllLeafTasks(automation)
		if len(leafTasks) == 0 {
			// No tasks - query at workflow start for trigger refs only
			allLeafTasks = append(allLeafTasks, leafTaskRef{
				automationIndex: i,
				leafTask:        nil, // nil indicates query at workflow start
			})
		} else {
			for _, leaf := range leafTasks {
				allLeafTasks = append(allLeafTasks, leafTaskRef{
					automationIndex: i,
					leafTask:        leaf,
				})
			}
		}
	}

	if len(allLeafTasks) == 0 {
		return nil
	}

	logger.Debug("fetching refs for leaf tasks in parallel", "totalLeafTasks", len(allLeafTasks))

	// Create tasks for parallel refs fetching
	refsTasks := make([]func() error, len(allLeafTasks))
	for i, ref := range allLeafTasks {
		ref := ref // capture
		refsTasks[i] = func() error {
			automation := &app.Automations[ref.automationIndex]

			var parentTaskID *string
			if ref.leafTask != nil {
				parentTaskID = &ref.leafTask.ID
			}

			allRefs, err := queryAvailableReferences(ctx, c, app.ID, automation.ID, parentTaskID)
			if err != nil {
				// Non-fatal - continue with other refs
				return nil
			}

			// Build task lookup map
			taskByID := make(map[string]*Task)
			for j := range automation.Tasks {
				taskByID[automation.Tasks[j].ID] = &automation.Tasks[j]
			}

			// Lock this automation's mutex to protect FieldRefs map writes
			automationMutexes[ref.automationIndex].Lock()
			processAvailableRefs(allRefs, automation, taskByID)
			automationMutexes[ref.automationIndex].Unlock()
			return nil
		}
	}

	_ = executor.RunParallel(ctx, refsTasks)

	logger.Debug("automation details and refs fetch complete", "automations", len(app.Automations))
	return nil
}

// getAppBasicInfo retrieves basic app information using genqlient
func getAppBasicInfo(ctx context.Context, c *client.Client, app *App) error {
	result, err := client.GetAppBasicInfo(ctx, c.Genqlient(), app.ID)
	if err != nil {
		return fmt.Errorf("failed to get app info: %w", err)
	}

	if result.Organization.Aspect == nil {
		return fmt.Errorf("app not found: %s", app.ID)
	}

	// Type assert to get the AspectApp type
	aspectApp, ok := (*result.Organization.Aspect).(*client.GetAppBasicInfoOrganizationAspectAspectApp)
	if !ok {
		return fmt.Errorf("aspect %s is not an app", app.ID)
	}

	app.Name = aspectApp.Name
	app.Namespace = aspectApp.Namespace

	// Set category info if available
	app.CategoryID = aspectApp.Category.Id
	app.CategoryName = aspectApp.Category.Name

	// Set cloudLink info if available
	if aspectApp.CloudLink != nil {
		cloudLink := *aspectApp.CloudLink
		app.CloudLinkID = cloudLink.GetId()
		app.CloudLinkName = cloudLink.GetName()
	}

	return nil
}

// getAppFields retrieves all fields for an app using a three-phase approach to avoid timeouts.
// Phase 1: Get all fields with config but NO values (lightweight)
// Phase 2: Fetch values for static picklist fields only
// Phase 3: Fetch aspect names for dynamic picklist configs
func getAppFields(ctx context.Context, c *client.Client, app *App) error {
	logger.Debug("fetching fields (phase 1 - lightweight)", "appId", app.ID)

	// Phase 1: Get all fields with config but NO values
	resp, err := client.GetAspectFieldsNoValues(ctx, c.Genqlient(), app.ID)
	if err != nil {
		return fmt.Errorf("failed to fetch fields: %w", err)
	}

	aspect := resp.Organization.Aspect
	if aspect == nil {
		return fmt.Errorf("fields query failed: aspect is nil (possible timeout)")
	}

	// Extract fields based on aspect type
	var fieldEdges []struct {
		Node client.FieldDetailsNoValues
	}
	switch a := (*aspect).(type) {
	case *client.GetAspectFieldsNoValuesOrganizationAspectAspectApp:
		for _, edge := range a.Fields.Edges {
			fieldEdges = append(fieldEdges, struct{ Node client.FieldDetailsNoValues }{Node: edge.Node})
		}
	case *client.GetAspectFieldsNoValuesOrganizationAspectAspectElement:
		for _, edge := range a.Fields.Edges {
			fieldEdges = append(fieldEdges, struct{ Node client.FieldDetailsNoValues }{Node: edge.Node})
		}
	case *client.GetAspectFieldsNoValuesOrganizationAspectAspectTask:
		for _, edge := range a.Fields.Edges {
			fieldEdges = append(fieldEdges, struct{ Node client.FieldDetailsNoValues }{Node: edge.Node})
		}
	default:
		return nil
	}

	// Track which fields need Phase 2 (static picklist values) and Phase 3 (dynamic aspect names)
	type fieldInfo struct {
		index            int
		isStaticPicklist bool
	}
	var staticPicklistFields []fieldInfo
	dynamicAspectIDs := make(map[string]bool)

	app.Fields = make([]Field, 0, len(fieldEdges))
	for _, edge := range fieldEdges {
		node := edge.Node

		// Convert semantic tags
		var semanticTags []string
		for _, tag := range node.GetSemanticTags() {
			semanticTags = append(semanticTags, string(tag))
		}

		typename := ""
		if node.GetTypename() != nil {
			typename = *node.GetTypename()
		}

		field := Field{
			ID:           node.GetId(),
			Name:         node.GetName(),
			Type:         mapFieldType(typename),
			Required:     node.GetRequired(),
			System:       node.GetSystem(),
			SemanticTags: semanticTags,
		}

		// Check if this is the stage field and capture lockOnCreate
		for _, tag := range semanticTags {
			if tag == "stage" {
				app.LockStages = node.GetLockOnCreate()
				break
			}
		}

		// Add calculation text for calculated fields
		if node.GetCalculation() != nil {
			field.Calculation = node.GetCalculation().Text
		}

		// Check config type for picklist fields
		fieldIndex := len(app.Fields)
		switch f := node.(type) {
		case *client.FieldDetailsNoValuesAspectPicklistField:
			if f.Config != nil {
				switch cfg := (*f.Config).(type) {
				case *client.FieldDetailsNoValuesConfigAspectStaticPicklistConfig:
					// Static picklist - need to fetch values in Phase 2
					staticPicklistFields = append(staticPicklistFields, fieldInfo{
						index:            fieldIndex,
						isStaticPicklist: true,
					})
					_ = cfg // avoid unused variable
				case *client.FieldDetailsNoValuesConfigAspectDynamicPicklistConfig:
					// Dynamic picklist - need to fetch aspect name in Phase 3
					relatedAspectID := cfg.RelatedAspect.GetId()
					if relatedAspectID != "" {
						dynamicAspectIDs[relatedAspectID] = true
					}
				}
			}
		case *client.FieldDetailsNoValuesAspectMultiPicklistField:
			if f.Config != nil {
				switch cfg := (*f.Config).(type) {
				case *client.FieldDetailsNoValuesConfigAspectStaticPicklistConfig:
					// Static multi-picklist - need to fetch values in Phase 2
					staticPicklistFields = append(staticPicklistFields, fieldInfo{
						index:            fieldIndex,
						isStaticPicklist: true,
					})
					_ = cfg // avoid unused variable
				case *client.FieldDetailsNoValuesConfigAspectDynamicPicklistConfig:
					// Dynamic multi-picklist - need to fetch aspect name in Phase 3
					relatedAspectID := cfg.RelatedAspect.GetId()
					if relatedAspectID != "" {
						dynamicAspectIDs[relatedAspectID] = true
					}
				}
			}
		}

		app.Fields = append(app.Fields, field)
	}

	// Phase 2: Fetch values for static picklist fields only
	if len(staticPicklistFields) > 0 {
		logger.Debug("fetching picklist values (phase 2)", "count", len(staticPicklistFields))
		for _, info := range staticPicklistFields {
			field := &app.Fields[info.index]
			valuesResp, err := client.GetFieldValues(ctx, c.Genqlient(), app.ID, field.ID)
			if err != nil {
				logger.Warn("failed to fetch values for field", "fieldId", field.ID, "error", err)
				continue
			}

			if valuesResp.Organization.Aspect == nil {
				continue
			}

			// Extract values from the response
			fieldData := (*valuesResp.Organization.Aspect).GetField()
			if fieldData == nil {
				continue
			}

			switch f := fieldData.(type) {
			case *client.GetFieldValuesOrganizationAspectFieldAspectPicklistField:
				field.Options = make([]FieldOption, 0, len(f.Values.Edges))
				for _, edge := range f.Values.Edges {
					color := ""
					if edge.Node.Color != nil {
						color = *edge.Node.Color
					}
					var tags []string
					for _, tag := range edge.Node.Tags {
						tags = append(tags, string(tag))
					}
					field.Options = append(field.Options, FieldOption{
						ID:    edge.Node.Id,
						Label: edge.Node.Label,
						Color: color,
						Tags:  tags,
					})
				}
			case *client.GetFieldValuesOrganizationAspectFieldAspectMultiPicklistField:
				field.Options = make([]FieldOption, 0, len(f.Values.Edges))
				for _, edge := range f.Values.Edges {
					color := ""
					if edge.Node.Color != nil {
						color = *edge.Node.Color
					}
					var tags []string
					for _, tag := range edge.Node.Tags {
						tags = append(tags, string(tag))
					}
					field.Options = append(field.Options, FieldOption{
						ID:    edge.Node.Id,
						Label: edge.Node.Label,
						Color: color,
						Tags:  tags,
					})
				}
			}
		}
	}

	// Phase 3: Fetch aspect names for dynamic picklist configs (if needed in the future)
	// Currently, aspect names are not used during export, but this is where we'd fetch them
	if len(dynamicAspectIDs) > 0 {
		logger.Debug("skipping aspect name resolution (phase 3)", "count", len(dynamicAspectIDs),
			"reason", "not required for export")
	}

	logger.Debug("fields discovery complete", "totalFields", len(app.Fields),
		"staticPicklistsWithValues", len(staticPicklistFields))

	return nil
}

// getAppLayouts retrieves all layouts (stages) for an app using genqlient
func getAppLayouts(ctx context.Context, c *client.Client, app *App) error {
	result, err := client.GetAspectStagesWithDisplayBlocks(ctx, c.Genqlient(), app.ID)
	if err != nil {
		return err
	}

	if result.Organization.Aspect == nil {
		return nil
	}

	// Type assert to get the AspectApp type
	aspectApp, ok := (*result.Organization.Aspect).(*client.GetAspectStagesWithDisplayBlocksOrganizationAspectAspectApp)
	if !ok {
		return nil // Not an app, no stages
	}

	// Each stage represents one layout
	app.Layouts = make([]Layout, 0, len(aspectApp.Stages))
	logger.Debug("retrieved stages from API", "stageCount", len(aspectApp.Stages))

	for _, stage := range aspectApp.Stages {
		// Only include stages that have display blocks (i.e., have a layout configured)
		if len(stage.DisplayBlocks.Edges) > 0 {
			layout := Layout{
				ID:         stage.Id,
				Name:       stage.Name,
				IsInitiate: stage.Name == "Initiate", // Mark Initiate layout - it needs empty stage_id
			}

			// Fetch full layout details via GetStageLayout for complete display block data
			// (field IDs, icon, color, displayOrder, sideNavItem, displayLocation)
			stageResult, stageErr := client.GetStageLayout(ctx, c.Genqlient(), app.ID, stage.Id)
			if stageErr != nil {
				logger.Warn("failed to fetch detailed layout for stage, falling back to minimal data",
					"stageId", stage.Id, "error", stageErr.Error())
				// Fall back to minimal display block data from the initial query
				layout.DisplayBlocks = buildMinimalDisplayBlocks(stage.DisplayBlocks.Edges, stage.Id, app.ID)
				app.Layouts = append(app.Layouts, layout)
				continue
			}

			stageAspect := stageResult.Organization.Aspect
			if stageAspect == nil {
				layout.DisplayBlocks = buildMinimalDisplayBlocks(stage.DisplayBlocks.Edges, stage.Id, app.ID)
				app.Layouts = append(app.Layouts, layout)
				continue
			}

			stageDetail := (*stageAspect).GetStage()
			layout.DisplayBlocks = buildDetailedDisplayBlocks(stageDetail.DisplayBlocks.Edges, stage.Id, app.ID)

			app.Layouts = append(app.Layouts, layout)
		}
	}

	return nil
}

// buildMinimalDisplayBlocks creates DisplayBlock entries from the initial discovery query edges
// (fallback when the detailed GetStageLayout query fails).
func buildMinimalDisplayBlocks(edges []*client.GetAspectStagesWithDisplayBlocksOrganizationAspectAspectAppStagesAspectStageDisplayBlocksAspectDisplayBlockConnectionEdgesAspectDisplayBlockEdge, stageID, aspectID string) []DisplayBlock {
	blocks := make([]DisplayBlock, 0, len(edges))
	for _, blockEdge := range edges {
		if blockEdge == nil {
			continue
		}
		blockName := ""
		if groupBlock, ok := blockEdge.Node.(*client.GetAspectStagesWithDisplayBlocksOrganizationAspectAspectAppStagesAspectStageDisplayBlocksAspectDisplayBlockConnectionEdgesAspectDisplayBlockEdgeNodeAspectGroupDisplayBlock); ok {
			blockName = groupBlock.Name
		}
		typename := ""
		if blockEdge.Node.GetTypename() != nil {
			typename = *blockEdge.Node.GetTypename()
		}
		blocks = append(blocks, DisplayBlock{
			ID:       blockEdge.Node.GetId(),
			Type:     mapDisplayBlockType(typename),
			Name:     blockName,
			StageID:  stageID,
			AspectID: aspectID,
		})
	}
	return blocks
}

// buildDetailedDisplayBlocks creates DisplayBlock entries from the GetStageLayout response,
// including full details like field IDs, icon, color, displayOrder, etc.
func buildDetailedDisplayBlocks(edges []*client.GetStageLayoutOrganizationAspectStageDisplayBlocksAspectDisplayBlockConnectionEdgesAspectDisplayBlockEdge, stageID, aspectID string) []DisplayBlock {
	blocks := make([]DisplayBlock, 0, len(edges))
	for _, edge := range edges {
		if edge == nil {
			continue
		}
		node := edge.Node
		if node == nil {
			continue
		}

		typename := ""
		if node.GetTypename() != nil {
			typename = *node.GetTypename()
		}

		block := DisplayBlock{
			ID:              node.GetId(),
			Type:            mapDisplayBlockType(typename),
			StageID:         stageID,
			AspectID:        aspectID,
			DisplayOrder:    node.GetDisplayOrder(),
			SideNavItem:     node.GetSideNavItem() == nil || *node.GetSideNavItem(),
			DisplayLocation: displayLocationString(node.GetDisplayLocation()),
		}

		// Extract group-specific fields
		if groupBlock, ok := node.(*client.GetStageLayoutOrganizationAspectStageDisplayBlocksAspectDisplayBlockConnectionEdgesAspectDisplayBlockEdgeNodeAspectGroupDisplayBlock); ok {
			block.Name = groupBlock.Name
			block.Icon = groupBlock.Icon
			block.Color = groupBlock.Color

			// Extract field/widget IDs from child blocks
			for _, childEdge := range groupBlock.Blocks.Edges {
				if childEdge == nil {
					continue
				}
				switch cb := childEdge.Node.(type) {
				case *client.GetStageLayoutOrganizationAspectStageDisplayBlocksAspectDisplayBlockConnectionEdgesAspectDisplayBlockEdgeNodeAspectGroupDisplayBlockBlocksAspectDisplayBlockConnectionEdgesAspectDisplayBlockEdgeNodeAspectFieldDisplayBlock:
					fieldID := cb.Field.GetId()
					if fieldID != "" {
						block.FieldIDs = append(block.FieldIDs, fieldID)
					}
				case *client.GetStageLayoutOrganizationAspectStageDisplayBlocksAspectDisplayBlockConnectionEdgesAspectDisplayBlockEdgeNodeAspectGroupDisplayBlockBlocksAspectDisplayBlockConnectionEdgesAspectDisplayBlockEdgeNodeAspectWidgetDisplayBlock:
					widgetID := cb.Widget.GetId()
					if widgetID != "" {
						block.FieldIDs = append(block.FieldIDs, widgetID)
					}
				}
			}
		}

		blocks = append(blocks, block)
	}
	return blocks
}

// displayLocationString converts an AspectDisplayBlockLocation pointer to a string.
func displayLocationString(loc *client.AspectDisplayBlockLocation) string {
	if loc == nil {
		return "CENTER"
	}
	return string(*loc)
}

// automationInfo is a simplified struct for automation metadata from the list query
type automationInfo struct {
	ID                string
	Name              string
	Status            string
	CurrentWorkflowID string
	DraftWorkflowID   string
}

// getAppAutomations retrieves all automations and their triggers/tasks with full task details.
// Uses a two-phase approach to avoid timeouts on large apps:
// 1. Fetch lightweight automation list with just workflow IDs
// 2. Fetch each workflow's details individually
// If ec is provided, GraphQL partial-data warnings are recorded so callers know about
// fields that may be null/broken due to server-side data issues.
func getAppAutomations(ctx context.Context, c *client.Client, app *App, ec ...*ErrorCollector) error {
	var collector *ErrorCollector
	if len(ec) > 0 && ec[0] != nil {
		collector = ec[0]
	}

	// Phase 1: Get lightweight automation list using genqlient
	// Filter out DISABLED automations at the API level
	logger.Debug("fetching automation list (phase 1)", "appId", app.ID)
	listResp, err := client.GetAutomationsList(ctx, c.Genqlient(), app.ID, client.BuildNotDisabledFilter())
	if err != nil {
		return fmt.Errorf("failed to fetch automations list: %w", err)
	}

	aspect := listResp.Organization.Aspect
	if aspect == nil {
		return fmt.Errorf("automation query failed: aspect is nil (possible timeout)")
	}

	// Extract automations based on aspect type into a common format
	var automations []automationInfo
	switch a := (*aspect).(type) {
	case *client.GetAutomationsListOrganizationAspectAspectApp:
		if a.Automations != nil {
			for _, edge := range a.Automations.Edges {
				info := automationInfo{
					ID:     edge.Node.Id,
					Name:   edge.Node.Name,
					Status: string(edge.Node.Status),
				}
				if edge.Node.Current != nil {
					info.CurrentWorkflowID = edge.Node.Current.Id
				}
				if edge.Node.Draft != nil {
					info.DraftWorkflowID = edge.Node.Draft.Id
				}
				automations = append(automations, info)
			}
		}
	case *client.GetAutomationsListOrganizationAspectAspectElement:
		if a.Automations != nil {
			for _, edge := range a.Automations.Edges {
				info := automationInfo{
					ID:     edge.Node.Id,
					Name:   edge.Node.Name,
					Status: string(edge.Node.Status),
				}
				if edge.Node.Current != nil {
					info.CurrentWorkflowID = edge.Node.Current.Id
				}
				if edge.Node.Draft != nil {
					info.DraftWorkflowID = edge.Node.Draft.Id
				}
				automations = append(automations, info)
			}
		}
	default:
		// Aspect type doesn't support automations
		return nil
	}

	logger.Debug("found automations", "count", len(automations))

	// Phase 2: Fetch each workflow's details individually using lightweight genqlient query
	// This uses DiscoveryTaskFields/DiscoveryTriggerFields which only include basic info
	// and aspect references, avoiding the heavy query that times out on large workflows.
	app.Automations = make([]Automation, 0, len(automations))

	for _, info := range automations {
		automation := Automation{
			ID:       info.ID,
			Name:     info.Name,
			Status:   info.Status,
			Terminal: true, // Default: terminal (does NOT trigger other automations)
		}

		// Determine which workflow to fetch (prefer current/published, fall back to draft)
		var workflowID string
		if info.CurrentWorkflowID != "" {
			automation.HasPublished = true
			automation.WorkflowID = info.CurrentWorkflowID
			workflowID = info.CurrentWorkflowID
		} else if info.DraftWorkflowID != "" {
			automation.HasDraft = true
			automation.WorkflowID = info.DraftWorkflowID
			workflowID = info.DraftWorkflowID
		}

		if info.DraftWorkflowID != "" {
			automation.HasDraft = true
		}

		// Fetch workflow details if we have a workflow ID
		if workflowID != "" {
			logger.Debug("fetching workflow details (phase 2)", "automationId", automation.ID, "workflowId", workflowID)

			// Use lightweight genqlient query instead of heavy dynamic query
			workflowResp, err := client.GetWorkflowDetailsForDiscovery(ctx, c.Genqlient(), app.ID, workflowID)
			if err != nil {
				// Log the error but continue with other automations
				if collector != nil {
					collector.Add(SeverityFatal, "workflow_query", fmt.Sprintf("workflow %s for automation %s", workflowID, automation.Name), err)
				}
				logger.Warn("failed to fetch workflow details", "error", err, "automationId", automation.ID, "workflowId", workflowID)
				// Add automation without workflow details
				app.Automations = append(app.Automations, automation)
				continue
			}

			// Extract triggers and tasks from genqlient response
			if workflowResp.Organization.Aspect != nil {
				workflow := (*workflowResp.Organization.Aspect).GetWorkflow()
				if workflow != nil {
					// Extract terminal flag (non-terminal = triggers other automations)
					automation.Terminal = workflow.GetTerminal()

					// Extract triggers
					automation.Triggers = make([]Trigger, 0, len(workflow.Triggers))
					for _, trigger := range workflow.Triggers {
						automation.Triggers = append(automation.Triggers, extractTriggerFromDiscovery(trigger))
					}

					// Extract tasks
					automation.Tasks = make([]Task, 0, len(workflow.Tasks))
					for _, task := range workflow.Tasks {
						automation.Tasks = append(automation.Tasks, extractTaskFromDiscovery(task, automation.WorkflowID))
					}
				}
			}
		}

		app.Automations = append(app.Automations, automation)

		logger.Debug("discovered automation",
			"id", automation.ID,
			"name", automation.Name,
			"status", automation.Status,
			"hasPublished", automation.HasPublished,
			"hasDraft", automation.HasDraft,
			"workflowID", automation.WorkflowID,
			"triggerCount", len(automation.Triggers),
			"taskCount", len(automation.Tasks),
		)
	}

	logger.Debug("automation discovery complete", "totalAutomations", len(app.Automations))

	// Fetch full task details (RawData) for HCL generation
	// This is needed because the lightweight discovery doesn't fetch task-specific attributes
	if err := fetchFullTaskDetails(ctx, c, app); err != nil {
		logger.Warn("failed to fetch full task details", "error", err)
		// Continue anyway - some tasks may have partial data
	}

	// Fetch refs for triggers and tasks (for beautification)
	for i := range app.Automations {
		automation := &app.Automations[i]
		fetchAutomationRefs(ctx, c, app.ID, automation)
	}

	// Collect unique AI provider connectors from all tasks
	collectAiProviderConnectors(app)
	collectStoredFunctions(ctx, c, app)

	return nil
}

// fetchFullTaskDetails fetches full task details (RawData) for all tasks in all automations.
// This is called after the lightweight discovery to populate task-specific attributes
// needed for HCL generation.
func fetchFullTaskDetails(ctx context.Context, c *client.Client, app *App) error {
	// Build query with all task type fragments
	tasksFragment := client.BuildTasksQueryFragment()
	triggersFragment := buildTriggersQueryFragment()

	for i := range app.Automations {
		automation := &app.Automations[i]

		// Skip automations without workflows
		if automation.WorkflowID == "" {
			continue
		}

		// Skip unpublished/inactive automations
		if !automation.HasPublished || automation.Status != "ACTIVE" {
			continue
		}

		// Build dynamic query for this workflow's full task details
		query := fmt.Sprintf(`
			query GetFullWorkflowDetails($aspectId: ID!, $workflowId: ID!) {
				organization {
					aspect(id: $aspectId) {
						id
						workflow(id: $workflowId) {
							id
							triggers {
								id
								__typename
								%s
							}
							%s
						}
					}
				}
			}
		`, triggersFragment, tasksFragment)

		variables := map[string]interface{}{
			"aspectId":   app.ID,
			"workflowId": automation.WorkflowID,
		}

		var result map[string]interface{}
		if err := c.ExecuteInto(ctx, query, variables, &result); err != nil {
			logger.Warn("failed to fetch full workflow details",
				"automationId", automation.ID,
				"workflowId", automation.WorkflowID,
				"error", err)
			continue
		}

		// Extract workflow data from result
		workflowData := extractWorkflowData(result)
		if workflowData == nil {
			continue
		}

		// Update trigger RawData
		if triggers, ok := workflowData["triggers"].([]interface{}); ok {
			triggerByID := make(map[string]*Trigger)
			for j := range automation.Triggers {
				triggerByID[automation.Triggers[j].ID] = &automation.Triggers[j]
			}

			for _, triggerInterface := range triggers {
				if triggerData, ok := triggerInterface.(map[string]interface{}); ok {
					triggerID := getString(triggerData, "id")
					if trigger, exists := triggerByID[triggerID]; exists {
						trigger.RawData = triggerData
					}
				}
			}
		}

		// Update task RawData
		if tasks, ok := workflowData["tasks"].([]interface{}); ok {
			taskByID := make(map[string]*Task)
			for j := range automation.Tasks {
				taskByID[automation.Tasks[j].ID] = &automation.Tasks[j]
			}

			for _, taskInterface := range tasks {
				if taskData, ok := taskInterface.(map[string]interface{}); ok {
					taskID := getString(taskData, "id")
					if task, exists := taskByID[taskID]; exists {
						task.RawData = taskData
						// Re-determine task type now that we have RawData
						// (needed for variable/update_variable distinction)
						if task.Type == "variable" && isUpdateVariableTask(taskData) {
							task.Type = "update_variable"
						}
						// Extract operator children (switch cases, fork/join branches)
						if childrenArr, ok := taskData["children"].([]interface{}); ok {
							for _, childInterface := range childrenArr {
								if childData, ok := childInterface.(map[string]interface{}); ok {
									task.Children = append(task.Children, childData)
								}
							}
						}
					}
				}
			}
		}

		logger.Debug("fetched full task details",
			"automationId", automation.ID,
			"taskCount", len(automation.Tasks))
	}

	return nil
}

// buildTriggersQueryFragment builds the triggers fragment with type-specific fields.
// Uses only fields that are available in all schema versions to avoid validation errors.
func buildTriggersQueryFragment() string {
	return `
		... on WorkflowRecordCreateTrigger {
			filter
			changedFields { id name }
		}
		... on WorkflowRecordUpdateTrigger {
			filter
			changedFields { id name }
		}
		... on WorkflowOnDemandTrigger {
			showTriggeredBy
			parameters {
				id
				name
				fieldType
				required
				defaultValue
				multiple
			}
		}
		... on WorkflowWebhookTrigger {
			authenticated
			url
		}
		... on WorkflowDatamineTrigger {
			datamine { id name }
		}
	`
}

// extractWorkflowData extracts the workflow data from the GraphQL response
func extractWorkflowData(result map[string]interface{}) map[string]interface{} {
	org, ok := result["organization"].(map[string]interface{})
	if !ok {
		return nil
	}
	aspect, ok := org["aspect"].(map[string]interface{})
	if !ok {
		return nil
	}
	workflow, ok := aspect["workflow"].(map[string]interface{})
	if !ok {
		return nil
	}
	return workflow
}

// extractTrigger extracts a Trigger from the raw GraphQL response, including all type-specific fields
func extractTrigger(triggerData map[string]interface{}) Trigger {
	triggerType := mapTriggerType(getString(triggerData, "__typename"))

	t := Trigger{
		ID:      getString(triggerData, "id"),
		Type:    triggerType,
		Name:    triggerType, // Use type as name since triggers don't have explicit names
		RawData: triggerData, // Store the full raw data for type-specific extraction
	}

	// Extract datamine ID for datamine triggers
	if datamine, ok := triggerData["datamine"].(map[string]interface{}); ok && datamine != nil {
		t.DatamineID = getString(datamine, "id")
	}

	return t
}

// extractTask extracts a Task from the raw GraphQL response, including all type-specific fields
func extractTask(taskData map[string]interface{}, workflowID string) Task {
	// Extract parent_id from previous task
	parentID := ""
	if previous, ok := taskData["previous"].(map[string]interface{}); ok && previous != nil {
		parentID = getString(previous, "id")
	}

	// Extract object_id from aspect
	objectID := ""
	if aspect, ok := taskData["aspect"].(map[string]interface{}); ok && aspect != nil {
		objectID = getString(aspect, "id")
	}

	// Extract related_object_id from relatedAspect (for find_related_records task)
	relatedObjectID := ""
	if relatedAspect, ok := taskData["relatedAspect"].(map[string]interface{}); ok && relatedAspect != nil {
		relatedObjectID = getString(relatedAspect, "id")
	}

	// Extract document_model_id from documentModel (for ai_file_read and bulk_excel tasks)
	documentModelID := ""
	if documentModel, ok := taskData["documentModel"].(map[string]interface{}); ok && documentModel != nil {
		documentModelID = getString(documentModel, "id")
	}

	// Extract dynamic_category_source aspect ID from categorySource (for ai_classify task)
	dynamicCategoryAspectID := ""
	if categorySource, ok := taskData["categorySource"].(map[string]interface{}); ok && categorySource != nil {
		if typename := getString(categorySource, "__typename"); typename == "AiClassifyCategoryDynamicSource" {
			if aspect, ok := categorySource["aspect"].(map[string]interface{}); ok && aspect != nil {
				dynamicCategoryAspectID = getString(aspect, "id")
			}
		}
	}

	// Extract AI provider connector info for AI tasks (ai_file_read, ai_classify, ai_summarize, etc.)
	var aiProviderConnectorID, aiProviderConnectorModelName, aiProviderConnectorProvider string
	if aiConnector, ok := taskData["aiProviderConnector"].(map[string]interface{}); ok && aiConnector != nil {
		aiProviderConnectorID = getString(aiConnector, "id")
		if model, ok := aiConnector["model"].(map[string]interface{}); ok && model != nil {
			aiProviderConnectorModelName = getString(model, "name")
		}
		if provider, ok := aiConnector["provider"].(map[string]interface{}); ok && provider != nil {
			aiProviderConnectorProvider = getString(provider, "name")
		}
	}

	// Extract stored function info for procedure tasks
	var storedFunctionID, storedFunctionName, cloudLinkID string
	if storedFunction, ok := taskData["storedFunction"].(map[string]interface{}); ok && storedFunction != nil {
		storedFunctionID = getString(storedFunction, "id")
		storedFunctionName = getString(storedFunction, "displayName")
		if storedFunctionName == "" {
			storedFunctionName = getString(storedFunction, "name")
		}
	}
	// Extract cloudLinkId for procedure tasks
	cloudLinkID = getString(taskData, "cloudLinkId")

	// Extract field IDs from workflowFields
	var fieldIDs []string
	if wfFields, ok := taskData["workflowFields"].([]interface{}); ok {
		fieldIDs = make([]string, 0, len(wfFields))
		for _, wfInterface := range wfFields {
			if wf, ok := wfInterface.(map[string]interface{}); ok {
				if field, ok := wf["field"].(map[string]interface{}); ok {
					fieldIDs = append(fieldIDs, getString(field, "id"))
				}
			}
		}
	}

	return Task{
		ID:                           getString(taskData, "id"),
		Type:                         determineTaskType(taskData),
		Name:                         getString(taskData, "name"),
		WorkflowID:                   workflowID,
		ParentID:                     parentID,
		ObjectID:                     objectID,
		RelatedObjectID:              relatedObjectID,
		DocumentModelID:              documentModelID,
		DynamicCategoryAspectID:      dynamicCategoryAspectID,
		FieldIDs:                     fieldIDs,
		RawData:                      taskData, // Store the full raw data for type-specific extraction
		AiProviderConnectorID:        aiProviderConnectorID,
		AiProviderConnectorModelName: aiProviderConnectorModelName,
		AiProviderConnectorProvider:  aiProviderConnectorProvider,
		StoredFunctionID:             storedFunctionID,
		StoredFunctionName:           storedFunctionName,
		CloudLinkID:                  cloudLinkID,
	}
}

// getString safely extracts a string from a map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// collectAiProviderConnectors extracts unique AI provider connectors from all automation tasks
// and populates app.DiscoveredAiProviderConnectors for data source generation during export.
func collectAiProviderConnectors(app *App) {
	seen := make(map[string]bool)
	for _, automation := range app.Automations {
		for _, task := range automation.Tasks {
			if task.AiProviderConnectorID != "" && !seen[task.AiProviderConnectorID] {
				seen[task.AiProviderConnectorID] = true
				app.DiscoveredAiProviderConnectors = append(app.DiscoveredAiProviderConnectors, AiProviderConnector{
					ID:           task.AiProviderConnectorID,
					ModelName:    task.AiProviderConnectorModelName,
					ProviderName: task.AiProviderConnectorProvider,
				})
			}
		}
	}

	if len(app.DiscoveredAiProviderConnectors) > 0 {
		logger.Debug("collected AI provider connectors",
			"count", len(app.DiscoveredAiProviderConnectors),
		)
	}
}

// collectStoredFunctions extracts unique stored functions from all automation procedure tasks
// and populates app.DiscoveredStoredFunctions for data source generation during export.
func collectStoredFunctions(ctx context.Context, c *client.Client, app *App) {
	seen := make(map[string]bool)
	for _, automation := range app.Automations {
		for _, task := range automation.Tasks {
			if task.Type == "procedure" && task.StoredFunctionID != "" && !seen[task.StoredFunctionID] {
				seen[task.StoredFunctionID] = true

				// Try to fetch the stored function details
				sf, err := GetStoredFunctionByID(ctx, c, task.StoredFunctionID)
				if err != nil {
					// Fall back to the info we already have from the task
					logger.Debug("could not fetch stored function details, using task data",
						"storedFunctionID", task.StoredFunctionID,
						"error", err,
					)
					// Use task data - we have the ID and name already
					app.DiscoveredStoredFunctions = append(app.DiscoveredStoredFunctions, &StoredFunction{
						ID:          task.StoredFunctionID,
						DisplayName: task.StoredFunctionName,
						CloudLinkID: task.CloudLinkID,
					})
				} else {
					app.DiscoveredStoredFunctions = append(app.DiscoveredStoredFunctions, sf)
				}
			}
		}
	}

	if len(app.DiscoveredStoredFunctions) > 0 {
		logger.Debug("collected stored functions",
			"count", len(app.DiscoveredStoredFunctions),
		)
	}
}

// fetchAutomationRefs fetches available references for triggers and tasks in an automation.
// This populates the FieldRefs map which is used during beautification to convert
// raw UUIDs like "trigger.record.{uuid}" to "${trigger.refs[\"Field Name\"]}".
//
// We query from the LAST task in the chain to get ALL available references at once,
// since the API returns cumulative refs (trigger + all previous task outputs).
func fetchAutomationRefs(ctx context.Context, c *client.Client, appID string, automation *Automation) {
	if automation.WorkflowID == "" {
		return
	}

	// Initialize FieldRefs maps
	for i := range automation.Triggers {
		if automation.Triggers[i].FieldRefs == nil {
			automation.Triggers[i].FieldRefs = make(map[string]string)
		}
	}
	for i := range automation.Tasks {
		if automation.Tasks[i].FieldRefs == nil {
			automation.Tasks[i].FieldRefs = make(map[string]string)
		}
	}

	// Build a map of task ID to task pointer for quick lookup
	taskByID := make(map[string]*Task)
	for i := range automation.Tasks {
		taskByID[automation.Tasks[i].ID] = &automation.Tasks[i]
	}

	// Find ALL leaf tasks (end of each branch in switch/for-each logic)
	// Each leaf node has refs for its specific path through the automation
	leafTasks := findAllLeafTasks(automation)

	if len(leafTasks) == 0 {
		// No tasks - query at workflow start for trigger refs only
		allRefs, err := queryAvailableReferences(ctx, c, appID, automation.ID, nil)
		if err != nil {
			return
		}
		processAvailableRefs(allRefs, automation, taskByID)
		return
	}

	// First query refs from workflow START (null parent) to get trigger refs available BEFORE any task
	// This is important because the leaf task query returns refs available AFTER the task (its outputs)
	startRefs, err := queryAvailableReferences(ctx, c, appID, automation.ID, nil)
	if err == nil {
		processAvailableRefs(startRefs, automation, taskByID)
	}

	// Then query refs from EACH leaf task and merge results
	// This captures refs from all branches (switch cases, for-each loops, etc.)
	for _, leafTask := range leafTasks {
		allRefs, err := queryAvailableReferences(ctx, c, appID, automation.ID, &leafTask.ID)
		if err != nil {
			// Non-fatal - continue with other branches
			continue
		}
		processAvailableRefs(allRefs, automation, taskByID)
	}
}

// processAvailableRefs processes a set of available refs and populates FieldRefs maps
func processAvailableRefs(allRefs []AvailableRef, automation *Automation, taskByID map[string]*Task) {
	for _, ref := range allRefs {
		for _, prop := range ref.Properties {
			if prop.Deprecated {
				continue
			}

			// ALWAYS store by reference ID in the first trigger's FieldRefs
			// This ensures we can look up ANY ref by ID, even if it doesn't have
			// a specific reference type (triggerReference, taskReference, etc.)
			if prop.Reference.ID != "" && len(automation.Triggers) > 0 {
				trigger := &automation.Triggers[0]
				trigger.FieldRefs["id:"+prop.Reference.ID] = prop.Name
			}

			// Handle trigger references - store in first trigger's FieldRefs
			if prop.Reference.TriggerReference != nil && len(automation.Triggers) > 0 {
				trigger := &automation.Triggers[0]
				name := prop.Reference.TriggerReference.Name
				// Store both the full path and just the UUID portion
				trigger.FieldRefs[name] = prop.Name
				// Also extract just the UUID if it's a record field
				if strings.HasPrefix(name, "record.") {
					fieldID := strings.TrimPrefix(name, "record.")
					// Handle nested properties like "record.{uuid}.email"
					parts := strings.SplitN(fieldID, ".", 2)
					if len(parts) > 0 {
						trigger.FieldRefs[parts[0]] = prop.Name
					}
				}
				// Also store by reference ID for lookup when we only have {id, valid}
				if prop.Reference.ID != "" {
					trigger.FieldRefs["id:"+prop.Reference.ID] = prop.Name
				}
			}

			// Handle task references - store in the specific task's FieldRefs
			if prop.Reference.TaskReference != nil {
				taskID := prop.Reference.TaskReference.TaskID
				refName := prop.Reference.TaskReference.Name
				if task, ok := taskByID[taskID]; ok {
					task.FieldRefs[refName] = prop.Name
					// Also store by reference ID
					if prop.Reference.ID != "" {
						task.FieldRefs["id:"+prop.Reference.ID] = prop.Name
					}
				}
			}

			// Handle forEach references - store in the forEach task's FieldRefs
			if prop.Reference.ForEachReference != nil {
				forEachTaskID := prop.Reference.ForEachReference.ForEachTaskID
				refName := prop.Reference.ForEachReference.Name
				if task, ok := taskByID[forEachTaskID]; ok {
					task.FieldRefs[refName] = prop.Name
					// Also store by reference ID
					if prop.Reference.ID != "" {
						task.FieldRefs["id:"+prop.Reference.ID] = prop.Name
					}
				}
			}

			// Handle variable references - store by variable name for lookup
			// Variable refs point to a variable defined by a variable task
			if prop.Reference.VariableReference != nil {
				varName := prop.Reference.VariableReference.Name
				// Store in all variable tasks - we'll match by variable name later
				for _, task := range taskByID {
					if task.Type == "variable" {
						task.FieldRefs["variable."+varName] = prop.Name
						// Also store by reference ID
						if prop.Reference.ID != "" {
							task.FieldRefs["id:"+prop.Reference.ID] = prop.Name
						}
					}
				}
			}
		}
	}
}

// findAllLeafTasks finds ALL leaf tasks in the automation - one for each branch.
// With switch cases and for-each loops, there can be multiple branches, each with
// its own leaf node. Querying refs from each leaf captures all available refs.
func findAllLeafTasks(automation *Automation) []*Task {
	if len(automation.Tasks) == 0 {
		return nil
	}

	// Build set of task IDs that are referenced as parents
	parentIDs := make(map[string]bool)
	for _, task := range automation.Tasks {
		if task.ParentID != "" {
			parentIDs[task.ParentID] = true
		}
	}

	// Find ALL tasks that are not parents of any other task (leaf nodes)
	var leafTasks []*Task
	for i := range automation.Tasks {
		if !parentIDs[automation.Tasks[i].ID] {
			leafTasks = append(leafTasks, &automation.Tasks[i])
		}
	}

	return leafTasks
}

// AvailableRefProperty represents a property in available references
type AvailableRefProperty struct {
	FieldType  string
	Name       string
	Multiple   bool
	Deprecated bool
	Reference  struct {
		ID               string
		Value            string
		TriggerReference *struct {
			Name string
		}
		TaskReference *struct {
			Name   string
			TaskID string
		}
		ForEachReference *struct {
			Name          string
			ForEachTaskID string
		}
		VariableReference *struct {
			VariableID string
			Name       string
			Type       string
		}
	}
}

// AvailableRef represents a section of available references
type AvailableRef struct {
	Name       string
	Properties []AvailableRefProperty
}

// queryAvailableReferences queries the published workflow for available references at a given point.
// Uses the lightweight query through automation.current to ensure consistent querying.
func queryAvailableReferences(ctx context.Context, c *client.Client, appID, automationID string, parentTaskID *string) ([]AvailableRef, error) {
	// Use lightweight genqlient query through automation.current
	isOperatorChild := false
	resp, err := client.GetCurrentWorkflowAvailableReferencesLightweight(
		ctx,
		c.Genqlient(),
		appID,
		automationID,
		parentTaskID,
		&isOperatorChild,
	)
	if err != nil {
		return nil, err
	}

	if resp.Organization.Aspect == nil {
		return nil, fmt.Errorf("aspect not found")
	}

	// Get automation from aspect
	automation := (*resp.Organization.Aspect).GetAutomation()
	if automation == nil {
		return nil, fmt.Errorf("automation not found")
	}

	// Get current (published) workflow
	current := automation.GetCurrent()
	if current == nil {
		return nil, fmt.Errorf("published workflow not found")
	}

	// Convert genqlient response to our struct format
	availableRefs := current.GetAvailableReferences()
	refs := make([]AvailableRef, 0, len(availableRefs))

	// Debug: log raw API response
	logger.Debug("availableReferences API response", "groupCount", len(availableRefs), "automationID", automationID)
	for _, ar := range availableRefs {
		logger.Debug("  ref group", "name", ar.GetName(), "propCount", len(ar.GetProperties()))
		for _, p := range ar.GetProperties() {
			reference := p.GetReference()
			logger.Debug("    property", "name", p.GetName(), "refID", reference.GetId())
		}
	}

	for _, ar := range availableRefs {
		ref := AvailableRef{
			Name:       ar.GetName(),
			Properties: make([]AvailableRefProperty, 0, len(ar.GetProperties())),
		}
		for _, p := range ar.GetProperties() {
			// Lightweight query doesn't include FieldType or Multiple
			prop := AvailableRefProperty{
				FieldType:  "", // Not fetched in lightweight query
				Name:       p.GetName(),
				Multiple:   false, // Not fetched in lightweight query
				Deprecated: p.GetDeprecated(),
			}
			reference := p.GetReference()
			prop.Reference.ID = reference.GetId()
			// Value is *map[string]interface{} in genqlient, but we only need the ID/TriggerRef/TaskRef
			if reference.GetTriggerReference() != nil {
				prop.Reference.TriggerReference = &struct{ Name string }{
					Name: reference.GetTriggerReference().GetName(),
				}
			}
			if reference.GetTaskReference() != nil {
				taskRef := reference.GetTaskReference()
				taskID := ""
				if taskRef.GetTask() != nil {
					taskID = (*taskRef.GetTask()).GetId()
				}
				prop.Reference.TaskReference = &struct {
					Name   string
					TaskID string
				}{
					Name:   taskRef.GetName(),
					TaskID: taskID,
				}
			}
			if reference.GetForEachReference() != nil {
				forEachRef := reference.GetForEachReference()
				forEachTaskID := ""
				if forEachRef.GetForEachTask() != nil {
					forEachTaskID = forEachRef.GetForEachTask().GetId()
				}
				name := ""
				if forEachRef.GetName() != nil {
					name = *forEachRef.GetName()
				}
				prop.Reference.ForEachReference = &struct {
					Name          string
					ForEachTaskID string
				}{
					Name:          name,
					ForEachTaskID: forEachTaskID,
				}
			}
			if reference.GetVariableReference() != nil {
				varRef := reference.GetVariableReference()
				varID := ""
				if varRef.GetVariableId() != nil {
					varID = *varRef.GetVariableId()
				}
				varName := ""
				if varRef.GetName() != nil {
					varName = *varRef.GetName()
				}
				varType := ""
				if varRef.GetType() != nil {
					varType = string(*varRef.GetType())
				}
				prop.Reference.VariableReference = &struct {
					VariableID string
					Name       string
					Type       string
				}{
					VariableID: varID,
					Name:       varName,
					Type:       varType,
				}
			}
			ref.Properties = append(ref.Properties, prop)
		}
		refs = append(refs, ref)
	}

	return refs, nil
}

// getAppAgents retrieves all agents for an app using genqlient
func getAppAgents(ctx context.Context, c *client.Client, app *App) error {
	result, err := client.GetAspectAgents(ctx, c.Genqlient(), app.ID)
	if err != nil {
		return err
	}

	if result.Organization.Aspect == nil {
		return nil
	}

	// Type switch to get AspectApp (only apps have agents)
	aspectApp, ok := (*result.Organization.Aspect).(*client.GetAspectAgentsOrganizationAspectAspectApp)
	if !ok {
		return nil // Not an app, no agents
	}

	app.Agents = make([]Agent, 0, len(aspectApp.AgentsV2.Edges))
	for _, edge := range aspectApp.AgentsV2.Edges {
		agent := extractAgentFromNode(edge.Node)
		// A2A skills live on the AgentCard (AgentElementum-only) and need
		// their own query — the AgentsV2 fragment doesn't include them.
		if a2a, err := GetA2ASkillsForAgent(ctx, c, agent.ID); err != nil {
			logger.Warn("failed to fetch a2a skills", "agent", agent.Name, "error", err)
		} else {
			agent.A2ASkills = a2a
		}
		app.Agents = append(app.Agents, agent)
	}

	logger.Debug("discovered agents", "app", app.Name, "count", len(app.Agents))

	// Collect unique AI provider connectors from agents
	collectAgentAiProviderConnectors(app)

	// Discover agentic skills + their tools. Skills are an independent
	// resource tree referenced from agents via skill_ids — they aren't
	// returned by the AgentsV2 fragment and need their own fetch.
	if skills, err := GetSkillsForApp(ctx, c, app.ID); err != nil {
		logger.Warn("failed to fetch agentic skills", "app", app.Name, "error", err)
	} else {
		app.Skills = skills
	}

	return nil
}

// collectAgentAiProviderConnectors extracts unique AI provider connectors from all agents
// and adds them to app.DiscoveredAiProviderConnectors for data source generation during export.
func collectAgentAiProviderConnectors(app *App) {
	// Build a set of already-seen connector IDs (from automation tasks)
	seen := make(map[string]bool)
	for _, conn := range app.DiscoveredAiProviderConnectors {
		seen[conn.ID] = true
	}

	// Collect from agents
	for _, agent := range app.Agents {
		if agent.AiProviderConnectorID != "" && !seen[agent.AiProviderConnectorID] {
			seen[agent.AiProviderConnectorID] = true
			app.DiscoveredAiProviderConnectors = append(app.DiscoveredAiProviderConnectors, AiProviderConnector{
				ID:           agent.AiProviderConnectorID,
				ModelName:    agent.AiProviderConnectorModelName,
				ProviderName: agent.AiProviderConnectorProvider,
			})
		}
	}

	if len(app.Agents) > 0 {
		logger.Debug("collected AI provider connectors from agents",
			"agentCount", len(app.Agents),
			"totalConnectors", len(app.DiscoveredAiProviderConnectors),
		)
	}
}

// CollectSearchTableAiProviderConnectors extracts unique AI provider connectors from all AI search tables
// on the app and discovered elements, and adds them to app.DiscoveredAiProviderConnectors.
// This MUST be called AFTER discovered elements are merged into the app (during export, not discovery)
// because discovered elements' AI search tables are only available after unified discovery completes.
func CollectSearchTableAiProviderConnectors(app *App) {
	if app == nil {
		return
	}

	// Build a set of already-seen connector IDs
	seen := make(map[string]bool)
	for _, conn := range app.DiscoveredAiProviderConnectors {
		seen[conn.ID] = true
	}

	addedCount := 0

	// Collect from app-level AI search tables
	for _, searchTable := range app.AISearchTables {
		if searchTable.AIProviderConnectorID != "" && !seen[searchTable.AIProviderConnectorID] {
			seen[searchTable.AIProviderConnectorID] = true
			app.DiscoveredAiProviderConnectors = append(app.DiscoveredAiProviderConnectors, AiProviderConnector{
				ID:        searchTable.AIProviderConnectorID,
				ModelName: searchTable.AIProviderConnectorName,
			})
			addedCount++
		}
	}

	// Collect from discovered elements' AI search tables
	for _, element := range app.DiscoveredElements {
		for _, searchTable := range element.AISearchTables {
			if searchTable.AIProviderConnectorID != "" && !seen[searchTable.AIProviderConnectorID] {
				seen[searchTable.AIProviderConnectorID] = true
				app.DiscoveredAiProviderConnectors = append(app.DiscoveredAiProviderConnectors, AiProviderConnector{
					ID:        searchTable.AIProviderConnectorID,
					ModelName: searchTable.AIProviderConnectorName,
				})
				addedCount++
			}
		}
	}

	if addedCount > 0 {
		logger.Debug("collected AI provider connectors from AI search tables",
			"addedCount", addedCount,
			"totalConnectors", len(app.DiscoveredAiProviderConnectors),
		)
	}
}

// extractAgentFromNode extracts an Agent from the genqlient agent node interface
func extractAgentFromNode(node client.GetAspectAgentsOrganizationAspectAspectAppAgentsV2AgentV2ConnectionEdgesAgentV2EdgeNodeAgentV2) Agent {
	agent := Agent{
		ID:   node.GetId(),
		Name: node.GetName(),
	}

	if typename := node.GetTypename(); typename != nil {
		agent.Type = *typename
	}

	// Type switch to extract agent-type-specific fields
	switch a := node.(type) {
	case *client.GetAspectAgentsOrganizationAspectAspectAppAgentsV2AgentV2ConnectionEdgesAgentV2EdgeNodeAgentElementum:
		agent.Description = a.Description
		agent.Instructions = a.Instructions
		if a.FirstMessage != nil {
			agent.FirstMessage = *a.FirstMessage
		}
		extractAiProviderConnectorFromElementum(a.AiProviderConnector, &agent)
		agent.Tools = extractToolsFromElementum(a.Tools)
		agent.StartingActions = extractStartingActionsFromElementum(a.StartingActions)

	case *client.GetAspectAgentsOrganizationAspectAspectAppAgentsV2AgentV2ConnectionEdgesAgentV2EdgeNodeAgentSnowflake:
		agent.Description = a.Description
		agent.Instructions = a.Instructions
		if a.FirstMessage != nil {
			agent.FirstMessage = *a.FirstMessage
		}
		extractAiProviderConnectorFromSnowflake(a.AiProviderConnector, &agent)
		// AgentSnowflakeConfig is an embedded struct, not a pointer
		agent.SnowflakeDatabase = a.AgentSnowflakeConfig.Database
		agent.SnowflakeSchema = a.AgentSnowflakeConfig.Schema
		agent.SnowflakeName = a.AgentSnowflakeConfig.Name
		agent.Tools = extractToolsFromSnowflake(a.Tools)
		agent.StartingActions = extractStartingActionsFromSnowflake(a.StartingActions)

	case *client.GetAspectAgentsOrganizationAspectAspectAppAgentsV2AgentV2ConnectionEdgesAgentV2EdgeNodeAgentBedrock:
		agent.Description = a.Description
		agent.Instructions = a.Instructions
		if a.FirstMessage != nil {
			agent.FirstMessage = *a.FirstMessage
		}
		extractAiProviderConnectorFromBedrock(a.AiProviderConnector, &agent)
		// AgentBedrockConfig is an embedded struct, not a pointer
		agent.BedrockAgentArn = a.AgentBedrockConfig.AgentArn
		agent.Tools = extractToolsFromBedrock(a.Tools)
		agent.StartingActions = extractStartingActionsFromBedrock(a.StartingActions)

	case *client.GetAspectAgentsOrganizationAspectAspectAppAgentsV2AgentV2ConnectionEdgesAgentV2EdgeNodeAgentBrowserUse:
		agent.Description = a.Description
		agent.Instructions = a.Instructions
		if a.FirstMessage != nil {
			agent.FirstMessage = *a.FirstMessage
		}
		extractAiProviderConnectorFromBrowserUse(a.AiProviderConnector, &agent)
		if a.BrowserAgentInstructions != nil {
			agent.BrowserAgentInstructions = *a.BrowserAgentInstructions
		}
		agent.Tools = extractToolsFromBrowserUse(a.Tools)
		agent.StartingActions = extractStartingActionsFromBrowserUse(a.StartingActions)
	}

	return agent
}

// AI Provider Connector extraction helpers for each agent type
// Note: These functions take pointers to interface types, so we must dereference before calling methods
func extractAiProviderConnectorFromElementum(conn *client.GetAspectAgentsOrganizationAspectAspectAppAgentsV2AgentV2ConnectionEdgesAgentV2EdgeNodeAgentElementumAiProviderConnector, agent *Agent) {
	if conn == nil {
		return
	}
	c := *conn // Dereference pointer to interface
	agent.AiProviderConnectorID = c.GetId()
	agent.AiProviderConnectorModelName = c.GetModel().Name
	agent.AiProviderConnectorProvider = c.GetProvider().GetName()
}

func extractAiProviderConnectorFromSnowflake(conn *client.GetAspectAgentsOrganizationAspectAspectAppAgentsV2AgentV2ConnectionEdgesAgentV2EdgeNodeAgentSnowflakeAiProviderConnector, agent *Agent) {
	if conn == nil {
		return
	}
	c := *conn // Dereference pointer to interface
	agent.AiProviderConnectorID = c.GetId()
	agent.AiProviderConnectorModelName = c.GetModel().Name
	agent.AiProviderConnectorProvider = c.GetProvider().GetName()
}

func extractAiProviderConnectorFromBedrock(conn *client.GetAspectAgentsOrganizationAspectAspectAppAgentsV2AgentV2ConnectionEdgesAgentV2EdgeNodeAgentBedrockAiProviderConnector, agent *Agent) {
	if conn == nil {
		return
	}
	c := *conn // Dereference pointer to interface
	agent.AiProviderConnectorID = c.GetId()
	agent.AiProviderConnectorModelName = c.GetModel().Name
	agent.AiProviderConnectorProvider = c.GetProvider().GetName()
}

func extractAiProviderConnectorFromBrowserUse(conn *client.GetAspectAgentsOrganizationAspectAspectAppAgentsV2AgentV2ConnectionEdgesAgentV2EdgeNodeAgentBrowserUseAiProviderConnector, agent *Agent) {
	if conn == nil {
		return
	}
	c := *conn // Dereference pointer to interface
	agent.AiProviderConnectorID = c.GetId()
	agent.AiProviderConnectorModelName = c.GetModel().Name
	agent.AiProviderConnectorProvider = c.GetProvider().GetName()
}

// Tool extraction helpers for each agent type
func extractToolsFromElementum(toolsConn client.GetAspectAgentsOrganizationAspectAspectAppAgentsV2AgentV2ConnectionEdgesAgentV2EdgeNodeAgentElementumToolsAgentToolConnection) []AgentTool {
	tools := make([]AgentTool, 0, len(toolsConn.Edges))
	for _, edge := range toolsConn.Edges {
		tool := extractToolFromAgentToolInfo(edge.Node)
		tools = append(tools, tool)
	}
	return tools
}

func extractToolsFromSnowflake(toolsConn client.GetAspectAgentsOrganizationAspectAspectAppAgentsV2AgentV2ConnectionEdgesAgentV2EdgeNodeAgentSnowflakeToolsAgentToolConnection) []AgentTool {
	tools := make([]AgentTool, 0, len(toolsConn.Edges))
	for _, edge := range toolsConn.Edges {
		tool := extractToolFromAgentToolInfo(edge.Node)
		tools = append(tools, tool)
	}
	return tools
}

func extractToolsFromBedrock(toolsConn client.GetAspectAgentsOrganizationAspectAspectAppAgentsV2AgentV2ConnectionEdgesAgentV2EdgeNodeAgentBedrockToolsAgentToolConnection) []AgentTool {
	tools := make([]AgentTool, 0, len(toolsConn.Edges))
	for _, edge := range toolsConn.Edges {
		tool := extractToolFromAgentToolInfo(edge.Node)
		tools = append(tools, tool)
	}
	return tools
}

func extractToolsFromBrowserUse(toolsConn client.GetAspectAgentsOrganizationAspectAspectAppAgentsV2AgentV2ConnectionEdgesAgentV2EdgeNodeAgentBrowserUseToolsAgentToolConnection) []AgentTool {
	tools := make([]AgentTool, 0, len(toolsConn.Edges))
	for _, edge := range toolsConn.Edges {
		tool := extractToolFromAgentToolInfo(edge.Node)
		tools = append(tools, tool)
	}
	return tools
}

// Starting action extraction helpers for each agent type
func extractStartingActionsFromElementum(actions []client.GetAspectAgentsOrganizationAspectAspectAppAgentsV2AgentV2ConnectionEdgesAgentV2EdgeNodeAgentElementumStartingActionsStartingAction) []StartingAction {
	result := make([]StartingAction, 0, len(actions))
	for i := range actions {
		result = append(result, extractStartingActionFromInfo(&actions[i]))
	}
	return result
}

func extractStartingActionsFromSnowflake(actions []client.GetAspectAgentsOrganizationAspectAspectAppAgentsV2AgentV2ConnectionEdgesAgentV2EdgeNodeAgentSnowflakeStartingActionsStartingAction) []StartingAction {
	result := make([]StartingAction, 0, len(actions))
	for i := range actions {
		result = append(result, extractStartingActionFromInfo(&actions[i]))
	}
	return result
}

func extractStartingActionsFromBedrock(actions []client.GetAspectAgentsOrganizationAspectAspectAppAgentsV2AgentV2ConnectionEdgesAgentV2EdgeNodeAgentBedrockStartingActionsStartingAction) []StartingAction {
	result := make([]StartingAction, 0, len(actions))
	for i := range actions {
		result = append(result, extractStartingActionFromInfo(&actions[i]))
	}
	return result
}

func extractStartingActionsFromBrowserUse(actions []client.GetAspectAgentsOrganizationAspectAspectAppAgentsV2AgentV2ConnectionEdgesAgentV2EdgeNodeAgentBrowserUseStartingActionsStartingAction) []StartingAction {
	result := make([]StartingAction, 0, len(actions))
	for i := range actions {
		result = append(result, extractStartingActionFromInfo(&actions[i]))
	}
	return result
}

// Interface for starting action getters
type startingActionInfoGetter interface {
	GetId() string
	GetName() string
	GetPrompt() string
	GetOrder() int
	GetEnabled() bool
	GetIcon() *string
	GetVariables() []client.StartingActionInfoVariablesStartingActionVariable
}

// extractStartingActionFromInfo extracts a single starting action from any type that implements startingActionInfoGetter
func extractStartingActionFromInfo(a startingActionInfoGetter) StartingAction {
	action := StartingAction{
		ID:      a.GetId(),
		Name:    a.GetName(),
		Prompt:  a.GetPrompt(),
		Order:   a.GetOrder(),
		Enabled: a.GetEnabled(),
	}
	if icon := a.GetIcon(); icon != nil {
		action.Icon = *icon
	}
	// Extract variables
	for _, v := range a.GetVariables() {
		variable := StartingActionVariable{
			Key:      v.Key,
			Label:    v.Label,
			Type:     string(v.Type),
			Required: v.Required,
		}
		if v.Placeholder != nil {
			variable.Placeholder = *v.Placeholder
		}
		// Extract options
		for _, opt := range v.Options {
			variable.Options = append(variable.Options, StartingActionSelectOption{
				Label: opt.Label,
				Value: opt.Value,
			})
		}
		// Extract validation
		if v.Validation != nil {
			variable.Validation = &StartingActionValidation{
				MinLength:    v.Validation.MinLength,
				MaxLength:    v.Validation.MaxLength,
				ErrorMessage: "",
				Pattern:      "",
			}
			if v.Validation.Pattern != nil {
				variable.Validation.Pattern = *v.Validation.Pattern
			}
			if v.Validation.ErrorMessage != nil {
				variable.Validation.ErrorMessage = *v.Validation.ErrorMessage
			}
		}
		action.Variables = append(action.Variables, variable)
	}
	return action
}

// Tool type getter interfaces - these match the getter methods generated by genqlient
// for any concrete type that embeds the corresponding fragment type.
// This allows us to use interface assertions that work with all concrete types,
// regardless of which query they came from.

type createRecordToolGetter interface {
	client.AgentToolInfo
	GetAspect() *client.AgentToolInfoAspect
	GetFieldsV2() []client.AgentToolInfoFieldsV2AgentRecordCreateFieldTool
}

type searchAspectToolGetter interface {
	client.AgentToolInfo
	GetAspect() *client.AgentToolInfoAspect
	GetQueryDescription() string
	GetLimit() *int
	GetFields() []client.AgentToolInfoFieldsAgentSearchFieldTool
}

type updateRecordToolGetter interface {
	client.AgentToolInfo
	GetAspect() *client.AgentToolInfoAspect
	GetHandleDescription() *string
	GetFieldsV2() []client.AgentToolInfoFieldsV2AgentRecordUpdateFieldTool
}

type searchTableToolGetter interface {
	client.AgentToolInfo
	GetAspect() *client.AgentToolInfoAspect
	GetTable() *client.AgentToolInfoTableAspectSearchTable
	GetQueryDescription() string
	GetLimit() *int
	GetAttributeFields() []client.AgentToolInfoAttributeFieldsAspectField
	GetReturnFields() []client.AgentToolInfoReturnFieldsAgentSearchTableToolReturnField
}

type relateRecordToolGetter interface {
	client.AgentToolInfo
	GetAspect() *client.AgentToolInfoAspect
	GetRelatedAspect() *client.AgentToolInfoRelatedAspect
}

type executeWorkflowToolGetter interface {
	client.AgentToolInfo
	GetAutomation() *client.AgentToolInfoAutomation
	GetInputs() []client.AgentToolInfoInputsAgentExecuteWorkflowInputParameter
	GetOutputs() []client.AgentToolInfoOutputsAgentExecuteWorkflowOutputParameter
}

type runAgentToolGetter interface {
	client.AgentToolInfo
	GetTargetAgent() client.AgentToolInfoTargetAgentAgentV2
	GetWorkerTaskPrompt() string
}

type mcpToolGetter interface {
	client.AgentToolInfo
	GetMcpToolName() string
	GetServerUrl() string
	GetHeaders() []client.AgentToolInfoHeadersAgentToolHeader
}

// extractToolFromAgentToolInfo extracts an AgentTool from the genqlient tool interface.
// Uses interface assertions to work with all concrete types that embed the fragment types.
func extractToolFromAgentToolInfo(toolInfo client.AgentToolInfo) AgentTool {
	tool := AgentTool{
		ID:          toolInfo.GetId(),
		Name:        toolInfo.GetName(),
		Description: toolInfo.GetDescription(),
	}

	if typename := toolInfo.GetTypename(); typename != nil {
		tool.Type = *typename
	}
	if startMsg := toolInfo.GetStartMessage(); startMsg != nil {
		tool.StartMessage = *startMsg
	}

	// Use interface assertions to extract tool-type-specific fields.
	// This works with all concrete types that embed the fragment types,
	// regardless of which GraphQL query they came from.
	if t, ok := toolInfo.(createRecordToolGetter); ok {
		if t.GetAspect() != nil {
			aspect := *t.GetAspect()
			tool.AspectID = aspect.GetId()
			tool.AspectName = aspect.GetName()
		}
		tool.RawConfig = extractCreateRecordToolConfigFromGetter(t)
	} else if t, ok := toolInfo.(searchAspectToolGetter); ok {
		if t.GetAspect() != nil {
			aspect := *t.GetAspect()
			tool.AspectID = aspect.GetId()
			tool.AspectName = aspect.GetName()
		}
		tool.RawConfig = extractSearchAspectToolConfigFromGetter(t)
	} else if t, ok := toolInfo.(updateRecordToolGetter); ok {
		if t.GetAspect() != nil {
			aspect := *t.GetAspect()
			tool.AspectID = aspect.GetId()
			tool.AspectName = aspect.GetName()
		}
		tool.RawConfig = extractUpdateRecordToolConfigFromGetter(t)
	} else if t, ok := toolInfo.(searchTableToolGetter); ok {
		if t.GetAspect() != nil {
			aspect := *t.GetAspect()
			tool.AspectID = aspect.GetId()
			tool.AspectName = aspect.GetName()
		}
		if t.GetTable() != nil {
			table := *t.GetTable()
			tool.SearchTableID = table.GetId()
			// Extract the aspect the search table belongs to (for recursive discovery)
			// SearchTables live on Apps/Elements, not as global organization Tables
			tableAspect := table.GetAspect()
			if tableAspect != nil {
				tool.SearchTableAspectID = tableAspect.GetId()
				// Determine the type from __typename
				// Note: Only Elements have SearchTables, not Apps
				if typename := tableAspect.GetTypename(); typename != nil {
					switch *typename {
					case "AspectElement":
						tool.SearchTableAspectType = "Element"
					case "AspectApp":
						// Apps don't actually have SearchTables, but handle it anyway
						tool.SearchTableAspectType = "App"
					default:
						tool.SearchTableAspectType = "Element" // Default to Element (the only type with SearchTables)
					}
				}
			}
		}
		tool.RawConfig = extractSearchTableToolConfigFromGetter(t)
	} else if t, ok := toolInfo.(relateRecordToolGetter); ok {
		if t.GetAspect() != nil {
			aspect := *t.GetAspect()
			tool.AspectID = aspect.GetId()
			tool.AspectName = aspect.GetName()
		}
		if t.GetRelatedAspect() != nil {
			relatedAspect := *t.GetRelatedAspect()
			tool.RelatedAspectID = relatedAspect.GetId()
			tool.RelatedAspectName = relatedAspect.GetName()
		}
		// RelateRecordTool has no RawConfig (no tool-specific fields to export)
	} else if t, ok := toolInfo.(executeWorkflowToolGetter); ok {
		if t.GetAutomation() != nil {
			tool.AutomationID = t.GetAutomation().Id
			tool.AutomationName = t.GetAutomation().Name
		}
		tool.RawConfig = extractExecuteWorkflowToolConfigFromGetter(t)
	} else if t, ok := toolInfo.(runAgentToolGetter); ok {
		targetAgent := t.GetTargetAgent()
		tool.TargetAgentID = targetAgent.GetId()
		tool.TargetAgentName = targetAgent.GetName()
		tool.RawConfig = extractRunAgentToolConfigFromGetter(t)
	} else if t, ok := toolInfo.(mcpToolGetter); ok {
		config := map[string]interface{}{
			"mcp_tool_name": t.GetMcpToolName(),
			"server_url":    t.GetServerUrl(),
		}
		// Extract headers if present
		headersData := t.GetHeaders()
		if len(headersData) > 0 {
			headers := make([]map[string]interface{}, 0, len(headersData))
			for _, h := range headersData {
				headers = append(headers, map[string]interface{}{
					"key":   h.Key,
					"value": h.Value,
				})
			}
			config["headers"] = headers
		}
		tool.RawConfig = config
	}
	// SelectBotRouteTool has no additional fields beyond common AgentTool fields

	return tool
}

// Tool config extraction helpers - store relevant fields for HCL generation.
// These functions use the getter interfaces to work with all concrete types.

func extractCreateRecordToolConfigFromGetter(t createRecordToolGetter) map[string]interface{} {
	config := make(map[string]interface{})
	fieldsV2 := t.GetFieldsV2()
	fields := make([]map[string]interface{}, 0, len(fieldsV2))
	for _, f := range fieldsV2 {
		// Use f.Name if provided, otherwise fallback to the field's actual name
		var name string
		if f.Name != nil && *f.Name != "" {
			name = *f.Name
		} else if f.Field != nil {
			name = f.Field.GetName()
		}
		// Use f.Description if provided, otherwise generate a default
		var description string
		if f.Description != nil && *f.Description != "" {
			description = *f.Description
		} else {
			description = "Field for " + name
		}
		// Handle required - use false as default if nil
		required := false
		if f.Required != nil {
			required = *f.Required
		}
		field := map[string]interface{}{
			"name":        name,
			"description": description,
			"required":    required,
		}
		if f.Field != nil {
			field["field_id"] = f.Field.GetId()
			field["field_name"] = f.Field.GetName()
		}
		fields = append(fields, field)
	}
	config["fields"] = fields
	return config
}

func extractSearchAspectToolConfigFromGetter(t searchAspectToolGetter) map[string]interface{} {
	config := make(map[string]interface{})
	if qd := t.GetQueryDescription(); qd != "" {
		config["query_description"] = qd
	}
	if limit := t.GetLimit(); limit != nil {
		config["result_limit"] = *limit
	}
	fieldsData := t.GetFields()
	fields := make([]map[string]interface{}, 0, len(fieldsData))
	for _, f := range fieldsData {
		// Use f.Name if provided, otherwise fallback to the field's actual name
		var name string
		if f.Name != nil && *f.Name != "" {
			name = *f.Name
		} else if f.Field != nil {
			name = f.Field.GetName()
		}
		// Use f.Description if provided, otherwise generate a default
		var description string
		if f.Description != nil && *f.Description != "" {
			description = *f.Description
		} else {
			description = "Field for " + name
		}
		field := map[string]interface{}{
			"name":        name,
			"description": description,
		}
		if f.Field != nil {
			field["field_id"] = f.Field.GetId()
			field["field_name"] = f.Field.GetName()
		}
		fields = append(fields, field)
	}
	config["fields"] = fields
	return config
}

func extractUpdateRecordToolConfigFromGetter(t updateRecordToolGetter) map[string]interface{} {
	config := make(map[string]interface{})
	if hd := t.GetHandleDescription(); hd != nil {
		config["handle_description"] = *hd
	}
	fieldsV2 := t.GetFieldsV2()
	fields := make([]map[string]interface{}, 0, len(fieldsV2))
	for _, f := range fieldsV2 {
		// Use f.Name if provided, otherwise fallback to the field's actual name
		var name string
		if f.Name != nil && *f.Name != "" {
			name = *f.Name
		} else if f.Field != nil {
			name = f.Field.GetName()
		}
		// Use f.Description if provided, otherwise generate a default
		var description string
		if f.Description != nil && *f.Description != "" {
			description = *f.Description
		} else {
			description = "Field for " + name
		}
		// Handle required - use false as default if nil
		required := false
		if f.Required != nil {
			required = *f.Required
		}
		field := map[string]interface{}{
			"name":        name,
			"description": description,
			"required":    required,
		}
		if f.Field != nil {
			field["field_id"] = f.Field.GetId()
			field["field_name"] = f.Field.GetName()
		}
		fields = append(fields, field)
	}
	config["fields"] = fields
	return config
}

func extractSearchTableToolConfigFromGetter(t searchTableToolGetter) map[string]interface{} {
	config := make(map[string]interface{})
	if qd := t.GetQueryDescription(); qd != "" {
		config["query_description"] = qd
	}
	if limit := t.GetLimit(); limit != nil {
		config["result_limit"] = *limit
	}
	// Attribute fields - these are interfaces with GetId() and GetName() methods
	attrFieldsData := t.GetAttributeFields()
	attrFields := make([]map[string]interface{}, 0, len(attrFieldsData))
	for _, f := range attrFieldsData {
		attrFields = append(attrFields, map[string]interface{}{
			"field_id":   f.GetId(),
			"field_name": f.GetName(),
		})
	}
	config["attribute_fields"] = attrFields
	// Return fields
	returnFieldsData := t.GetReturnFields()
	returnFields := make([]map[string]interface{}, 0, len(returnFieldsData))
	for _, f := range returnFieldsData {
		rf := map[string]interface{}{
			"display_name": f.Name,
		}
		if f.Field != nil {
			fieldIface := *f.Field // Dereference pointer to interface
			rf["field_id"] = fieldIface.GetId()
			rf["field_name"] = fieldIface.GetName()
		}
		returnFields = append(returnFields, rf)
	}
	config["return_fields"] = returnFields
	return config
}

func extractExecuteWorkflowToolConfigFromGetter(t executeWorkflowToolGetter) map[string]interface{} {
	config := make(map[string]interface{})
	// Inputs: parameter_name (required) is the binding; display_name (optional) is agent-facing when different
	inputsData := t.GetInputs()
	inputs := make([]map[string]interface{}, 0, len(inputsData))
	for _, inp := range inputsData {
		paramName := inp.TriggerParameter.Name
		input := map[string]interface{}{
			"parameter_name": paramName,
			"description":    inp.Description,
			"required":       inp.Required,
		}
		if inp.Name != "" && inp.Name != paramName {
			input["display_name"] = inp.Name
		}
		inputs = append(inputs, input)
	}
	config["inputs"] = inputs
	// Outputs: workflow_property_name (required) is the binding; display_name (optional) is agent-facing when different
	outputsData := t.GetOutputs()
	outputs := make([]map[string]interface{}, 0, len(outputsData))
	for _, out := range outputsData {
		propName := out.WorkflowPropertyName
		output := map[string]interface{}{
			"workflow_property_name": propName,
		}
		if out.Name != "" && out.Name != propName {
			output["display_name"] = out.Name
		}
		outputs = append(outputs, output)
	}
	config["outputs"] = outputs
	return config
}

func extractRunAgentToolConfigFromGetter(t runAgentToolGetter) map[string]interface{} {
	config := make(map[string]interface{})
	if prompt := t.GetWorkerTaskPrompt(); prompt != "" {
		config["worker_task_prompt"] = prompt
	}
	return config
}

// getAppFlows retrieves flow information for an app (if configured) using genqlient
func getAppFlows(ctx context.Context, c *client.Client, app *App) error {
	result, err := client.GetAspectFlowBlocks(ctx, c.Genqlient(), app.ID)
	if err != nil {
		return err
	}

	if result.Organization.Aspect == nil {
		return nil
	}

	// Type assert to get the AspectApp type
	aspectApp, ok := (*result.Organization.Aspect).(*client.GetAspectFlowBlocksOrganizationAspectAspectApp)
	if !ok {
		return nil // Not an app, no flows
	}

	// Check if any stage has flow nodes
	hasFlowNodes := false
	for _, stage := range aspectApp.Stages {
		if len(stage.FlowBlock.Nodes) > 0 {
			hasFlowNodes = true
			break
		}
	}

	// If the app has flow nodes configured, add the flow
	if hasFlowNodes {
		app.Flows = []Flow{
			{
				ID:   app.ID, // Flow ID is the same as app ID
				Name: app.Name + " Flow",
			},
		}
	}

	return nil
}

// getAppWidgets retrieves all widgets for an app (if any exist) using genqlient
func getAppWidgets(ctx context.Context, c *client.Client, app *App) error {
	result, err := client.GetAspectDisplayWidgets(ctx, c.Genqlient(), app.ID)
	if err != nil {
		return err
	}

	if result.Organization.Aspect == nil {
		return nil
	}

	// Type assert to get the AspectApp type
	aspectApp, ok := (*result.Organization.Aspect).(*client.GetAspectDisplayWidgetsOrganizationAspectAspectApp)
	if !ok {
		return nil // Not an app, no widgets
	}

	app.Widgets = make([]Widget, 0, len(aspectApp.DisplayWidgets.Edges))
	logger.Debug("retrieved widgets from API", "widgetCount", len(aspectApp.DisplayWidgets.Edges))

	for _, edge := range aspectApp.DisplayWidgets.Edges {
		// Node is an interface, use getter methods
		widgetType := ""
		if tn := edge.Node.GetTypename(); tn != nil {
			widgetType = *tn
		}
		widget := Widget{
			ID:   edge.Node.GetId(),
			Name: edge.Node.GetName(),
			Type: widgetType,
		}

		// Extract type-specific fields based on widget type
		switch w := edge.Node.(type) {
		case *client.GetAspectDisplayWidgetsOrganizationAspectAspectAppDisplayWidgetsDisplayWidgetConnectionEdgesDisplayWidgetEdgeNodeDisplayWidgetRelatedAspect:
			if w.Aspect != nil {
				widget.AspectID = (*w.Aspect).GetId()
			}
			widget.Columns = w.Columns
			widget.Rows = w.Rows
		case *client.GetAspectDisplayWidgetsOrganizationAspectAspectAppDisplayWidgetsDisplayWidgetConnectionEdgesDisplayWidgetEdgeNodeDisplayWidgetRelatedLinkAction:
			if w.Aspect != nil {
				widget.AspectID = (*w.Aspect).GetId()
			}
			widget.ButtonType = string(w.ButtonType)
			if w.Color != nil {
				widget.Color = *w.Color
			}
			if w.Icon != nil {
				widget.Icon = *w.Icon
			}
			widget.FullWidth = w.FullWidth
			widget.Columns = w.Columns
			widget.Rows = w.Rows
		case *client.GetAspectDisplayWidgetsOrganizationAspectAspectAppDisplayWidgetsDisplayWidgetConnectionEdgesDisplayWidgetEdgeNodeDisplayWidgetRelatedCreateAction:
			if w.Aspect != nil {
				widget.AspectID = (*w.Aspect).GetId()
			}
			widget.ButtonType = string(w.ButtonType)
			if w.Color != nil {
				widget.Color = *w.Color
			}
			if w.Icon != nil {
				widget.Icon = *w.Icon
			}
			widget.FullWidth = w.FullWidth
		case *client.GetAspectDisplayWidgetsOrganizationAspectAspectAppDisplayWidgetsDisplayWidgetConnectionEdgesDisplayWidgetEdgeNodeDisplayWidgetRunAutomationAction:
			if w.Automation != nil {
				widget.AutomationID = w.Automation.GetId()
			}
			widget.ButtonType = string(w.ButtonType)
			if w.Color != nil {
				widget.Color = *w.Color
			}
			if w.Icon != nil {
				widget.Icon = *w.Icon
			}
			widget.FullWidth = w.FullWidth
		}

		app.Widgets = append(app.Widgets, widget)
	}

	return nil
}

// getAspectViews retrieves all views for an aspect (App, Element, or Task) using genqlient
func getAspectViews(ctx context.Context, c *client.Client, aspectID string) ([]View, error) {
	result, err := client.GetAspectViews(ctx, c.Genqlient(), aspectID)
	if err != nil {
		return nil, err
	}

	if result.Organization.Aspect == nil {
		return nil, nil
	}

	aspect := *result.Organization.Aspect
	edges := aspect.GetViews().Edges
	views := make([]View, 0, len(edges))

	for _, edge := range edges {
		node := edge.Node
		typename := ""
		if node.GetTypename() != nil {
			typename = *node.GetTypename()
		}

		displayOrder := 0
		if node.GetDisplayOrder() != nil {
			displayOrder = *node.GetDisplayOrder()
		}

		view := View{
			ID:                  node.GetId(),
			Name:                node.GetName(),
			Type:                typename,
			AllowRecordCreation: node.GetAllowRecordCreation(),
			DisplayOrder:        displayOrder,
		}

		// Extract type-specific fields based on __typename
		switch v := node.(type) {
		case *client.GetAspectViewsOrganizationAspectViewsAspectViewConnectionEdgesAspectViewEdgeNodeAspectViewList:
			view.Columns = v.GetListColumns()
			if v.GetDensity() != nil {
				view.Density = string(*v.GetDensity())
			}
			if v.GetRows() != nil {
				view.Rows = *v.GetRows()
			}
		case *client.GetAspectViewsOrganizationAspectViewsAspectViewConnectionEdgesAspectViewEdgeNodeAspectViewKanban:
			view.Columns = v.GetKanbanColumns()
			if v.GetDisplayByField() != nil {
				view.DisplayByField = *v.GetDisplayByField()
			}
			view.AllowedFilterFields = v.GetAllowedFilterFields()
		case *client.GetAspectViewsOrganizationAspectViewsAspectViewConnectionEdgesAspectViewEdgeNodeAspectViewCalendar:
			view.Columns = v.GetCalendarColumns()
			if v.GetCalendarDisplayByField() != nil {
				view.DisplayByField = *v.GetCalendarDisplayByField()
			}
		case *client.GetAspectViewsOrganizationAspectViewsAspectViewConnectionEdgesAspectViewEdgeNodeAspectViewDashboard:
			if v.GetDashboard() != nil {
				view.DashboardID = v.GetDashboard().Id
			}
		case *client.GetAspectViewsOrganizationAspectViewsAspectViewConnectionEdgesAspectViewEdgeNodeAspectViewAgent:
			if v.GetAgentViewAgent() != nil {
				view.AgentID = v.GetAgentViewAgent().Id
			}
		}

		views = append(views, view)
	}

	logger.Debug("retrieved views from API", "aspectID", aspectID, "viewCount", len(views))
	return views, nil
}

// getAppViews retrieves all views for an app
func getAppViews(ctx context.Context, c *client.Client, app *App) error {
	views, err := getAspectViews(ctx, c, app.ID)
	if err != nil {
		return err
	}
	app.Views = views
	return nil
}

// getAppAIFileReaders retrieves all file readers (document models) for an app with full details
// Supports: DocumentModelAi, DocumentModelOCR, DocumentModelJson, DocumentModelXml
func getAppAIFileReaders(ctx context.Context, c *client.Client, app *App) error {
	// First, get the list of all document models
	listQuery := `
		query GetAppDocumentModels($aspectId: ID!) {
			organization {
				aspect(id: $aspectId) {
					... on AspectApp {
						documentModels(first: 500) {
							edges {
								node {
									__typename
									id
									name
								}
							}
						}
					}
				}
			}
		}
	`

	var listResult struct {
		Organization struct {
			Aspect *struct {
				DocumentModels struct {
					Edges []struct {
						Node struct {
							Typename string `json:"__typename"`
							ID       string `json:"id"`
							Name     string `json:"name"`
						} `json:"node"`
					} `json:"edges"`
				} `json:"documentModels"`
			} `json:"aspect"`
		} `json:"organization"`
	}

	err := c.ExecuteInto(ctx, listQuery, map[string]interface{}{"aspectId": app.ID}, &listResult)
	if err != nil {
		// Document models may not be available in all environments, so log but don't fail
		return nil
	}

	if listResult.Organization.Aspect == nil {
		return nil
	}

	app.AIFileReaders = make([]FileReader, 0, len(listResult.Organization.Aspect.DocumentModels.Edges))

	// For each file reader, fetch full details based on type
	for _, edge := range listResult.Organization.Aspect.DocumentModels.Edges {
		var reader *FileReader
		var err error

		switch edge.Node.Typename {
		case FileReaderTypeAI:
			reader, err = getAIFileReaderDetails(ctx, c, app.ID, edge.Node.ID)
		case FileReaderTypeOCR:
			reader, err = getTextFileReaderDetails(ctx, c, app.ID, edge.Node.ID)
		case FileReaderTypeJSON:
			reader, err = getJSONFileReaderDetails(ctx, c, app.ID, edge.Node.ID)
		case FileReaderTypeXML:
			reader, err = getXMLFileReaderDetails(ctx, c, app.ID, edge.Node.ID)
		default:
			// Unknown type - store basic info
			reader = &FileReader{
				ID:   edge.Node.ID,
				Name: edge.Node.Name,
				Type: edge.Node.Typename,
			}
		}

		if err != nil {
			// Fall back to basic info if details fetch fails
			app.AIFileReaders = append(app.AIFileReaders, FileReader{
				ID:   edge.Node.ID,
				Name: edge.Node.Name,
				Type: edge.Node.Typename,
			})
			continue
		}

		app.AIFileReaders = append(app.AIFileReaders, *reader)
	}

	app.AIFileReaders, app.FileReaderIDAlias = dedupeFileReadersByName(app.AIFileReaders)

	return nil
}

// dedupeFileReadersByName collapses file readers that share the same name into
// a single canonical reader. Platform APIs often return one DocumentModel per
// workflow that references it, so a single logical "Validation Output" reader
// can appear 4+ times with identical name and structure. Truth HCL is
// hand-authored to have one resource per name. This dedup keeps the first
// occurrence (sorted by ID for determinism) and records the
// duplicate-ID → canonical-ID mapping so downstream resolution can rewrite
// file_reader_id references that pointed at the dropped copies.
func dedupeFileReadersByName(readers []FileReader) ([]FileReader, map[string]string) {
	if len(readers) == 0 {
		return readers, nil
	}
	// Sort by ID for determinism.
	sorted := make([]FileReader, len(readers))
	copy(sorted, readers)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })

	canonical := make(map[string]*FileReader) // key: type + "\x00" + name
	alias := make(map[string]string)          // duplicate ID -> canonical ID
	out := make([]FileReader, 0, len(sorted))

	for i := range sorted {
		r := sorted[i]
		key := r.Type + "\x00" + r.Name
		if existing, ok := canonical[key]; ok {
			// Prefer the one with non-empty structure (platform sometimes
			// returns partial data for dupes); otherwise keep the existing.
			if existing.Structure == "" && r.Structure != "" {
				// Swap: the newly-seen copy is richer, promote it.
				alias[existing.ID] = r.ID
				// Replace in out[]
				for idx := range out {
					if out[idx].ID == existing.ID {
						out[idx] = r
						break
					}
				}
				canonical[key] = &out[len(out)-1] // re-aim pointer
				continue
			}
			alias[r.ID] = existing.ID
			continue
		}
		out = append(out, r)
		canonical[key] = &out[len(out)-1]
	}
	return out, alias
}

// getAIFileReaderDetails fetches full details for a single AI file reader
func getAIFileReaderDetails(ctx context.Context, c *client.Client, appID, documentModelID string) (*AIFileReader, error) {
	query := `
		query GetDocumentModel($aspectId: ID!, $documentModelId: ID!) {
			organization {
				aspect(id: $aspectId) {
					... on AspectApp {
						documentModel(id: $documentModelId) {
							__typename
							id
							... on DocumentModelAi {
								name
								instructions
								fields {
									edges {
										node {
											id
											name
											fieldType
											... on DocumentModelAiField {
												description
												required
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"aspectId":        appID,
		"documentModelId": documentModelID,
	}

	var result struct {
		Organization struct {
			Aspect struct {
				DocumentModel *struct {
					Typename     string `json:"__typename"`
					ID           string `json:"id"`
					Name         string `json:"name"`
					Instructions string `json:"instructions"`
					Fields       struct {
						Edges []struct {
							Node struct {
								ID          string `json:"id"`
								Name        string `json:"name"`
								FieldType   string `json:"fieldType"`
								Description string `json:"description"`
								Required    bool   `json:"required"`
							} `json:"node"`
						} `json:"edges"`
					} `json:"fields"`
				} `json:"documentModel"`
			} `json:"aspect"`
		} `json:"organization"`
	}

	err := c.ExecuteInto(ctx, query, variables, &result)
	if err != nil {
		return nil, err
	}

	if result.Organization.Aspect.DocumentModel == nil {
		return nil, fmt.Errorf("document model not found: %s", documentModelID)
	}

	docModel := result.Organization.Aspect.DocumentModel

	reader := &AIFileReader{
		ID:           docModel.ID,
		Name:         docModel.Name,
		Type:         docModel.Typename,
		Instructions: docModel.Instructions,
		Fields:       make([]AIFileReaderField, 0, len(docModel.Fields.Edges)),
	}

	for _, fieldEdge := range docModel.Fields.Edges {
		reader.Fields = append(reader.Fields, AIFileReaderField{
			ID:          fieldEdge.Node.ID,
			Name:        fieldEdge.Node.Name,
			Description: fieldEdge.Node.Description,
			Type:        fieldEdge.Node.FieldType,
			Required:    fieldEdge.Node.Required,
		})
	}

	return reader, nil
}

// getTextFileReaderDetails fetches details for a text (OCR) file reader
func getTextFileReaderDetails(ctx context.Context, c *client.Client, appID, documentModelID string) (*FileReader, error) {
	query := `
		query GetDocumentModel($aspectId: ID!, $documentModelId: ID!) {
			organization {
				aspect(id: $aspectId) {
					... on AspectApp {
						documentModel(id: $documentModelId) {
							__typename
							id
							... on DocumentModelOCR {
								name
							}
						}
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"aspectId":        appID,
		"documentModelId": documentModelID,
	}

	var result struct {
		Organization struct {
			Aspect struct {
				DocumentModel *struct {
					Typename string `json:"__typename"`
					ID       string `json:"id"`
					Name     string `json:"name"`
				} `json:"documentModel"`
			} `json:"aspect"`
		} `json:"organization"`
	}

	err := c.ExecuteInto(ctx, query, variables, &result)
	if err != nil {
		return nil, err
	}

	if result.Organization.Aspect.DocumentModel == nil {
		return nil, fmt.Errorf("document model not found: %s", documentModelID)
	}

	docModel := result.Organization.Aspect.DocumentModel

	return &FileReader{
		ID:   docModel.ID,
		Name: docModel.Name,
		Type: docModel.Typename,
	}, nil
}

// getJSONFileReaderDetails fetches details for a JSON file reader.
//
// NOTE: the schema has no scalar `structure` field on DocumentModelJson — the
// real shape is `jsonStructure { properties { ... } }` where each property is
// a polymorphic DocumentModelJsonStructureParameter (text, bool, number,
// date, datetime, decimal, object, array). We fetch the typename + name of
// each parameter and reshape the result into the tagged-union form that the
// elementum_json_file_reader terraform resource expects:
//
//	{ properties: [ {text: {name: "x"}}, {bool: {name: "y"}}, ... ] }
//
// which is jsonencode()'d by the HCL generator.
func getJSONFileReaderDetails(ctx context.Context, c *client.Client, appID, documentModelID string) (*FileReader, error) {
	query := `
		query GetDocumentModel($aspectId: ID!, $documentModelId: ID!) {
			organization {
				aspect(id: $aspectId) {
					... on AspectApp {
						documentModel(id: $documentModelId) {
							__typename
							id
							... on DocumentModelJson {
								name
								jsonStructure {
									properties {
										__typename
										... on DocumentModelJsonStructureTextValue { name }
										... on DocumentModelJsonStructureBooleanValue { name }
										... on DocumentModelJsonStructureNumberValue { name }
										... on DocumentModelJsonStructureDecimalValue { name }
										... on DocumentModelJsonStructureDateValue { name format }
										... on DocumentModelJsonStructureDateTimeValue { name format }
										... on DocumentModelJsonStructureObject {
											name
											properties {
												__typename
												... on DocumentModelJsonStructureTextValue { name }
												... on DocumentModelJsonStructureBooleanValue { name }
												... on DocumentModelJsonStructureNumberValue { name }
												... on DocumentModelJsonStructureDecimalValue { name }
												... on DocumentModelJsonStructureDateValue { name format }
												... on DocumentModelJsonStructureDateTimeValue { name format }
											}
										}
										... on DocumentModelJsonStructureArray {
											name
											arrayType {
												__typename
												... on DocumentModelJsonStructureTextValue { name }
												... on DocumentModelJsonStructureBooleanValue { name }
												... on DocumentModelJsonStructureNumberValue { name }
												... on DocumentModelJsonStructureDecimalValue { name }
												... on DocumentModelJsonStructureDateValue { name format }
												... on DocumentModelJsonStructureDateTimeValue { name format }
												... on DocumentModelJsonStructureObject { name }
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"aspectId":        appID,
		"documentModelId": documentModelID,
	}

	var result struct {
		Organization struct {
			Aspect struct {
				DocumentModel *struct {
					Typename      string `json:"__typename"`
					ID            string `json:"id"`
					Name          string `json:"name"`
					JsonStructure *struct {
						Properties []map[string]interface{} `json:"properties"`
					} `json:"jsonStructure"`
				} `json:"documentModel"`
			} `json:"aspect"`
		} `json:"organization"`
	}

	err := c.ExecuteInto(ctx, query, variables, &result)
	if err != nil {
		return nil, err
	}

	if result.Organization.Aspect.DocumentModel == nil {
		return nil, fmt.Errorf("document model not found: %s", documentModelID)
	}

	docModel := result.Organization.Aspect.DocumentModel

	// Reshape the GraphQL response into the tagged-union form Terraform expects.
	structureJSON := ""
	if docModel.JsonStructure != nil {
		reshaped := map[string]interface{}{
			"properties": reshapeJSONStructureParams(docModel.JsonStructure.Properties),
		}
		if b, err := json.Marshal(reshaped); err == nil {
			structureJSON = string(b)
		}
	}

	return &FileReader{
		ID:        docModel.ID,
		Name:      docModel.Name,
		Type:      docModel.Typename,
		Structure: structureJSON,
	}, nil
}

// reshapeJSONStructureParams converts a raw list of GraphQL-returned params
// (each with `__typename` + type-specific fields) into the tagged-union form
// { text: {name}, bool: {name}, date: {name, format}, ... } used by the
// elementum_json_file_reader resource's `structure` attribute.
func reshapeJSONStructureParams(params []map[string]interface{}) []map[string]interface{} {
	if len(params) == 0 {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(params))
	for _, p := range params {
		tn, _ := p["__typename"].(string)
		name, _ := p["name"].(string)
		var tag string
		inner := map[string]interface{}{"name": name}
		switch tn {
		case "DocumentModelJsonStructureTextValue":
			tag = "text"
		case "DocumentModelJsonStructureBooleanValue":
			tag = "bool"
		case "DocumentModelJsonStructureNumberValue":
			tag = "number"
		case "DocumentModelJsonStructureDecimalValue":
			tag = "decimal"
		case "DocumentModelJsonStructureDateValue":
			tag = "date"
			if f, ok := p["format"].(string); ok && f != "" {
				inner["format"] = f
			}
		case "DocumentModelJsonStructureDateTimeValue":
			tag = "datetime"
			if f, ok := p["format"].(string); ok && f != "" {
				inner["format"] = f
			}
		case "DocumentModelJsonStructureObject":
			tag = "object"
			if nested, ok := p["properties"].([]interface{}); ok {
				nestedMaps := make([]map[string]interface{}, 0, len(nested))
				for _, n := range nested {
					if m, ok := n.(map[string]interface{}); ok {
						nestedMaps = append(nestedMaps, m)
					}
				}
				inner["properties"] = reshapeJSONStructureParams(nestedMaps)
			}
		case "DocumentModelJsonStructureArray":
			tag = "array"
			if at, ok := p["arrayType"].(map[string]interface{}); ok {
				reshapedElem := reshapeJSONStructureParams([]map[string]interface{}{at})
				if len(reshapedElem) == 1 {
					inner["array_type"] = reshapedElem[0]
				}
			}
		default:
			// Unknown typename — skip rather than corrupt the output.
			continue
		}
		out = append(out, map[string]interface{}{tag: inner})
	}
	return out
}

// getXMLFileReaderDetails fetches details for an XML file reader
func getXMLFileReaderDetails(ctx context.Context, c *client.Client, appID, documentModelID string) (*FileReader, error) {
	query := `
		query GetDocumentModel($aspectId: ID!, $documentModelId: ID!) {
			organization {
				aspect(id: $aspectId) {
					... on AspectApp {
						documentModel(id: $documentModelId) {
							__typename
							id
							... on DocumentModelXml {
								name
								values
							}
						}
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"aspectId":        appID,
		"documentModelId": documentModelID,
	}

	var result struct {
		Organization struct {
			Aspect struct {
				DocumentModel *struct {
					Typename string                 `json:"__typename"`
					ID       string                 `json:"id"`
					Name     string                 `json:"name"`
					Values   map[string]interface{} `json:"values"`
				} `json:"documentModel"`
			} `json:"aspect"`
		} `json:"organization"`
	}

	err := c.ExecuteInto(ctx, query, variables, &result)
	if err != nil {
		return nil, err
	}

	if result.Organization.Aspect.DocumentModel == nil {
		return nil, fmt.Errorf("document model not found: %s", documentModelID)
	}

	docModel := result.Organization.Aspect.DocumentModel

	// Encode the values back to JSON string for terraform
	valuesJSON := ""
	if docModel.Values != nil {
		if b, err := json.Marshal(docModel.Values); err == nil {
			valuesJSON = string(b)
		}
	}

	return &FileReader{
		ID:        docModel.ID,
		Name:      docModel.Name,
		Type:      docModel.Typename,
		Structure: valuesJSON, // Store in Structure field for XML too
	}, nil
}

// getAppApprovalProcesses retrieves all approval processes for an app using genqlient
func getAppApprovalProcesses(ctx context.Context, c *client.Client, app *App) error {
	result, err := client.GetAspectApprovalChainTemplates(ctx, c.Genqlient(), app.ID)
	if err != nil {
		return err
	}

	if result.Organization.Aspect == nil {
		return nil
	}

	// Type assert to get the AspectApp type
	aspectApp, ok := (*result.Organization.Aspect).(*client.GetAspectApprovalChainTemplatesOrganizationAspectAspectApp)
	if !ok {
		return nil // Not an app, no approval templates
	}

	app.Approvals = make([]ApprovalProcess, 0, len(aspectApp.ApprovalChainTemplates.Edges))
	for _, edge := range aspectApp.ApprovalChainTemplates.Edges {
		app.Approvals = append(app.Approvals, ApprovalProcess{
			ID:   edge.Node.Id,
			Name: edge.Node.Name,
		})
	}

	return nil
}

// getAppAccessPolicies retrieves all access policies for an app using genqlient
func getAppAccessPolicies(ctx context.Context, c *client.Client, app *App) error {
	result, err := client.GetAccessPolicies(ctx, c.Genqlient(), app.ID)
	if err != nil {
		return err
	}

	org := result.GetOrganization()
	aspect := org.Aspect
	if aspect == nil {
		return nil
	}

	// Type assert to get the AspectApp type
	aspectApp, ok := (*aspect).(*client.GetAccessPoliciesOrganizationAspectAspectApp)
	if !ok {
		return nil // Not an app
	}

	app.AccessPolicies = make([]AccessPolicy, 0, len(aspectApp.AccessPolicies.Edges))
	for _, edge := range aspectApp.AccessPolicies.Edges {
		policy := extractAccessPolicyFromApp(edge.Node)
		policy.ObjectID = app.ID
		app.AccessPolicies = append(app.AccessPolicies, policy)
	}

	return nil
}

// getAppRoles retrieves all roles for an app using genqlient
func getAppRoles(ctx context.Context, c *client.Client, app *App) error {
	result, err := client.GetRolesWithMembers(ctx, c.Genqlient(), app.ID)
	if err != nil {
		return err
	}

	org := result.GetOrganization()
	aspect := org.Aspect
	if aspect == nil {
		return nil
	}

	// Type assert to get the AspectApp type
	aspectApp, ok := (*aspect).(*client.GetRolesWithMembersOrganizationAspectAspectApp)
	if !ok {
		return nil // Not an app
	}

	app.Roles = make([]Role, 0, len(aspectApp.Roles.Edges))
	for _, edge := range aspectApp.Roles.Edges {
		node := edge.Node

		role := Role{
			ID:      node.Id,
			Name:    node.Name,
			Managed: node.Managed,
			Tags:    node.Tags,
		}

		// Extract description (may be nil)
		if node.Description != nil {
			role.Description = *node.Description
		}

		// Convert AutoShare enum values to strings
		for _, as := range node.AutoShare {
			role.AutoShare = append(role.AutoShare, string(as))
		}

		// Extract users with emails
		for _, userEdge := range node.Users.Edges {
			role.Users = append(role.Users, RoleMember{
				ID:   userEdge.Node.Id,
				Name: userEdge.Node.Email,
			})
		}

		// Extract groups with names
		for _, groupEdge := range node.Groups.Edges {
			role.Groups = append(role.Groups, RoleMember{
				ID:   groupEdge.Node.Id,
				Name: groupEdge.Node.Name,
			})
		}

		// Extract permissions
		for _, permEdge := range node.PermissionsV2.Edges {
			perm := RolePermission{
				Group: string(permEdge.Node.Group.Group),
				Level: string(permEdge.Node.Level.Level),
			}
			// Extract custom permissions
			for _, ap := range permEdge.Node.AssignedPermissions {
				perm.CustomPermissions = append(perm.CustomPermissions, string(ap.Permission))
			}
			role.Permissions = append(role.Permissions, perm)
		}

		app.Roles = append(app.Roles, role)
	}

	return nil
}

// extractAccessPolicyFromApp extracts an AccessPolicy from AspectApp response
func extractAccessPolicyFromApp(node client.GetAccessPoliciesOrganizationAspectAspectAppAccessPoliciesAspectAccessPolicyConnectionEdgesAspectAccessPolicyEdgeNodeAspectAccessPolicy) AccessPolicy {
	policy := AccessPolicy{
		ID:         node.Id,
		UserEmails: make(map[string]string),
		GroupNames: make(map[string]string),
	}

	// Parse filter
	if len(node.Filter) > 0 {
		if err := json.Unmarshal(node.Filter, &policy.Filter); err != nil {
			logger.Warn("failed to unmarshal access policy filter", "policyID", node.Id, "error", err)
		}
	}

	// Extract user IDs and emails
	for _, edge := range node.Users.Edges {
		policy.UserIDs = append(policy.UserIDs, edge.Node.Id)
		if edge.Node.Email != "" {
			policy.UserEmails[edge.Node.Id] = edge.Node.Email
		}
	}

	// Extract group IDs and names
	for _, edge := range node.Groups.Edges {
		policy.GroupIDs = append(policy.GroupIDs, edge.Node.Id)
		if edge.Node.Name != "" {
			policy.GroupNames[edge.Node.Id] = edge.Node.Name
		}
	}

	return policy
}

// extractAccessPolicyFromElement extracts an AccessPolicy from AspectElement response
func extractAccessPolicyFromElement(node client.GetAccessPoliciesOrganizationAspectAspectElementAccessPoliciesAspectAccessPolicyConnectionEdgesAspectAccessPolicyEdgeNodeAspectAccessPolicy) AccessPolicy {
	policy := AccessPolicy{
		ID:         node.Id,
		UserEmails: make(map[string]string),
		GroupNames: make(map[string]string),
	}

	// Parse filter
	if len(node.Filter) > 0 {
		if err := json.Unmarshal(node.Filter, &policy.Filter); err != nil {
			logger.Warn("failed to unmarshal access policy filter", "policyID", node.Id, "error", err)
		}
	}

	// Extract user IDs and emails
	for _, edge := range node.Users.Edges {
		policy.UserIDs = append(policy.UserIDs, edge.Node.Id)
		if edge.Node.Email != "" {
			policy.UserEmails[edge.Node.Id] = edge.Node.Email
		}
	}

	// Extract group IDs and names
	for _, edge := range node.Groups.Edges {
		policy.GroupIDs = append(policy.GroupIDs, edge.Node.Id)
		if edge.Node.Name != "" {
			policy.GroupNames[edge.Node.Id] = edge.Node.Name
		}
	}

	return policy
}

// extractAccessPolicyFromTask extracts an AccessPolicy from AspectTask response
func extractAccessPolicyFromTask(node client.GetAccessPoliciesOrganizationAspectAspectTaskAccessPoliciesAspectAccessPolicyConnectionEdgesAspectAccessPolicyEdgeNodeAspectAccessPolicy) AccessPolicy {
	policy := AccessPolicy{
		ID:         node.Id,
		UserEmails: make(map[string]string),
		GroupNames: make(map[string]string),
	}

	// Parse filter
	if len(node.Filter) > 0 {
		if err := json.Unmarshal(node.Filter, &policy.Filter); err != nil {
			logger.Warn("failed to unmarshal access policy filter", "policyID", node.Id, "error", err)
		}
	}

	// Extract user IDs and emails
	for _, edge := range node.Users.Edges {
		policy.UserIDs = append(policy.UserIDs, edge.Node.Id)
		if edge.Node.Email != "" {
			policy.UserEmails[edge.Node.Id] = edge.Node.Email
		}
	}

	// Extract group IDs and names
	for _, edge := range node.Groups.Edges {
		policy.GroupIDs = append(policy.GroupIDs, edge.Node.Id)
		if edge.Node.Name != "" {
			policy.GroupNames[edge.Node.Id] = edge.Node.Name
		}
	}

	return policy
}

// GetAccessPoliciesForAspect is a generic function to get access policies for any aspect type (App, Element, Task)
func GetAccessPoliciesForAspect(ctx context.Context, c *client.Client, aspectID string) ([]AccessPolicy, error) {
	result, err := client.GetAccessPolicies(ctx, c.Genqlient(), aspectID)
	if err != nil {
		return nil, err
	}

	org := result.GetOrganization()
	aspect := org.Aspect
	if aspect == nil {
		return nil, nil
	}

	var policies []AccessPolicy

	// Handle different aspect types
	switch a := (*aspect).(type) {
	case *client.GetAccessPoliciesOrganizationAspectAspectApp:
		policies = make([]AccessPolicy, 0, len(a.AccessPolicies.Edges))
		for _, edge := range a.AccessPolicies.Edges {
			policy := extractAccessPolicyFromApp(edge.Node)
			policy.ObjectID = aspectID
			policies = append(policies, policy)
		}
	case *client.GetAccessPoliciesOrganizationAspectAspectElement:
		policies = make([]AccessPolicy, 0, len(a.AccessPolicies.Edges))
		for _, edge := range a.AccessPolicies.Edges {
			policy := extractAccessPolicyFromElement(edge.Node)
			policy.ObjectID = aspectID
			policies = append(policies, policy)
		}
	case *client.GetAccessPoliciesOrganizationAspectAspectTask:
		policies = make([]AccessPolicy, 0, len(a.AccessPolicies.Edges))
		for _, edge := range a.AccessPolicies.Edges {
			policy := extractAccessPolicyFromTask(edge.Node)
			policy.ObjectID = aspectID
			policies = append(policies, policy)
		}
	}

	return policies, nil
}

// getAppRelationships retrieves all relationships (AutoRelations) for an app
func getAppRelationships(ctx context.Context, c *client.Client, app *App) error {
	query := `
		query GetAppRelationships($aspectId: ID!) {
			organization {
				aspect(id: $aspectId) {
					id
					autoRelations {
						edges {
							node {
								id
								filter
								columns {
									aspectField {
										id
										name
									}
									relatedAspectTableField {
										__typename
										... on AspectField {
											id
											name
										}
										... on TableField {
											id
											name
										}
									}
								}
								relatedAspectTable {
									__typename
									... on AspectApp {
										id
										name
										namespace
									}
									... on AspectElement {
										id
										name
										namespace
									}
									... on AspectTask {
										id
										name
									}
									... on Table {
										id
										name
									}
								}
							}
						}
					}
				}
			}
		}
	`

	var result struct {
		Organization struct {
			Aspect *struct {
				AutoRelations struct {
					Edges []struct {
						Node struct {
							ID      string                 `json:"id"`
							Filter  map[string]interface{} `json:"filter"`
							Columns []struct {
								AspectField struct {
									ID   string `json:"id"`
									Name string `json:"name"`
								} `json:"aspectField"`
								RelatedAspectTableField struct {
									Typename string `json:"__typename"`
									ID       string `json:"id"`
									Name     string `json:"name"`
								} `json:"relatedAspectTableField"`
							} `json:"columns"`
							RelatedAspectTable struct {
								Typename  string `json:"__typename"`
								ID        string `json:"id"`
								Name      string `json:"name"`
								Namespace string `json:"namespace"`
							} `json:"relatedAspectTable"`
						} `json:"node"`
					} `json:"edges"`
				} `json:"autoRelations"`
			} `json:"aspect"`
		} `json:"organization"`
	}

	err := c.ExecuteInto(ctx, query, map[string]interface{}{"aspectId": app.ID}, &result)
	if err != nil {
		return err
	}

	if result.Organization.Aspect == nil {
		return nil
	}

	app.Relationships = make([]Relationship, 0, len(result.Organization.Aspect.AutoRelations.Edges))
	for _, edge := range result.Organization.Aspect.AutoRelations.Edges {
		rel := Relationship{
			ID:                edge.Node.ID,
			RelatedObjectID:   edge.Node.RelatedAspectTable.ID,
			RelatedObjectType: mapAspectTypename(edge.Node.RelatedAspectTable.Typename),
			RelatedObjectName: edge.Node.RelatedAspectTable.Name,
			Filter:            edge.Node.Filter,
			Columns:           make([]RelationshipColumn, 0, len(edge.Node.Columns)),
		}

		for _, col := range edge.Node.Columns {
			rel.Columns = append(rel.Columns, RelationshipColumn{
				FieldID:          col.AspectField.ID,
				FieldName:        col.AspectField.Name,
				RelatedFieldID:   col.RelatedAspectTableField.ID,
				RelatedFieldName: col.RelatedAspectTableField.Name,
			})
		}

		app.Relationships = append(app.Relationships, rel)
	}

	return nil
}

// mapAspectTypename converts GraphQL typename to friendly type
func mapAspectTypename(typename string) string {
	switch typename {
	case "AspectApp":
		return "App"
	case "AspectElement":
		return "Element"
	case "AspectTask":
		return "Task"
	case "Table":
		return "Table"
	default:
		return "Unknown"
	}
}

// discoverRelatedObjects recursively discovers objects that are related through relationships
// It tracks visited objects to avoid infinite loops
func discoverRelatedObjects(ctx context.Context, c *client.Client, app *App, visited map[string]bool) error {
	if visited == nil {
		visited = make(map[string]bool)
	}

	// Mark the main app as visited
	visited[app.ID] = true

	// Collect all unique related object IDs that haven't been visited
	relatedToDiscover := make(map[string]*Relationship)
	for i := range app.Relationships {
		rel := &app.Relationships[i]
		if !visited[rel.RelatedObjectID] && (rel.RelatedObjectType == "App" || rel.RelatedObjectType == "Element") {
			relatedToDiscover[rel.RelatedObjectID] = rel
		}
	}

	if len(relatedToDiscover) == 0 {
		return nil
	}

	// Discover each related object
	for objID, rel := range relatedToDiscover {
		visited[objID] = true

		relatedObj, err := getRelatedObject(ctx, c, objID, rel.RelatedObjectType, rel.RelatedObjectName)
		if err != nil {
			// Log but don't fail - related object may be inaccessible
			continue
		}

		app.RelatedObjects = append(app.RelatedObjects, *relatedObj)

		// Recursively discover relationships from this object
		nestedRelationships, err := getObjectRelationships(ctx, c, objID)
		if err != nil {
			continue
		}

		// Add any new relationships and continue recursion
		for _, nestedRel := range nestedRelationships {
			if !visited[nestedRel.RelatedObjectID] && (nestedRel.RelatedObjectType == "App" || nestedRel.RelatedObjectType == "Element") {
				// Discover this nested relationship
				nestedObj, err := getRelatedObject(ctx, c, nestedRel.RelatedObjectID, nestedRel.RelatedObjectType, nestedRel.RelatedObjectName)
				if err != nil {
					continue
				}
				visited[nestedRel.RelatedObjectID] = true
				app.RelatedObjects = append(app.RelatedObjects, *nestedObj)

				// Continue recursion from the nested object
				err = discoverRelatedObjectsFrom(ctx, c, app, nestedRel.RelatedObjectID, visited)
				if err != nil {
					continue
				}
			}
		}
	}

	return nil
}

// discoverRelatedObjectsFrom continues recursive discovery from a specific object
func discoverRelatedObjectsFrom(ctx context.Context, c *client.Client, app *App, objectID string, visited map[string]bool) error {
	nestedRelationships, err := getObjectRelationships(ctx, c, objectID)
	if err != nil {
		return err
	}

	for _, rel := range nestedRelationships {
		if !visited[rel.RelatedObjectID] && (rel.RelatedObjectType == "App" || rel.RelatedObjectType == "Element") {
			visited[rel.RelatedObjectID] = true

			relatedObj, err := getRelatedObject(ctx, c, rel.RelatedObjectID, rel.RelatedObjectType, rel.RelatedObjectName)
			if err != nil {
				continue
			}

			app.RelatedObjects = append(app.RelatedObjects, *relatedObj)

			// Continue recursion
			err = discoverRelatedObjectsFrom(ctx, c, app, rel.RelatedObjectID, visited)
			if err != nil {
				continue
			}
		}
	}

	return nil
}

// getObjectRelationships retrieves relationships for any object (App/Element)
func getObjectRelationships(ctx context.Context, c *client.Client, objectID string) ([]Relationship, error) {
	query := `
		query GetObjectRelationships($aspectId: ID!) {
			organization {
				aspect(id: $aspectId) {
					id
					autoRelations {
						edges {
							node {
								id
								filter
								columns {
									aspectField {
										id
										name
									}
									relatedAspectTableField {
										__typename
										... on AspectField {
											id
											name
										}
										... on TableField {
											id
											name
										}
									}
								}
								relatedAspectTable {
									__typename
									... on AspectApp {
										id
										name
										namespace
									}
									... on AspectElement {
										id
										name
										namespace
									}
									... on AspectTask {
										id
										name
									}
									... on Table {
										id
										name
									}
								}
							}
						}
					}
				}
			}
		}
	`

	var result struct {
		Organization struct {
			Aspect *struct {
				AutoRelations struct {
					Edges []struct {
						Node struct {
							ID      string                 `json:"id"`
							Filter  map[string]interface{} `json:"filter"`
							Columns []struct {
								AspectField struct {
									ID   string `json:"id"`
									Name string `json:"name"`
								} `json:"aspectField"`
								RelatedAspectTableField struct {
									Typename string `json:"__typename"`
									ID       string `json:"id"`
									Name     string `json:"name"`
								} `json:"relatedAspectTableField"`
							} `json:"columns"`
							RelatedAspectTable struct {
								Typename  string `json:"__typename"`
								ID        string `json:"id"`
								Name      string `json:"name"`
								Namespace string `json:"namespace"`
							} `json:"relatedAspectTable"`
						} `json:"node"`
					} `json:"edges"`
				} `json:"autoRelations"`
			} `json:"aspect"`
		} `json:"organization"`
	}

	err := c.ExecuteInto(ctx, query, map[string]interface{}{"aspectId": objectID}, &result)
	if err != nil {
		return nil, err
	}

	if result.Organization.Aspect == nil {
		return nil, nil
	}

	relationships := make([]Relationship, 0, len(result.Organization.Aspect.AutoRelations.Edges))
	for _, edge := range result.Organization.Aspect.AutoRelations.Edges {
		rel := Relationship{
			ID:                edge.Node.ID,
			RelatedObjectID:   edge.Node.RelatedAspectTable.ID,
			RelatedObjectType: mapAspectTypename(edge.Node.RelatedAspectTable.Typename),
			RelatedObjectName: edge.Node.RelatedAspectTable.Name,
			Filter:            edge.Node.Filter,
			Columns:           make([]RelationshipColumn, 0, len(edge.Node.Columns)),
		}

		for _, col := range edge.Node.Columns {
			rel.Columns = append(rel.Columns, RelationshipColumn{
				FieldID:          col.AspectField.ID,
				FieldName:        col.AspectField.Name,
				RelatedFieldID:   col.RelatedAspectTableField.ID,
				RelatedFieldName: col.RelatedAspectTableField.Name,
			})
		}

		relationships = append(relationships, rel)
	}

	return relationships, nil
}

// getRelatedObject retrieves details about a related object
func getRelatedObject(ctx context.Context, c *client.Client, objectID, objectType, objectName string) (*RelatedObject, error) {
	query := `
		query GetRelatedObject($aspectId: ID!) {
			organization {
				aspect(id: $aspectId) {
					id
					name
					__typename
					... on AspectApp {
						namespace
					}
					... on AspectElement {
						namespace
					}
					fields {
						edges {
							node {
								id
								name
								__typename
								... on AspectField {
									semanticTags
								}
							}
						}
					}
				}
			}
		}
	`

	var result struct {
		Organization struct {
			Aspect *struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Typename  string `json:"__typename"`
				Namespace string `json:"namespace"`
				Fields    struct {
					Edges []struct {
						Node struct {
							ID           string   `json:"id"`
							Name         string   `json:"name"`
							Typename     string   `json:"__typename"`
							SemanticTags []string `json:"semanticTags"`
						} `json:"node"`
					} `json:"edges"`
				} `json:"fields"`
			} `json:"aspect"`
		} `json:"organization"`
	}

	err := c.ExecuteInto(ctx, query, map[string]interface{}{"aspectId": objectID}, &result)
	if err != nil {
		return nil, err
	}

	if result.Organization.Aspect == nil {
		return nil, fmt.Errorf("related object not found: %s", objectID)
	}

	aspect := result.Organization.Aspect
	relatedObj := &RelatedObject{
		ID:        aspect.ID,
		Name:      aspect.Name,
		Type:      mapAspectTypename(aspect.Typename),
		Namespace: aspect.Namespace,
		Fields:    make([]Field, 0, len(aspect.Fields.Edges)),
		FieldIDs:  make([]string, 0, len(aspect.Fields.Edges)),
	}

	for _, edge := range aspect.Fields.Edges {
		relatedObj.Fields = append(relatedObj.Fields, Field{
			ID:           edge.Node.ID,
			Name:         edge.Node.Name,
			Type:         mapFieldType(edge.Node.Typename),
			SemanticTags: edge.Node.SemanticTags,
		})
		relatedObj.FieldIDs = append(relatedObj.FieldIDs, edge.Node.ID)
	}

	return relatedObj, nil
}

// Helper functions to map GraphQL types to friendly names

func mapFieldType(typename string) string {
	result := fieldtypes.MapGraphQLFieldType(typename)
	// Normalize some names for CLI display
	if result == "bool" {
		return "boolean"
	}
	if result == "multi_select" {
		return "multiselect"
	}
	// If result equals typename, it wasn't mapped
	if result == typename {
		return "unknown"
	}
	return result
}

func mapTriggerType(typename string) string {
	// Use the shared registry from internal/client
	return client.MapTriggerTypename(typename)
}

func mapTaskType(typename string) string {
	// Use the shared registry from internal/client
	return client.MapTaskTypename(typename)
}

// determineTaskType maps GraphQL typename to terraform task type,
// with special handling for variable tasks that can be CREATE or UPDATE.
func determineTaskType(taskData map[string]interface{}) string {
	typename := getString(taskData, "__typename")
	baseType := mapTaskType(typename)

	// Special handling for variable tasks: check if it's CREATE or UPDATE
	if baseType == "variable" {
		if isUpdateVariableTask(taskData) {
			return "update_variable"
		}
	}

	return baseType
}

// isUpdateVariableTask checks if a variable task contains UPDATE operations
// (as opposed to CREATE operations).
func isUpdateVariableTask(taskData map[string]interface{}) bool {
	variables, ok := taskData["variables"].([]interface{})
	if !ok || len(variables) == 0 {
		return false
	}

	firstVar, ok := variables[0].(map[string]interface{})
	if !ok {
		return false
	}

	typename := getString(firstVar, "__typename")
	return typename == "WorkflowVariableTaskParameterUpdate"
}

func mapDisplayBlockType(typename string) string {
	switch typename {
	case "AspectGroupDisplayBlock":
		return "group"
	case "AspectFieldDisplayBlock":
		return "field"
	case "AspectWidgetDisplayBlock":
		return "widget"
	case "AspectActivityLogDisplayBlock":
		return "activity_log"
	case "AspectApprovalsDisplayBlock":
		return "approvals"
	case "AspectAttachmentDisplayBlock":
		return "attachments"
	case "AspectSurveysDisplayBlock":
		return "surveys"
	case "AspectStatusDisplayBlock":
		return "status"
	case "AspectCommentsDisplayBlock":
		return "comments"
	case "AspectRelatedItemsDisplayBlock":
		return "related_items"
	default:
		// Strip AspectDisplayBlock or AspectStageDisplayBlock prefix if present
		if strings.HasPrefix(typename, "Aspect") && strings.HasSuffix(typename, "DisplayBlock") {
			name := strings.TrimPrefix(typename, "Aspect")
			name = strings.TrimSuffix(name, "DisplayBlock")
			return strings.ToLower(name)
		}
		return typename
	}
}

// discoverTaskAndToolRelatedObjects discovers external apps/elements referenced by automation tasks
// and agent tools, adding them to RelatedObjects for data source generation.
// This ensures that cross-app references are properly resolved even without --recursive flag.
func discoverTaskAndToolRelatedObjects(ctx context.Context, c *client.Client, app *App) error {
	// Track objects we've already added to avoid duplicates
	seen := make(map[string]bool)

	// Also skip objects that are already in RelatedObjects
	for _, obj := range app.RelatedObjects {
		seen[obj.ID] = true
	}

	// Helper to add a related object if not already seen
	addRelatedObject := func(objectID string) {
		if objectID == "" || objectID == app.ID || seen[objectID] {
			return
		}
		// Use empty strings for type/name - getRelatedObject fetches actual values from API
		obj, err := getRelatedObject(ctx, c, objectID, "", "")
		if err == nil && obj != nil {
			app.RelatedObjects = append(app.RelatedObjects, *obj)
			seen[objectID] = true
		}
	}

	// Discover from automation tasks
	for _, automation := range app.Automations {
		for _, task := range automation.Tasks {
			// ObjectID from aspect (record_search, create_record, update_field, etc.)
			addRelatedObject(task.ObjectID)
			// RelatedObjectID from relatedAspect (find_related_records)
			addRelatedObject(task.RelatedObjectID)
			// DynamicCategoryAspectID from ai_classify task
			addRelatedObject(task.DynamicCategoryAspectID)
		}
	}

	// Discover from agent tools
	for _, agent := range app.Agents {
		for _, tool := range agent.Tools {
			// AspectID from various tools (CreateRecord, SearchAspect, UpdateRecord, etc.)
			addRelatedObject(tool.AspectID)
			// RelatedAspectID from RelateRecordTool
			addRelatedObject(tool.RelatedAspectID)
		}
	}

	return nil
}
