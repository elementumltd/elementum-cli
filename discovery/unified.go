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
	"fmt"
	"sync"

	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/elementumltd/elementum-cli/internal/client"
)

// DiscoverAllDependencies performs unified recursive discovery from any root object.
// It discovers all related objects through relationships, automation tasks, automation triggers,
// and agent tools, and returns a context containing all discovered objects.
// If ec is provided, errors are recorded to the collector for structured reporting.
func DiscoverAllDependencies(ctx context.Context, c *client.Client, rootID, rootType string, ec ...*ErrorCollector) (*UnifiedDiscoveryContext, error) {
	uCtx := NewUnifiedDiscoveryContext(rootID, rootType)

	// Use the error collector if provided
	var collector *ErrorCollector
	if len(ec) > 0 && ec[0] != nil {
		collector = ec[0]
	}

	// Seed the queue based on root type
	switch rootType {
	case "App":
		app, err := GetApp(ctx, c, rootID, collector)
		if err != nil {
			return nil, err
		}
		queueFromAspect(app, uCtx)
	case "Element":
		element, err := GetElementFull(ctx, c, rootID)
		if err != nil {
			return nil, err
		}
		queueFromAspect(element, uCtx)
	}

	// Process queue until empty
	if err := ProcessDiscoveryQueue(ctx, c, uCtx, collector); err != nil {
		return uCtx, err
	}

	return uCtx, nil
}

// DiscoverAllDependenciesParallel is a parallel version of DiscoverAllDependencies.
// It uses parallel discovery for the root object and parallel BFS queue processing.
func DiscoverAllDependenciesParallel(ctx context.Context, c *client.Client, rootID, rootType string, config *ParallelConfig, ec ...*ErrorCollector) (*UnifiedDiscoveryContext, error) {
	uCtx := NewUnifiedDiscoveryContext(rootID, rootType)

	var collector *ErrorCollector
	if len(ec) > 0 && ec[0] != nil {
		collector = ec[0]
	}

	// Seed the queue based on root type - use parallel versions
	switch rootType {
	case "App":
		app, err := GetAppParallel(ctx, c, rootID, config, collector)
		if err != nil {
			return nil, err
		}
		queueFromAspect(app, uCtx)
	case "Element":
		element, err := GetElementFull(ctx, c, rootID)
		if err != nil {
			return nil, err
		}
		queueFromAspect(element, uCtx)
	}

	// Process queue in parallel
	if err := ProcessDiscoveryQueueParallel(ctx, c, uCtx, config, collector); err != nil {
		return uCtx, err
	}

	return uCtx, nil
}

// ProcessDiscoveryQueue processes all items in the discovery queue.
// If ec is provided, errors are classified and recorded. Fatal errors cause the queue
// processing to abort. Warning-level errors (e.g., "item does not exist") are recorded
// and processing continues.
func ProcessDiscoveryQueue(ctx context.Context, c *client.Client, uCtx *UnifiedDiscoveryContext, ec ...*ErrorCollector) error {
	var collector *ErrorCollector
	if len(ec) > 0 && ec[0] != nil {
		collector = ec[0]
	}

	// handleDiscoveryError classifies an error and records it. Returns true if processing should abort.
	handleDiscoveryError := func(itemType, itemID string, err error) bool {
		if err == nil {
			return false
		}
		severity := ClassifyError(err)
		if collector != nil {
			collector.Add(severity, "recursive_discovery", fmt.Sprintf("%s %s", itemType, itemID), err)
			return severity == SeverityFatal
		}
		// Legacy mode: log and continue (preserve old behavior when no collector)
		logger.Warn(fmt.Sprintf("skipping inaccessible %s", itemType), "id", itemID, "error", err)
		return false
	}

	for {
		item := uCtx.Dequeue()
		if item == nil {
			break
		}

		if uCtx.IsVisited(item.ID) {
			continue
		}
		uCtx.MarkVisited(item.ID)

		// Resolve the actual aspect type when the queued type might be wrong.
		// Relationships and automation tasks often queue everything as "App" even
		// when the target is an Element or Task. A fast type-check avoids wasted
		// calls and misleading errors.
		resolvedType := item.Type
		if item.Type == "App" || item.Type == "Element" || item.Type == "Task" {
			if actualType, err := ResolveAspectType(ctx, c, item.ID); err == nil && actualType != "" {
				if actualType != item.Type {
					logger.Debug("resolved aspect type differs from queued type",
						"id", item.ID, "queued", item.Type, "actual", actualType)
				}
				resolvedType = actualType
			}
			// If resolve fails, fall through with the original type
		}

		switch resolvedType {
		case "App":
			app, err := GetApp(ctx, c, item.ID, collector)
			if err != nil {
				if handleDiscoveryError("app", item.ID, err) {
					return fmt.Errorf("fatal error discovering app %s: %w", item.ID, err)
				}
				continue
			}
			uCtx.Apps[item.ID] = app
			fmt.Printf("  %s Discovered app: %s\n", ui.MutedStyle.Render("→"), ui.InfoStyle.Render(app.Name))
			queueFromAspect(app, uCtx)

		case "Task":
			task, err := GetTaskFull(ctx, c, item.ID)
			if err != nil {
				if handleDiscoveryError("task", item.ID, err) {
					return fmt.Errorf("fatal error discovering task %s: %w", item.ID, err)
				}
				continue
			}
			uCtx.Tasks[item.ID] = task
			fmt.Printf("  %s Discovered task: %s\n", ui.MutedStyle.Render("→"), ui.InfoStyle.Render(task.Name))
			queueFromAspect(task, uCtx)

		case "Element":
			element, err := GetElementFull(ctx, c, item.ID)
			if err != nil {
				if handleDiscoveryError("element", item.ID, err) {
					return fmt.Errorf("fatal error discovering element %s: %w", item.ID, err)
				}
				continue
			}
			uCtx.Elements[item.ID] = element
			fmt.Printf("  %s Discovered element: %s\n", ui.MutedStyle.Render("→"), ui.InfoStyle.Render(element.Name))
			queueFromAspect(element, uCtx)

		case "Table":
			table, err := GetTable(ctx, c, item.ID)
			if err != nil {
				if handleDiscoveryError("table", item.ID, err) {
					return fmt.Errorf("fatal error discovering table %s: %w", item.ID, err)
				}
				continue
			}
			uCtx.Tables[item.ID] = table
			fmt.Printf("  %s Discovered table: %s\n", ui.MutedStyle.Render("→"), ui.InfoStyle.Render(table.Name))

			if table.SourceID != "" {
				uCtx.Enqueue(table.SourceID, "Table")
			}
			if table.CloudLinkID != "" {
				uCtx.Enqueue(table.CloudLinkID, "CloudLink")
			}

		case "Datamine":
			datamine, table, err := findDatamineByID(ctx, c, item.ID)
			if err != nil {
				if handleDiscoveryError("datamine", item.ID, err) {
					return fmt.Errorf("fatal error discovering datamine %s: %w", item.ID, err)
				}
				continue
			}
			uCtx.Datamines[item.ID] = datamine
			fmt.Printf("  %s Discovered datamine: %s\n", ui.MutedStyle.Render("→"), ui.InfoStyle.Render(datamine.Name))
			if table != nil && table.ID != "" && !uCtx.IsVisited(table.ID) {
				uCtx.MarkVisited(table.ID)
				uCtx.Tables[table.ID] = table
				fmt.Printf("  %s Discovered table: %s\n", ui.MutedStyle.Render("→"), ui.InfoStyle.Render(table.Name))

				if table.SourceID != "" {
					uCtx.Enqueue(table.SourceID, "Table")
				}
				if table.CloudLinkID != "" {
					uCtx.Enqueue(table.CloudLinkID, "CloudLink")
				}
			}

		case "CloudLink":
			cloudlink, err := GetCloudLink(ctx, c, item.ID)
			if err != nil {
				if handleDiscoveryError("cloudlink", item.ID, err) {
					return fmt.Errorf("fatal error discovering cloudlink %s: %w", item.ID, err)
				}
				continue
			}
			uCtx.CloudLinks[item.ID] = cloudlink
			fmt.Printf("  %s Discovered cloudlink: %s\n", ui.MutedStyle.Render("→"), ui.InfoStyle.Render(cloudlink.Name))
		}
	}
	return nil
}

// ResolveAspectType makes a lightweight GraphQL call to determine the actual type of an aspect.
// Returns one of: "App", "Element", "Task", or "" if not found.
func ResolveAspectType(ctx context.Context, c *client.Client, aspectID string) (string, error) {
	query := `
		query ResolveAspectType($aspectId: ID!) {
			organization {
				aspect(id: $aspectId) {
					__typename
				}
			}
		}
	`
	var result struct {
		Organization struct {
			Aspect *struct {
				Typename string `json:"__typename"`
			} `json:"aspect"`
		} `json:"organization"`
	}
	err := c.ExecuteInto(ctx, query, map[string]interface{}{"aspectId": aspectID}, &result)
	if err != nil {
		return "", err
	}
	if result.Organization.Aspect == nil {
		return "", fmt.Errorf("aspect not found: %s", aspectID)
	}
	return mapAspectTypename(result.Organization.Aspect.Typename), nil
}

// DiscoverableAspect is implemented by types that can be queued for dependency discovery.
// All aspect types (App, Element, AspectTask) share relationships and automations;
// agents are optional (only App and AspectTask support them).
type DiscoverableAspect interface {
	GetAspectID() string
	GetRelationships() []Relationship
	GetAutomations() []Automation
	GetAgents() []Agent // Returns nil for types that don't support agents (e.g., Element)
}

// queueFromAspect extracts all discoverable references from any aspect type and adds them to the queue.
// This unifies the previously duplicated queueFromApp/queueFromTask/queueFromElement functions.
func queueFromAspect(aspect DiscoverableAspect, uCtx *UnifiedDiscoveryContext) {
	aspectID := aspect.GetAspectID()

	// 1. Relationships -> target App, Element, Task, or Table
	for _, rel := range aspect.GetRelationships() {
		uCtx.Enqueue(rel.RelatedObjectID, rel.RelatedObjectType)
	}

	// 2. Automation triggers (datamines) and tasks (cross-app references)
	for _, automation := range aspect.GetAutomations() {
		// Check triggers for datamine references
		for _, trigger := range automation.Triggers {
			if trigger.Type == "datamine" && trigger.DatamineID != "" {
				uCtx.Enqueue(trigger.DatamineID, "Datamine")
			}
		}

		// Check tasks for cross-app/element references
		for _, task := range automation.Tasks {
			// ObjectID from aspect (record_search, create_record, update_field, aspect_record_field_locking, bulk_excel)
			if task.ObjectID != "" && task.ObjectID != aspectID {
				uCtx.Enqueue(task.ObjectID, "App")
			}
			// RelatedObjectID from relatedAspect (find_related_records)
			if task.RelatedObjectID != "" && task.RelatedObjectID != aspectID {
				uCtx.Enqueue(task.RelatedObjectID, "App")
			}
			// DynamicCategoryAspectID from ai_classify task's dynamic_category_source.aspect
			if task.DynamicCategoryAspectID != "" && task.DynamicCategoryAspectID != aspectID {
				uCtx.Enqueue(task.DynamicCategoryAspectID, "App")
			}
		}
	}

	// 3. Agent tools -> aspect references, table references
	// Returns nil for types that don't support agents (e.g., Element)
	for _, agent := range aspect.GetAgents() {
		for _, tool := range agent.Tools {
			// Target aspect (CreateRecord, SearchAspect, UpdateRecord, SearchTable tools)
			if tool.AspectID != "" && tool.AspectID != aspectID {
				uCtx.Enqueue(tool.AspectID, "App")
			}
			// Related aspect (RelateRecordTool)
			if tool.RelatedAspectID != "" && tool.RelatedAspectID != aspectID {
				uCtx.Enqueue(tool.RelatedAspectID, "App")
			}
			// AI Search Table (SearchTableTool) - queue the search table's aspect, not the table itself
			// SearchTables live on Elements (NOT Apps, and NOT global organization Tables)
			// The aspect discovery will pull down its AI search tables via GetAISearchTablesForAspect
			if tool.SearchTableAspectID != "" && tool.SearchTableAspectID != aspectID {
				aspectType := tool.SearchTableAspectType
				if aspectType == "" {
					aspectType = "Element" // Default to Element (only type with SearchTables)
				}
				uCtx.Enqueue(tool.SearchTableAspectID, aspectType)
			}
		}
	}
}

// ProcessDiscoveryQueueParallel uses a worker pool to process discovery items.
// Workers continuously pull items from the queue, allowing newly discovered items
// to be processed immediately by idle workers (pipeline processing).
func ProcessDiscoveryQueueParallel(ctx context.Context, c *client.Client, uCtx *UnifiedDiscoveryContext, config *ParallelConfig, ec ...*ErrorCollector) error {
	var collector *ErrorCollector
	if len(ec) > 0 && ec[0] != nil {
		collector = ec[0]
	}

	executor := NewParallelExecutor(config)
	numWorkers := int(executor.GetMaxConcurrency())

	// Shared state for workers
	var (
		fatalErr   error
		fatalMu    sync.Mutex
		activeJobs sync.WaitGroup
	)

	// handleError classifies and records errors
	handleError := func(itemType, itemID string, err error) bool {
		if err == nil {
			return false
		}
		severity := ClassifyError(err)
		if collector != nil {
			collector.Add(severity, "recursive_discovery", fmt.Sprintf("%s %s", itemType, itemID), err)
			return severity == SeverityFatal
		}
		logger.Warn(fmt.Sprintf("skipping inaccessible %s", itemType), "id", itemID, "error", err)
		return false
	}

	// processItem handles a single discovery item
	processItem := func(item DiscoveryItem) {
		defer activeJobs.Done()

		// Check if already visited
		if uCtx.IsVisitedSafe(item.ID) {
			return
		}
		uCtx.MarkVisitedSafe(item.ID)

		// Resolve actual type for aspects that might be misclassified
		resolvedType := item.Type
		if item.Type == "App" || item.Type == "Element" || item.Type == "Task" {
			if actualType, err := ResolveAspectType(ctx, c, item.ID); err == nil && actualType != "" {
				if actualType != item.Type {
					logger.Debug("resolved aspect type", "id", item.ID, "queued", item.Type, "actual", actualType)
				}
				resolvedType = actualType
			}
		}

		var err error
		var name string

		switch resolvedType {
		case "App":
			app, e := GetAppParallel(ctx, c, item.ID, config, collector)
			err = e
			if app != nil {
				name = app.Name
				uCtx.StoreAppSafe(item.ID, app)
				queueFromAspectSafe(app, uCtx)
			}

		case "Task":
			task, e := GetTaskFull(ctx, c, item.ID)
			err = e
			if task != nil {
				name = task.Name
				uCtx.mu.Lock()
				uCtx.Tasks[item.ID] = task
				uCtx.mu.Unlock()
				queueFromAspectSafe(task, uCtx)
			}

		case "Element":
			element, e := GetElementFull(ctx, c, item.ID)
			err = e
			if element != nil {
				name = element.Name
				uCtx.StoreElementSafe(item.ID, element)
				queueFromAspectSafe(element, uCtx)
			}

		case "Table":
			table, e := GetTable(ctx, c, item.ID)
			err = e
			if table != nil {
				name = table.Name
				uCtx.StoreTableSafe(item.ID, table)
				if table.SourceID != "" {
					uCtx.EnqueueSafe(table.SourceID, "Table")
				}
				if table.CloudLinkID != "" {
					uCtx.EnqueueSafe(table.CloudLinkID, "CloudLink")
				}
			}

		case "Datamine":
			datamine, table, e := findDatamineByID(ctx, c, item.ID)
			err = e
			if datamine != nil {
				name = datamine.Name
				uCtx.StoreDatamineSafe(item.ID, datamine)
				if table != nil && table.ID != "" && !uCtx.IsVisitedSafe(table.ID) {
					uCtx.MarkVisitedSafe(table.ID)
					uCtx.StoreTableSafe(table.ID, table)
					if table.SourceID != "" {
						uCtx.EnqueueSafe(table.SourceID, "Table")
					}
					if table.CloudLinkID != "" {
						uCtx.EnqueueSafe(table.CloudLinkID, "CloudLink")
					}
				}
			}

		case "CloudLink":
			cloudlink, e := GetCloudLink(ctx, c, item.ID)
			err = e
			if cloudlink != nil {
				name = cloudlink.Name
				uCtx.StoreCloudLinkSafe(item.ID, cloudlink)
			}
		}

		// Handle errors
		if err != nil {
			if handleError(resolvedType, item.ID, err) {
				fatalMu.Lock()
				if fatalErr == nil {
					fatalErr = fmt.Errorf("fatal error discovering %s %s: %w", resolvedType, item.ID, err)
				}
				fatalMu.Unlock()
			}
			return
		}

		// Print discovery message
		if name != "" {
			fmt.Printf("  %s Discovered %s: %s\n",
				ui.MutedStyle.Render("→"),
				resolvedType,
				ui.InfoStyle.Render(name))
		}
	}

	// Work distribution channel
	workChan := make(chan DiscoveryItem, numWorkers*2)
	var workersDone sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		workersDone.Add(1)
		go func() {
			defer workersDone.Done()
			for item := range workChan {
				processItem(item)
			}
		}()
	}

	// Dispatcher: continuously pull items from queue and send to workers
	// Exits when queue is empty AND no active jobs (nothing can add more items)
	go func() {
		for {
			// Try to get an item
			item := uCtx.Dequeue()
			if item != nil {
				activeJobs.Add(1)
				workChan <- *item
				continue
			}

			// Queue is empty - check if any jobs are still active
			// If active jobs exist, they might add more items, so wait
			activeJobs.Wait()

			// Double-check queue after waiting
			item = uCtx.Dequeue()
			if item != nil {
				activeJobs.Add(1)
				workChan <- *item
				continue
			}

			// Queue is truly empty and no active jobs
			break
		}
		close(workChan)
	}()

	// Wait for all workers to finish
	workersDone.Wait()

	return fatalErr
}

// queueFromAspectSafe is a thread-safe version of queueFromAspect
func queueFromAspectSafe(aspect DiscoverableAspect, uCtx *UnifiedDiscoveryContext) {
	aspectID := aspect.GetAspectID()

	// 1. Relationships -> target App, Element, Task, or Table
	for _, rel := range aspect.GetRelationships() {
		uCtx.EnqueueSafe(rel.RelatedObjectID, rel.RelatedObjectType)
	}

	// 2. Automation triggers (datamines) and tasks (cross-app references)
	for _, automation := range aspect.GetAutomations() {
		for _, trigger := range automation.Triggers {
			if trigger.Type == "datamine" && trigger.DatamineID != "" {
				uCtx.EnqueueSafe(trigger.DatamineID, "Datamine")
			}
		}

		for _, task := range automation.Tasks {
			if task.ObjectID != "" && task.ObjectID != aspectID {
				uCtx.EnqueueSafe(task.ObjectID, "App")
			}
			if task.RelatedObjectID != "" && task.RelatedObjectID != aspectID {
				uCtx.EnqueueSafe(task.RelatedObjectID, "App")
			}
			if task.DynamicCategoryAspectID != "" && task.DynamicCategoryAspectID != aspectID {
				uCtx.EnqueueSafe(task.DynamicCategoryAspectID, "App")
			}
		}
	}

	// 3. Agent tools -> aspect references, table references
	for _, agent := range aspect.GetAgents() {
		for _, tool := range agent.Tools {
			if tool.AspectID != "" && tool.AspectID != aspectID {
				uCtx.EnqueueSafe(tool.AspectID, "App")
			}
			if tool.RelatedAspectID != "" && tool.RelatedAspectID != aspectID {
				uCtx.EnqueueSafe(tool.RelatedAspectID, "App")
			}
			if tool.SearchTableAspectID != "" && tool.SearchTableAspectID != aspectID {
				aspectType := tool.SearchTableAspectType
				if aspectType == "" {
					aspectType = "Element"
				}
				uCtx.EnqueueSafe(tool.SearchTableAspectID, aspectType)
			}
		}
	}
}
