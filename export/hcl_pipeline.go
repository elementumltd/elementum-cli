// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package export: hcl_pipeline.go implements the new IR-based export pipeline.
//
// This is the Strangler Fig entry point that unifies both generation tracks
// (tofu plan output + CLI generators) into a single []*HCLBlock pipeline:
//
//	tofu plan output  ──ParseTofuOutput──►  []*HCLBlock ─┐
//	                                                      ├─► MergeBlocks ─► ApplyAllTransforms ─► Serialize
//	CLI generators    ──GenerateAll─────►  []*HCLBlock ─┘
//
// Usage:
//
//	pipeline := NewExportPipeline(app, imports)
//	pipeline.AddTofuBlocks(tofuText)
//	pipeline.AddCLIBlocks()
//	result := pipeline.Execute()
package export

import (
	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/elementumltd/elementum-cli/logger"
)

// ExportPipeline orchestrates the IR-based export, connecting:
//   - Phase 4: tofu output parsing
//   - Phase 2-3: CLI generator IR output
//   - Phase 5: IR transforms (strip nulls, resolve UUIDs, fix self-refs)
//   - Phase 6: file organization
//   - Phase 7: generator registry
//   - Phase 1: serialization
type ExportPipeline struct {
	app     *discovery.App
	imports []ImportBlock
	uuidMap map[string]string

	tofuBlocks []*HCLBlock // blocks from tofu plan output
	cliBlocks  []*HCLBlock // blocks from CLI generators

	registry *GeneratorRegistry
}

// NewExportPipeline creates a new IR-based export pipeline.
func NewExportPipeline(app *discovery.App, imports []ImportBlock) *ExportPipeline {
	uuidMap := buildUUIDMap(imports, app)

	return &ExportPipeline{
		app:     app,
		imports: imports,
		uuidMap: uuidMap,
	}
}

// AddTofuOutput parses raw tofu plan output into IR blocks and adds them to the pipeline.
func (p *ExportPipeline) AddTofuOutput(tofuText string) error {
	blocks, err := ParseTofuOutput(tofuText)
	if err != nil {
		return err
	}
	logger.Debug("parsed %d blocks from tofu output", len(blocks))
	p.tofuBlocks = append(p.tofuBlocks, blocks...)
	return nil
}

// AddCLIBlocks generates IR blocks from all CLI generators and adds them to the pipeline.
// Uses the registry pattern from Phase 7.
func (p *ExportPipeline) AddCLIBlocks() {
	p.registry = NewAppGeneratorRegistry(p.app, p.imports, p.uuidMap)
	blocks := p.registry.GenerateAll()
	logger.Debug("CLI generators produced %d blocks", len(blocks))
	p.cliBlocks = append(p.cliBlocks, blocks...)
}

// AddBlocks adds pre-built blocks to the CLI blocks pipeline.
func (p *ExportPipeline) AddBlocks(blocks []*HCLBlock) {
	p.cliBlocks = append(p.cliBlocks, blocks...)
}

// Execute runs the full pipeline and returns the final HCL text.
// This is the single-file output mode.
func (p *ExportPipeline) Execute() string {
	// Merge tofu and CLI blocks (CLI overrides tofu for same resource)
	merged := MergeBlocks(p.tofuBlocks, p.cliBlocks)

	// Deduplicate
	merged = DeduplicateBlocks(merged)

	// Apply transforms
	ApplyAllTransforms(merged, p.uuidMap, p.app, p.imports)

	// Serialize
	return SerializeBlocks(merged)
}

// ExecuteMultiFile runs the pipeline and returns organized multi-file export.
func (p *ExportPipeline) ExecuteMultiFile() *BlockMultiFileExport {
	// Merge tofu and CLI blocks
	merged := MergeBlocks(p.tofuBlocks, p.cliBlocks)

	// Deduplicate
	merged = DeduplicateBlocks(merged)

	// Apply transforms
	ApplyAllTransforms(merged, p.uuidMap, p.app, p.imports)

	// Organize into files
	return OrganizeBlocksExport(merged, p.app, p.imports)
}

// GetMergedBlocks returns the merged and transformed blocks without serializing.
// Useful for testing or further manipulation.
func (p *ExportPipeline) GetMergedBlocks() []*HCLBlock {
	merged := MergeBlocks(p.tofuBlocks, p.cliBlocks)
	merged = DeduplicateBlocks(merged)
	ApplyAllTransforms(merged, p.uuidMap, p.app, p.imports)
	return merged
}

// GetUUIDMap returns the UUID map built from imports and app for external use.
func (p *ExportPipeline) GetUUIDMap() map[string]string {
	return p.uuidMap
}
