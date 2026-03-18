// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package export

import (
	"strings"
	"testing"

	"github.com/elementumltd/elementum-cli/discovery"
)

func TestFileReaderHCLGenerator_GenerateAll(t *testing.T) {
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		AIFileReaders: []discovery.FileReader{
			{
				ID:           "reader-ai",
				Name:         "AI Invoice Reader",
				Type:         discovery.FileReaderTypeAI,
				Instructions: "Extract invoice data from documents",
				Fields: []discovery.FileReaderField{
					{ID: "f1", Name: "vendor", Description: "Vendor name", Type: "TEXT", Required: true},
					{ID: "f2", Name: "amount", Description: "Total amount", Type: "DECIMAL", Required: true},
				},
			},
			{
				ID:   "reader-ocr",
				Name: "OCR Scanner",
				Type: discovery.FileReaderTypeOCR,
			},
			{
				ID:        "reader-json",
				Name:      "JSON Parser",
				Type:      discovery.FileReaderTypeJSON,
				Structure: `{"properties":[{"text":{"name":"order_id"}}]}`,
			},
			{
				ID:        "reader-xml",
				Name:      "XML Parser",
				Type:      discovery.FileReaderTypeXML,
				Structure: `{"properties":[{"text":{"name":"invoice_id"}}]}`,
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:reader-ai", ResourceType: "elementum_ai_file_reader", ResourceName: "ai_invoice_reader"},
		{ID: "app-123:reader-ocr", ResourceType: "elementum_text_file_reader", ResourceName: "ocr_scanner"},
		{ID: "app-123:reader-json", ResourceType: "elementum_json_file_reader", ResourceName: "json_parser"},
		{ID: "app-123:reader-xml", ResourceType: "elementum_xml_file_reader", ResourceName: "xml_parser"},
	}

	gen := NewFileReaderHCLGenerator(app, imports, make(map[string]string))
	hcl := gen.GenerateAll()

	// Check AI file reader
	if !strings.Contains(hcl, `resource "elementum_ai_file_reader" "ai_invoice_reader"`) {
		t.Error("Expected AI file reader resource")
	}
	if !strings.Contains(hcl, `instructions = "Extract invoice data from documents"`) {
		t.Error("Expected AI file reader instructions")
	}
	if !strings.Contains(hcl, `name = "vendor"`) {
		t.Error("Expected AI file reader field")
	}

	// Check Text/OCR file reader
	if !strings.Contains(hcl, `resource "elementum_text_file_reader" "ocr_scanner"`) {
		t.Error("Expected text file reader resource")
	}

	// Check JSON file reader
	if !strings.Contains(hcl, `resource "elementum_json_file_reader" "json_parser"`) {
		t.Error("Expected JSON file reader resource")
	}
	if !strings.Contains(hcl, `structure = `) {
		t.Error("Expected JSON file reader structure")
	}

	// Check XML file reader
	if !strings.Contains(hcl, `resource "elementum_xml_file_reader" "xml_parser"`) {
		t.Error("Expected XML file reader resource")
	}
	if !strings.Contains(hcl, `values = `) {
		t.Error("Expected XML file reader values")
	}
}

func TestFileReaderHCLGenerator_AIFileReader(t *testing.T) {
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		AIFileReaders: []discovery.FileReader{
			{
				ID:           "reader-1",
				Name:         "Invoice Extractor",
				Type:         discovery.FileReaderTypeAI,
				Instructions: "Extract vendor, amount, and date from invoices",
				Fields: []discovery.FileReaderField{
					{ID: "f1", Name: "vendor", Description: "Vendor name", Type: "TEXT", Required: true},
					{ID: "f2", Name: "amount", Description: "Invoice amount", Type: "DECIMAL", Required: true},
					{ID: "f3", Name: "date", Description: "Invoice date", Type: "DATE", Required: false},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:reader-1", ResourceType: "elementum_ai_file_reader", ResourceName: "invoice_extractor"},
	}

	gen := NewFileReaderHCLGenerator(app, imports, make(map[string]string))
	hcl := gen.GenerateAll()

	// Verify resource block
	if !strings.Contains(hcl, `resource "elementum_ai_file_reader" "invoice_extractor"`) {
		t.Error("Expected AI file reader resource block")
	}

	// Verify app_id reference
	if !strings.Contains(hcl, `app_id = elementum_app.test_app.id`) {
		t.Error("Expected app_id reference")
	}

	// Verify name
	if !strings.Contains(hcl, `name = "Invoice Extractor"`) {
		t.Error("Expected name attribute")
	}

	// Verify instructions
	if !strings.Contains(hcl, `instructions = "Extract vendor, amount, and date from invoices"`) {
		t.Error("Expected instructions attribute")
	}

	// Verify fields block
	if !strings.Contains(hcl, `fields = [`) {
		t.Error("Expected fields array")
	}

	// Verify field attributes
	if !strings.Contains(hcl, `name = "vendor"`) {
		t.Error("Expected vendor field")
	}
	if !strings.Contains(hcl, `type = "TEXT"`) {
		t.Error("Expected TEXT type")
	}
	if !strings.Contains(hcl, `required = false`) {
		t.Error("Expected optional field to have required = false")
	}
}

func TestFileReaderHCLGenerator_TextFileReader(t *testing.T) {
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		AIFileReaders: []discovery.FileReader{
			{
				ID:   "reader-1",
				Name: "Document Scanner",
				Type: discovery.FileReaderTypeOCR,
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:reader-1", ResourceType: "elementum_text_file_reader", ResourceName: "document_scanner"},
	}

	gen := NewFileReaderHCLGenerator(app, imports, make(map[string]string))
	hcl := gen.GenerateAll()

	// Verify resource block
	if !strings.Contains(hcl, `resource "elementum_text_file_reader" "document_scanner"`) {
		t.Errorf("Expected text file reader resource block, got:\n%s", hcl)
	}

	// Verify app_id reference
	if !strings.Contains(hcl, `app_id = elementum_app.test_app.id`) {
		t.Error("Expected app_id reference")
	}

	// Verify name
	if !strings.Contains(hcl, `name = "Document Scanner"`) {
		t.Error("Expected name attribute")
	}

	// Text file readers should NOT have instructions or fields
	if strings.Contains(hcl, `instructions`) {
		t.Error("Text file reader should not have instructions")
	}
	if strings.Contains(hcl, `fields`) {
		t.Error("Text file reader should not have fields")
	}
}

func TestFileReaderHCLGenerator_JSONFileReader(t *testing.T) {
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		AIFileReaders: []discovery.FileReader{
			{
				ID:        "reader-1",
				Name:      "Order JSON Parser",
				Type:      discovery.FileReaderTypeJSON,
				Structure: `{"properties":[{"text":{"name":"order_id"}},{"number":{"name":"quantity"}}]}`,
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:reader-1", ResourceType: "elementum_json_file_reader", ResourceName: "order_json_parser"},
	}

	gen := NewFileReaderHCLGenerator(app, imports, make(map[string]string))
	hcl := gen.GenerateAll()

	// Verify resource block
	if !strings.Contains(hcl, `resource "elementum_json_file_reader" "order_json_parser"`) {
		t.Errorf("Expected JSON file reader resource block, got:\n%s", hcl)
	}

	// Verify app_id reference
	if !strings.Contains(hcl, `app_id = elementum_app.test_app.id`) {
		t.Error("Expected app_id reference")
	}

	// Verify name
	if !strings.Contains(hcl, `name = "Order JSON Parser"`) {
		t.Error("Expected name attribute")
	}

	// Verify structure attribute
	if !strings.Contains(hcl, `structure = `) {
		t.Error("Expected structure attribute")
	}
	if !strings.Contains(hcl, `order_id`) {
		t.Error("Expected structure to contain order_id")
	}
}

func TestFileReaderHCLGenerator_XMLFileReader(t *testing.T) {
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		AIFileReaders: []discovery.FileReader{
			{
				ID:        "reader-1",
				Name:      "Invoice XML Parser",
				Type:      discovery.FileReaderTypeXML,
				Structure: `{"properties":[{"text":{"name":"invoice_number"}},{"decimal":{"name":"total"}}]}`,
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:reader-1", ResourceType: "elementum_xml_file_reader", ResourceName: "invoice_xml_parser"},
	}

	gen := NewFileReaderHCLGenerator(app, imports, make(map[string]string))
	hcl := gen.GenerateAll()

	// Verify resource block
	if !strings.Contains(hcl, `resource "elementum_xml_file_reader" "invoice_xml_parser"`) {
		t.Errorf("Expected XML file reader resource block, got:\n%s", hcl)
	}

	// Verify app_id reference
	if !strings.Contains(hcl, `app_id = elementum_app.test_app.id`) {
		t.Error("Expected app_id reference")
	}

	// Verify name
	if !strings.Contains(hcl, `name = "Invoice XML Parser"`) {
		t.Error("Expected name attribute")
	}

	// Verify values attribute (not structure for XML)
	if !strings.Contains(hcl, `values = `) {
		t.Error("Expected values attribute for XML file reader")
	}
	if !strings.Contains(hcl, `invoice_number`) {
		t.Error("Expected values to contain invoice_number")
	}
}

func TestFileReaderHCLGenerator_EmptyApp(t *testing.T) {
	app := &discovery.App{
		ID:            "app-123",
		Name:          "Test App",
		AIFileReaders: []discovery.FileReader{},
	}

	gen := NewFileReaderHCLGenerator(app, nil, make(map[string]string))
	hcl := gen.GenerateAll()

	if hcl != "" {
		t.Errorf("Expected empty output for app with no file readers, got: %s", hcl)
	}
}

func TestFileReaderHCLGenerator_NilApp(t *testing.T) {
	gen := NewFileReaderHCLGenerator(nil, nil, make(map[string]string))
	hcl := gen.GenerateAll()

	if hcl != "" {
		t.Errorf("Expected empty output for nil app, got: %s", hcl)
	}
}

func TestFileReaderHCLGenerator_MissingImport(t *testing.T) {
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		AIFileReaders: []discovery.FileReader{
			{
				ID:   "reader-1",
				Name: "Test Reader",
				Type: discovery.FileReaderTypeOCR,
			},
		},
	}

	// No import block for the file reader
	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
	}

	gen := NewFileReaderHCLGenerator(app, imports, make(map[string]string))
	hcl := gen.GenerateAll()

	// Should not generate HCL for readers without import blocks
	if strings.Contains(hcl, "elementum_text_file_reader") {
		t.Error("Should not generate HCL for reader without import block")
	}
}
