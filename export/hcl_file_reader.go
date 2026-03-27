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
	"strings"

	"github.com/elementumltd/elementum-cli/discovery"
)

// FileReaderHCLGenerator generates HCL for all file reader resource types
// Supports: elementum_ai_file_reader, elementum_text_file_reader, elementum_json_file_reader, elementum_xml_file_reader
type FileReaderHCLGenerator struct {
	app     *discovery.App
	imports []ImportBlock
	uuidMap map[string]string
}

// NewFileReaderHCLGenerator creates a new file reader HCL generator
func NewFileReaderHCLGenerator(app *discovery.App, imports []ImportBlock, uuidMap map[string]string) *FileReaderHCLGenerator {
	return &FileReaderHCLGenerator{
		app:     app,
		imports: imports,
		uuidMap: uuidMap,
	}
}

// GenerateAll generates HCL for all file readers in the app
func (g *FileReaderHCLGenerator) GenerateAll() string {
	return SerializeBlocks(g.GenerateAllIR())
}

// GenerateAllIR generates IR blocks for all file readers in the app
func (g *FileReaderHCLGenerator) GenerateAllIR() []*HCLBlock {
	if g.app == nil || len(g.app.AIFileReaders) == 0 {
		return nil
	}

	appResourceName := AppResourceName(g.app)
	var blocks []*HCLBlock

	for _, reader := range g.app.AIFileReaders {
		b := g.GenerateFileReaderIR(&reader, appResourceName)
		if b != nil {
			blocks = append(blocks, b)
		}
	}

	return blocks
}

// GenerateFileReaderIR generates an IR block for a single file reader resource
func (g *FileReaderHCLGenerator) GenerateFileReaderIR(reader *discovery.FileReader, appResourceName string) *HCLBlock {
	resourceType := reader.TerraformResourceType()

	// Find the resource name from imports
	var resourceName string
	for _, imp := range g.imports {
		if imp.ResourceType == resourceType && strings.Contains(imp.ID, reader.ID) {
			resourceName = imp.ResourceName
			break
		}
	}
	if resourceName == "" {
		return nil
	}

	appRef := Ref("elementum_app." + appResourceName + ".id")

	switch reader.Type {
	case discovery.FileReaderTypeAI:
		return g.generateAIFileReaderIR(reader, resourceName, appRef)
	case discovery.FileReaderTypeOCR:
		return g.generateTextFileReaderIR(reader, resourceName, appRef)
	case discovery.FileReaderTypeJSON:
		return g.generateJSONFileReaderIR(reader, resourceName, appRef)
	case discovery.FileReaderTypeXML:
		return g.generateXMLFileReaderIR(reader, resourceName, appRef)
	default:
		return g.generateAIFileReaderIR(reader, resourceName, appRef)
	}
}

func (g *FileReaderHCLGenerator) generateAIFileReaderIR(reader *discovery.FileReader, resourceName string, appRef HCLReference) *HCLBlock {
	b := NewResourceBlock("elementum_ai_file_reader", resourceName)
	b.SetAttr("app_id", appRef)
	b.SetAttr("name", Str(reader.Name))

	if reader.Instructions != "" {
		b.SetAttr("instructions", Str(reader.Instructions))
	}

	if len(reader.Fields) > 0 {
		var fieldObjs []HCLValue
		for _, field := range reader.Fields {
			attrs := []*HCLAttribute{
				Attr("name", Str(field.Name)),
				Attr("description", Str(field.Description)),
				Attr("type", Str(field.Type)),
			}
			if !field.Required {
				attrs = append(attrs, Attr("required", Bool(false)))
			}
			fieldObjs = append(fieldObjs, Obj(attrs...))
		}
		b.SetAttr("fields", HCLList{Values: fieldObjs})
	}

	return b
}

func (g *FileReaderHCLGenerator) generateTextFileReaderIR(reader *discovery.FileReader, resourceName string, appRef HCLReference) *HCLBlock {
	b := NewResourceBlock("elementum_text_file_reader", resourceName)
	b.SetAttr("app_id", appRef)
	b.SetAttr("name", Str(reader.Name))
	return b
}

func (g *FileReaderHCLGenerator) generateJSONFileReaderIR(reader *discovery.FileReader, resourceName string, appRef HCLReference) *HCLBlock {
	b := NewResourceBlock("elementum_json_file_reader", resourceName)
	b.SetAttr("app_id", appRef)
	b.SetAttr("name", Str(reader.Name))
	if reader.Structure != "" {
		b.SetAttr("structure", Str(reader.Structure))
	}
	return b
}

func (g *FileReaderHCLGenerator) generateXMLFileReaderIR(reader *discovery.FileReader, resourceName string, appRef HCLReference) *HCLBlock {
	b := NewResourceBlock("elementum_xml_file_reader", resourceName)
	b.SetAttr("app_id", appRef)
	b.SetAttr("name", Str(reader.Name))
	if reader.Structure != "" {
		b.SetAttr("values", Str(reader.Structure))
	}
	return b
}
