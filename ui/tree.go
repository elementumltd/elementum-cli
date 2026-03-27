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

package ui

import (
	"fmt"
	"strings"
)

// TreeNode represents a node in the tree
type TreeNode struct {
	Label    string
	Meta     string // Optional metadata (shown in muted color)
	Children []*TreeNode
}

// NewTreeNode creates a new tree node
func NewTreeNode(label string) *TreeNode {
	return &TreeNode{
		Label:    label,
		Children: []*TreeNode{},
	}
}

// AddChild adds a child node
func (n *TreeNode) AddChild(child *TreeNode) *TreeNode {
	n.Children = append(n.Children, child)
	return child
}

// AddChildWithLabel adds a child node with the given label
func (n *TreeNode) AddChildWithLabel(label string, meta ...string) *TreeNode {
	child := NewTreeNode(label)
	if len(meta) > 0 {
		child.Meta = meta[0]
	}
	n.Children = append(n.Children, child)
	return child
}

// Tree renders a tree structure
type Tree struct {
	Root *TreeNode
}

// NewTree creates a new tree with a root node
func NewTree(rootLabel string) *Tree {
	return &Tree{
		Root: NewTreeNode(rootLabel),
	}
}

// Render renders the tree as a string
func (t *Tree) Render() string {
	if t.Root == nil {
		return ""
	}

	var b strings.Builder

	// Render root
	label := TreeItemStyle.Render(t.Root.Label)
	if t.Root.Meta != "" {
		label += " " + TreeMetaStyle.Render(t.Root.Meta)
	}
	b.WriteString(label)
	b.WriteString("\n")

	// Render children
	for i, child := range t.Root.Children {
		isLast := i == len(t.Root.Children)-1
		t.renderNode(child, "", isLast, &b)
	}

	return b.String()
}

// renderNode recursively renders a node and its children
func (t *Tree) renderNode(node *TreeNode, prefix string, isLast bool, b *strings.Builder) {
	// Determine branch characters
	var branch, continuation string
	if isLast {
		branch = "└── "
		continuation = "    "
	} else {
		branch = "├── "
		continuation = "│   "
	}

	// Render current node
	branchStr := TreeBranchStyle.Render(prefix + branch)
	label := TreeItemStyle.Render(node.Label)
	if node.Meta != "" {
		label += " " + TreeMetaStyle.Render(node.Meta)
	}
	b.WriteString(branchStr + label + "\n")

	// Render children
	for i, child := range node.Children {
		childIsLast := i == len(node.Children)-1
		t.renderNode(child, prefix+continuation, childIsLast, b)
	}
}

// RenderCompact renders a more compact tree (for large trees)
func (t *Tree) RenderCompact() string {
	if t.Root == nil {
		return ""
	}

	var b strings.Builder

	// Render root
	b.WriteString(TreeHighlightStyle.Render(t.Root.Label))
	if t.Root.Meta != "" {
		b.WriteString(" " + TreeMetaStyle.Render(t.Root.Meta))
	}
	b.WriteString("\n")

	// Render children with minimal branching
	for _, child := range t.Root.Children {
		t.renderCompactNode(child, "  ", &b)
	}

	return b.String()
}

// renderCompactNode renders a node in compact mode
func (t *Tree) renderCompactNode(node *TreeNode, indent string, b *strings.Builder) {
	bullet := TreeBranchStyle.Render(indent + "• ")
	label := TreeItemStyle.Render(node.Label)
	if node.Meta != "" {
		label += " " + TreeMetaStyle.Render(node.Meta)
	}
	b.WriteString(bullet + label + "\n")

	// Render children with more indentation
	for _, child := range node.Children {
		t.renderCompactNode(child, indent+"  ", b)
	}
}

// Example function to demonstrate tree rendering
func ExampleTree() {
	tree := NewTree("IT Support Requests (app_abc123)")

	// Add fields
	fields := tree.Root.AddChildWithLabel("Fields", "(12)")
	fields.AddChildWithLabel("title", "[text]")
	fields.AddChildWithLabel("description", "[longtext]")
	fields.AddChildWithLabel("priority", "[dropdown: Low, Medium, High, Critical]")
	fields.AddChildWithLabel("assignee", "[user]")
	fields.AddChildWithLabel("... 8 more")

	// Add layouts
	layouts := tree.Root.AddChildWithLabel("Layouts", "(2)")
	layouts.AddChildWithLabel("intake_form")
	layouts.AddChildWithLabel("detail_view")

	// Add automations
	automations := tree.Root.AddChildWithLabel("Automations", "(3)")

	autoAssign := automations.AddChildWithLabel("auto_assign")
	autoAssign.AddChildWithLabel("trigger: record_created")
	autoAssign.AddChildWithLabel("tasks: 4")

	notify := automations.AddChildWithLabel("notify_assignee")
	notify.AddChildWithLabel("trigger: record_updated")
	notify.AddChildWithLabel("tasks: 2")

	// Add agents
	agents := tree.Root.AddChildWithLabel("Agents", "(1)")
	supportBot := agents.AddChildWithLabel("support_bot")
	supportBot.AddChildWithLabel("tools: 3")

	// Add approval processes
	tree.Root.AddChildWithLabel("Approval Processes", "(1)")

	fmt.Println(tree.Render())
	fmt.Println()
	fmt.Println("Compact version:")
	fmt.Println(tree.RenderCompact())
}
