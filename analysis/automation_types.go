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

package analysis

import (
	"fmt"

	"github.com/elementumltd/elementum-cli/internal/client"
)

// AutomationConfig is the top-level structure for rendering an automation's
// configuration as a top-to-bottom flow of trigger → tasks.
type AutomationConfig struct {
	AutomationID   string            `json:"automationId"`
	AutomationName string            `json:"automationName"`
	Status         string            `json:"status"`
	WorkflowID     string            `json:"workflowId"`
	Version        int               `json:"version"`
	Trigger        *AutomationNode   `json:"trigger,omitempty"`
	Tasks          []*AutomationNode `json:"tasks"`
	Outputs        []string          `json:"outputs,omitempty"`
	Summary        AutomationSummary `json:"summary"`
}

// AutomationNode represents a single element in the automation flow.
// It can be a trigger, a regular task, a switch (with cases), or a for_each (with body).
type AutomationNode struct {
	ID       string             `json:"id"`
	Name     string             `json:"name"`
	Type     string             `json:"type"`
	NodeKind AutomationNodeKind `json:"nodeKind"`
	Index    string             `json:"index"`
	Details  map[string]any     `json:"details,omitempty"`

	// Switch-specific: cases with their nested task chains
	Cases []*SwitchCase `json:"cases,omitempty"`

	// ForEach-specific: the loop body task chain
	LoopList string            `json:"loopList,omitempty"`
	LoopBody []*AutomationNode `json:"loopBody,omitempty"`
}

// SwitchCase represents a single case branch inside a switch task.
type SwitchCase struct {
	ID     string            `json:"id"`
	Label  string            `json:"label"`
	Filter map[string]any    `json:"filter,omitempty"`
	Tasks  []*AutomationNode `json:"tasks"`
}

// AutomationNodeKind describes what kind of node this is in the flow.
type AutomationNodeKind string

const (
	NodeTrigger AutomationNodeKind = "trigger"
	NodeTask    AutomationNodeKind = "task"
	NodeSwitch  AutomationNodeKind = "switch"
	NodeForEach AutomationNodeKind = "for_each"
)

// AutomationSummary provides aggregate counts for the automation.
type AutomationSummary struct {
	TriggerType   string `json:"triggerType"`
	TopLevelTasks int    `json:"topLevelTasks"`
	TotalTasks    int    `json:"totalTasks"`
	SwitchCount   int    `json:"switchCount"`
	ForEachCount  int    `json:"forEachCount"`
	TotalCases    int    `json:"totalCases"`
	OutputCount   int    `json:"outputCount"`
}

// BuildAutomationConfig parses a WorkflowFullDetails into a structured
// AutomationConfig suitable for rendering.
func BuildAutomationConfig(
	automationID, automationName, status string,
	wf *client.WorkflowFullDetails,
) *AutomationConfig {
	cfg := &AutomationConfig{
		AutomationID:   automationID,
		AutomationName: automationName,
		Status:         status,
		WorkflowID:     wf.ID,
		Version:        wf.Version,
		Tasks:          []*AutomationNode{},
	}

	for _, o := range wf.Outputs {
		cfg.Outputs = append(cfg.Outputs, o.Name)
	}

	// Build trigger node
	if len(wf.Triggers) > 0 {
		t := wf.Triggers[0]
		triggerType := client.GetTriggerTypeLabel(t.Typename)
		cfg.Trigger = &AutomationNode{
			ID:       t.ID,
			Name:     triggerType,
			Type:     triggerType,
			NodeKind: NodeTrigger,
			Details:  filterTriggerDetails(t.RawData),
		}
	}

	// Parse flat task list into tree structure
	cfg.Tasks = buildTaskTree(wf.Tasks)

	// Compute summary
	cfg.Summary = computeSummary(cfg)

	return cfg
}

// BuildAutomationConfigFromGenqlient creates an AutomationConfig from the
// genqlient-generated GetWorkflowForVisualization response. This properly
// handles nested tasks inside switch cases and for_each loops.
func BuildAutomationConfigFromGenqlient(
	automationID, automationName, status string,
	wf *client.GetWorkflowForVisualizationOrganizationWorkflow,
) *AutomationConfig {
	cfg := &AutomationConfig{
		AutomationID:   automationID,
		AutomationName: automationName,
		Status:         status,
		WorkflowID:     wf.Id,
		Version:        wf.Version,
		Tasks:          []*AutomationNode{},
	}

	for _, o := range wf.Outputs {
		cfg.Outputs = append(cfg.Outputs, o.Name)
	}

	// Build trigger node
	if len(wf.Triggers) > 0 {
		t := wf.Triggers[0]
		triggerType := mapTriggerTypename(t.GetTypename())
		cfg.Trigger = &AutomationNode{
			ID:       t.GetId(),
			Name:     triggerType,
			Type:     triggerType,
			NodeKind: NodeTrigger,
		}
	}

	// Build task tree from genqlient types
	cfg.Tasks = buildTaskTreeFromGenqlient(wf.Tasks)

	// Compute summary
	cfg.Summary = computeSummary(cfg)

	return cfg
}

// mapTriggerTypename converts a GraphQL typename to a user-friendly trigger type
func mapTriggerTypename(typename *string) string {
	if typename == nil {
		return "unknown"
	}
	switch *typename {
	case "WorkflowRecordCreateTrigger":
		return "record_created"
	case "WorkflowRecordUpdateTrigger":
		return "record_updated"
	case "WorkflowOnDemandTrigger":
		return "on_demand"
	case "WorkflowWebhookTrigger":
		return "webhook"
	case "WorkflowTemporalTrigger":
		return "temporal"
	case "WorkflowApprovalChainTrigger":
		return "approval_chain"
	case "WorkflowDatamineTrigger":
		return "datamine"
	default:
		return *typename
	}
}

// mapTaskTypename converts a GraphQL typename to a user-friendly task type
func mapTaskTypename(typename *string) string {
	if typename == nil {
		return "unknown"
	}
	// Use the existing client function if available
	return client.MapTaskTypename(*typename)
}

// buildTaskTreeFromGenqlient converts genqlient task types to AutomationNodes
// with proper nesting for switch and for_each.
func buildTaskTreeFromGenqlient(tasks []client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowTask) []*AutomationNode {
	if len(tasks) == 0 {
		return nil
	}

	// Build a map of tasks by ID for ordering
	taskByID := make(map[string]client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowTask)
	for _, t := range tasks {
		taskByID[t.GetId()] = t
	}

	// Find root tasks (those with no previous or previous is trigger/not in list)
	var rootTasks []client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowTask
	for _, t := range tasks {
		prev := t.GetPrevious()
		if prev == nil {
			rootTasks = append(rootTasks, t)
		} else if taskByID[(*prev).GetId()] == nil {
			rootTasks = append(rootTasks, t)
		}
	}

	// Order tasks by following next pointers from roots
	ordered := orderTasksFromGenqlient(rootTasks, taskByID)

	// Convert to nodes
	counter := &indexCounter{}
	var nodes []*AutomationNode
	for _, t := range ordered {
		node := convertTaskToNode(t, counter)
		nodes = append(nodes, node)
	}

	return nodes
}

// orderTasksFromGenqlient orders tasks by following the next chain
func orderTasksFromGenqlient(roots []client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowTask, taskByID map[string]client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowTask) []client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowTask {
	seen := make(map[string]bool)
	var ordered []client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowTask

	for _, root := range roots {
		current := root
		for current != nil && !seen[current.GetId()] {
			seen[current.GetId()] = true
			ordered = append(ordered, current)

			if next := current.GetNext(); next != nil {
				current = taskByID[(*next).GetId()]
			} else {
				current = nil
			}
		}
	}

	return ordered
}

// convertTaskToNode converts a genqlient task to an AutomationNode
func convertTaskToNode(t client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowTask, counter *indexCounter) *AutomationNode {
	taskType := mapTaskTypename(t.GetTypename())
	name := ""
	if n := t.GetName(); n != nil {
		name = *n
	}

	node := &AutomationNode{
		ID:       t.GetId(),
		Name:     name,
		Type:     taskType,
		NodeKind: NodeTask,
		Index:    counter.next(),
	}

	// Handle switch tasks
	if switchTask, ok := t.(*client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowSwitchTask); ok {
		node.NodeKind = NodeSwitch
		for _, child := range switchTask.Children {
			label := ""
			if child.Label != nil {
				label = *child.Label
			}
			sc := &SwitchCase{
				ID:    child.Id,
				Label: label,
				Tasks: convertSwitchCaseTasksL1(child.Tasks, counter.sub(node.Index)),
			}
			node.Cases = append(node.Cases, sc)
		}
	}

	// Handle for_each tasks
	if forEachTask, ok := t.(*client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowForEachTask); ok {
		node.NodeKind = NodeForEach
		if forEachTask.ForEach.Label != nil {
			node.LoopList = *forEachTask.ForEach.Label
		}
		for _, child := range forEachTask.Children {
			node.LoopBody = append(node.LoopBody, convertForEachBodyTasksL1(child.Tasks, counter.sub(node.Index))...)
		}
	}

	return node
}

// Helper to get ID from CLITaskBasicPreviousWorkflowTask pointer
func getPrevID(prev *client.CLITaskBasicPreviousWorkflowTask) string {
	if prev == nil {
		return ""
	}
	return (*prev).GetId()
}

// Helper to get ID from CLITaskBasicNextWorkflowTask pointer
func getNextID(next *client.CLITaskBasicNextWorkflowTask) string {
	if next == nil {
		return ""
	}
	return (*next).GetId()
}

// convertSwitchCaseTasksL1 converts tasks inside a switch case (level 1)
func convertSwitchCaseTasksL1(tasks []client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowSwitchTaskChildrenWorkflowTaskChildTasksWorkflowTask, counter *indexCounter) []*AutomationNode {
	if len(tasks) == 0 {
		return nil
	}

	// Build ordered list by following next chain
	ordered := orderSwitchCaseTasksL1(tasks)

	var nodes []*AutomationNode
	for _, t := range ordered {
		taskType := mapTaskTypename(t.GetTypename())
		name := ""
		if n := t.GetName(); n != nil {
			name = *n
		}

		node := &AutomationNode{
			ID:       t.GetId(),
			Name:     name,
			Type:     taskType,
			NodeKind: NodeTask,
			Index:    counter.next(),
		}

		// Handle nested switch at level 2
		if switchTask, ok := t.(*client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowSwitchTaskChildrenWorkflowTaskChildTasksWorkflowSwitchTask); ok {
			node.NodeKind = NodeSwitch
			for _, child := range switchTask.Children {
				label := ""
				if child.Label != nil {
					label = *child.Label
				}
				sc := &SwitchCase{
					ID:    child.Id,
					Label: label,
					Tasks: convertL2SwitchInSwitchTasks(child.Tasks, counter.sub(node.Index)),
				}
				node.Cases = append(node.Cases, sc)
			}
		}

		// Handle nested for_each at level 2
		if forEachTask, ok := t.(*client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowSwitchTaskChildrenWorkflowTaskChildTasksWorkflowForEachTask); ok {
			node.NodeKind = NodeForEach
			if forEachTask.ForEach.Label != nil {
				node.LoopList = *forEachTask.ForEach.Label
			}
			for _, child := range forEachTask.Children {
				node.LoopBody = append(node.LoopBody, convertL2ForEachInSwitchTasks(child.Tasks, counter.sub(node.Index))...)
			}
		}

		nodes = append(nodes, node)
	}

	return nodes
}

// orderSwitchCaseTasksL1 orders tasks by following the chain
func orderSwitchCaseTasksL1(tasks []client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowSwitchTaskChildrenWorkflowTaskChildTasksWorkflowTask) []client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowSwitchTaskChildrenWorkflowTaskChildTasksWorkflowTask {
	if len(tasks) == 0 {
		return nil
	}

	taskByID := make(map[string]client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowSwitchTaskChildrenWorkflowTaskChildTasksWorkflowTask)
	for _, t := range tasks {
		taskByID[t.GetId()] = t
	}

	// Find root (no previous in task list)
	var root client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowSwitchTaskChildrenWorkflowTaskChildTasksWorkflowTask
	for _, t := range tasks {
		prevID := getPrevID(t.GetPrevious())
		if prevID == "" || taskByID[prevID] == nil {
			root = t
			break
		}
	}

	// Follow chain
	var ordered []client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowSwitchTaskChildrenWorkflowTaskChildTasksWorkflowTask
	seen := make(map[string]bool)
	current := root
	for current != nil && !seen[current.GetId()] {
		seen[current.GetId()] = true
		ordered = append(ordered, current)
		nextID := getNextID(current.GetNext())
		if nextID != "" {
			current = taskByID[nextID]
		} else {
			current = nil
		}
	}

	return ordered
}

// convertForEachBodyTasksL1 converts tasks inside a for_each body (level 1)
func convertForEachBodyTasksL1(tasks []client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowForEachTaskChildrenWorkflowTaskChildTasksWorkflowTask, counter *indexCounter) []*AutomationNode {
	if len(tasks) == 0 {
		return nil
	}

	// Build ordered list by following next chain
	ordered := orderForEachBodyTasksL1(tasks)

	var nodes []*AutomationNode
	for _, t := range ordered {
		taskType := mapTaskTypename(t.GetTypename())
		name := ""
		if n := t.GetName(); n != nil {
			name = *n
		}

		node := &AutomationNode{
			ID:       t.GetId(),
			Name:     name,
			Type:     taskType,
			NodeKind: NodeTask,
			Index:    counter.next(),
		}

		// Handle nested switch at level 2
		if switchTask, ok := t.(*client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowForEachTaskChildrenWorkflowTaskChildTasksWorkflowSwitchTask); ok {
			node.NodeKind = NodeSwitch
			for _, child := range switchTask.Children {
				label := ""
				if child.Label != nil {
					label = *child.Label
				}
				sc := &SwitchCase{
					ID:    child.Id,
					Label: label,
					Tasks: convertL2SwitchInForEachTasks(child.Tasks, counter.sub(node.Index)),
				}
				node.Cases = append(node.Cases, sc)
			}
		}

		// Handle nested for_each at level 2
		if forEachTask, ok := t.(*client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowForEachTaskChildrenWorkflowTaskChildTasksWorkflowForEachTask); ok {
			node.NodeKind = NodeForEach
			if forEachTask.ForEach.Label != nil {
				node.LoopList = *forEachTask.ForEach.Label
			}
			for _, child := range forEachTask.Children {
				node.LoopBody = append(node.LoopBody, convertL2ForEachInForEachTasks(child.Tasks, counter.sub(node.Index))...)
			}
		}

		nodes = append(nodes, node)
	}

	return nodes
}

// orderForEachBodyTasksL1 orders tasks by following the chain
func orderForEachBodyTasksL1(tasks []client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowForEachTaskChildrenWorkflowTaskChildTasksWorkflowTask) []client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowForEachTaskChildrenWorkflowTaskChildTasksWorkflowTask {
	if len(tasks) == 0 {
		return nil
	}

	taskByID := make(map[string]client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowForEachTaskChildrenWorkflowTaskChildTasksWorkflowTask)
	for _, t := range tasks {
		taskByID[t.GetId()] = t
	}

	// Find root
	var root client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowForEachTaskChildrenWorkflowTaskChildTasksWorkflowTask
	for _, t := range tasks {
		prevID := getPrevID(t.GetPrevious())
		if prevID == "" || taskByID[prevID] == nil {
			root = t
			break
		}
	}

	// Follow chain
	var ordered []client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowForEachTaskChildrenWorkflowTaskChildTasksWorkflowTask
	seen := make(map[string]bool)
	current := root
	for current != nil && !seen[current.GetId()] {
		seen[current.GetId()] = true
		ordered = append(ordered, current)
		nextID := getNextID(current.GetNext())
		if nextID != "" {
			current = taskByID[nextID]
		} else {
			current = nil
		}
	}

	return ordered
}

// Generic task info extractor for any type with GetId/GetTypename/GetName
type basicTaskInfo interface {
	GetId() string
	GetTypename() *string
	GetName() *string
}

// convertBasicTasks converts CLITaskBasic tasks (level 2+, no further nesting)
func convertBasicTasks(tasks []client.CLITaskBasic, counter *indexCounter) []*AutomationNode {
	if len(tasks) == 0 {
		return nil
	}

	var nodes []*AutomationNode
	for _, t := range tasks {
		nodes = append(nodes, makeBasicNode(t, counter))
	}
	return nodes
}

// makeBasicNode creates a basic AutomationNode from any task with basic info
func makeBasicNode(t basicTaskInfo, counter *indexCounter) *AutomationNode {
	taskType := mapTaskTypename(t.GetTypename())
	name := ""
	if n := t.GetName(); n != nil {
		name = *n
	}
	return &AutomationNode{
		ID:       t.GetId(),
		Name:     name,
		Type:     taskType,
		NodeKind: NodeTask,
		Index:    counter.next(),
	}
}

// Level 2 converters for switch->switch and switch->foreach
func convertL2SwitchInSwitchTasks(tasks []client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowSwitchTaskChildrenWorkflowTaskChildTasksWorkflowSwitchTaskChildrenWorkflowTaskChildTasksWorkflowTask, counter *indexCounter) []*AutomationNode {
	var nodes []*AutomationNode
	for _, t := range tasks {
		nodes = append(nodes, makeBasicNode(t, counter))
	}
	return nodes
}

func convertL2ForEachInSwitchTasks(tasks []client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowSwitchTaskChildrenWorkflowTaskChildTasksWorkflowForEachTaskChildrenWorkflowTaskChildTasksWorkflowTask, counter *indexCounter) []*AutomationNode {
	var nodes []*AutomationNode
	for _, t := range tasks {
		nodes = append(nodes, makeBasicNode(t, counter))
	}
	return nodes
}

func convertL2SwitchInForEachTasks(tasks []client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowForEachTaskChildrenWorkflowTaskChildTasksWorkflowSwitchTaskChildrenWorkflowTaskChildTasksWorkflowTask, counter *indexCounter) []*AutomationNode {
	var nodes []*AutomationNode
	for _, t := range tasks {
		nodes = append(nodes, makeBasicNode(t, counter))
	}
	return nodes
}

func convertL2ForEachInForEachTasks(tasks []client.GetWorkflowForVisualizationOrganizationWorkflowTasksWorkflowForEachTaskChildrenWorkflowTaskChildTasksWorkflowForEachTaskChildrenWorkflowTaskChildTasksWorkflowTask, counter *indexCounter) []*AutomationNode {
	var nodes []*AutomationNode
	for _, t := range tasks {
		nodes = append(nodes, makeBasicNode(t, counter))
	}
	return nodes
}

// buildTaskTree converts a flat list of tasks into a tree with proper nesting
// for switch and for_each tasks.
func buildTaskTree(tasks []client.WorkflowTaskRaw) []*AutomationNode {
	if len(tasks) == 0 {
		return nil
	}

	taskMap := make(map[string]client.WorkflowTaskRaw, len(tasks))
	for _, t := range tasks {
		taskMap[t.ID] = t
	}

	// Identify switch children (case IDs) and for_each children (chain IDs)
	switchCases := map[string][]switchCaseRaw{} // switchTaskID -> cases
	forEachChains := map[string]string{}        // forEachTaskID -> chainEntryID
	nestedIDs := map[string]string{}            // child/chain ID -> parent task ID

	for _, t := range tasks {
		taskType := client.GetTaskTypeLabel(t.Typename)

		if taskType == "switch" {
			if children, ok := t.RawData["children"].([]interface{}); ok {
				for _, c := range children {
					if cm, ok := c.(map[string]interface{}); ok {
						caseID, _ := cm["id"].(string)
						caseLabel, _ := cm["label"].(string)
						if caseID != "" {
							switchCases[t.ID] = append(switchCases[t.ID], switchCaseRaw{
								ID:    caseID,
								Label: caseLabel,
							})
							nestedIDs[caseID] = t.ID
						}
					}
				}
			}
		}

		if taskType == "for_each" {
			if children, ok := t.RawData["children"].([]interface{}); ok {
				for _, c := range children {
					if cm, ok := c.(map[string]interface{}); ok {
						chainID, _ := cm["id"].(string)
						if chainID != "" {
							forEachChains[t.ID] = chainID
							nestedIDs[chainID] = t.ID
						}
					}
				}
			}
		}
	}

	// Determine which tasks belong to which scope by tracing previous chains
	// A task's scope is determined by whether its chain eventually traces back
	// to a nested ID (case or for_each chain) or to the root
	taskScope := map[string]string{} // taskID -> scopeID ("root" or the case/chain ID)

	for _, t := range tasks {
		if _, already := taskScope[t.ID]; already {
			continue
		}
		resolveScope(t.ID, taskMap, nestedIDs, taskScope)
	}

	// Group tasks by scope
	scopeTasks := map[string][]client.WorkflowTaskRaw{}
	for _, t := range tasks {
		scope := taskScope[t.ID]
		scopeTasks[scope] = append(scopeTasks[scope], t)
	}

	// Build ordered chains within each scope
	counter := &indexCounter{prefix: ""}

	// Root-level chain
	rootChain := orderAndBuildChain(scopeTasks["root"], taskMap, counter,
		switchCases, forEachChains, scopeTasks)

	return rootChain
}

type switchCaseRaw struct {
	ID    string
	Label string
}

type indexCounter struct {
	prefix string
	count  int
}

func (c *indexCounter) next() string {
	c.count++
	if c.prefix == "" {
		return fmt.Sprintf("%d", c.count)
	}
	return fmt.Sprintf("%s%c", c.prefix, 'a'+rune(c.count-1))
}

func (c *indexCounter) sub(prefix string) *indexCounter {
	return &indexCounter{prefix: prefix}
}

// resolveScope traces the previous chain of a task to determine its scope.
func resolveScope(taskID string, taskMap map[string]client.WorkflowTaskRaw, nestedIDs map[string]string, taskScope map[string]string) string {
	if scope, ok := taskScope[taskID]; ok {
		return scope
	}

	t, exists := taskMap[taskID]
	if !exists {
		taskScope[taskID] = "root"
		return "root"
	}

	if t.Previous == nil {
		taskScope[taskID] = "root"
		return "root"
	}

	prevID := t.Previous.ID

	// If previous is a nested scope entry point (case or chain ID)
	if _, isNested := nestedIDs[prevID]; isNested {
		taskScope[taskID] = prevID
		return prevID
	}

	// If previous is another task, inherit its scope
	if _, isPrevTask := taskMap[prevID]; isPrevTask {
		scope := resolveScope(prevID, taskMap, nestedIDs, taskScope)
		taskScope[taskID] = scope
		return scope
	}

	// Previous is something else (trigger ID, etc.) - root scope
	taskScope[taskID] = "root"
	return "root"
}

// orderAndBuildChain orders tasks within a scope and converts them to AutomationNodes.
func orderAndBuildChain(
	tasks []client.WorkflowTaskRaw,
	allTasks map[string]client.WorkflowTaskRaw,
	counter *indexCounter,
	switchCases map[string][]switchCaseRaw,
	forEachChains map[string]string,
	scopeTasks map[string][]client.WorkflowTaskRaw,
) []*AutomationNode {
	if len(tasks) == 0 {
		return nil
	}

	ordered := orderTasksByChain(tasks)
	var nodes []*AutomationNode

	for _, t := range ordered {
		taskType := client.GetTaskTypeLabel(t.Typename)
		idx := counter.next()

		node := &AutomationNode{
			ID:      t.ID,
			Name:    t.Name,
			Type:    taskType,
			Index:   idx,
			Details: filterTaskDetails(t.RawData),
		}

		switch taskType {
		case "switch":
			node.NodeKind = NodeSwitch
			if cases, ok := switchCases[t.ID]; ok {
				for _, c := range cases {
					sc := &SwitchCase{
						ID:    c.ID,
						Label: c.Label,
						Tasks: orderAndBuildChain(
							scopeTasks[c.ID], allTasks,
							counter.sub(idx),
							switchCases, forEachChains, scopeTasks,
						),
					}
					node.Cases = append(node.Cases, sc)
				}
			}

		case "for_each":
			node.NodeKind = NodeForEach
			node.LoopList = extractForEachList(t.RawData)
			if chainID, ok := forEachChains[t.ID]; ok {
				node.LoopBody = orderAndBuildChain(
					scopeTasks[chainID], allTasks,
					counter.sub(idx),
					switchCases, forEachChains, scopeTasks,
				)
			}

		default:
			node.NodeKind = NodeTask
		}

		nodes = append(nodes, node)
	}

	return nodes
}

// orderTasksByChain orders tasks by following previous→next chain links.
func orderTasksByChain(tasks []client.WorkflowTaskRaw) []client.WorkflowTaskRaw {
	if len(tasks) <= 1 {
		return tasks
	}

	taskByID := make(map[string]client.WorkflowTaskRaw, len(tasks))
	nextMap := make(map[string]string)
	hasIncoming := make(map[string]bool)
	taskIDSet := make(map[string]bool, len(tasks))

	for _, t := range tasks {
		taskByID[t.ID] = t
		taskIDSet[t.ID] = true
		if t.Next != nil && t.Next.ID != "" {
			nextMap[t.ID] = t.Next.ID
			hasIncoming[t.Next.ID] = true
		}
	}

	var ordered []client.WorkflowTaskRaw
	for _, t := range tasks {
		if !hasIncoming[t.ID] {
			current := t
			for {
				ordered = append(ordered, current)
				nextID, ok := nextMap[current.ID]
				if !ok {
					break
				}
				if next, exists := taskByID[nextID]; exists && taskIDSet[nextID] {
					current = next
				} else {
					break
				}
			}
		}
	}

	// Include orphans
	seen := make(map[string]bool, len(ordered))
	for _, t := range ordered {
		seen[t.ID] = true
	}
	for _, t := range tasks {
		if !seen[t.ID] {
			ordered = append(ordered, t)
		}
	}

	return ordered
}

// filterTriggerDetails extracts interesting fields from trigger raw data.
func filterTriggerDetails(raw map[string]interface{}) map[string]any {
	details := map[string]any{}
	for k, v := range raw {
		switch k {
		case "id", "__typename":
			continue
		default:
			if v != nil {
				details[k] = v
			}
		}
	}
	return details
}

// filterTaskDetails extracts interesting fields from task raw data.
func filterTaskDetails(raw map[string]interface{}) map[string]any {
	details := map[string]any{}
	for k, v := range raw {
		switch k {
		case "id", "__typename", "name", "previous", "next", "children":
			continue
		default:
			if v != nil {
				details[k] = v
			}
		}
	}
	return details
}

// extractForEachList gets a human-readable label for the for_each list reference.
func extractForEachList(raw map[string]interface{}) string {
	if forEach, ok := raw["forEach"].(map[string]interface{}); ok {
		if label, ok := forEach["label"].(string); ok && label != "" {
			return label
		}
	}
	return ""
}

// computeSummary calculates aggregate statistics for the automation.
func computeSummary(cfg *AutomationConfig) AutomationSummary {
	s := AutomationSummary{
		TopLevelTasks: len(cfg.Tasks),
		OutputCount:   len(cfg.Outputs),
	}

	if cfg.Trigger != nil {
		s.TriggerType = cfg.Trigger.Type
	}

	countNodes(cfg.Tasks, &s)
	return s
}

func countNodes(nodes []*AutomationNode, s *AutomationSummary) {
	for _, n := range nodes {
		s.TotalTasks++
		switch n.NodeKind {
		case NodeSwitch:
			s.SwitchCount++
			s.TotalCases += len(n.Cases)
			for _, c := range n.Cases {
				countNodes(c.Tasks, s)
			}
		case NodeForEach:
			s.ForEachCount++
			countNodes(n.LoopBody, s)
		}
	}
}
