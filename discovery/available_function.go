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
	"strings"

	"github.com/elementumltd/elementum-cli/internal/client"
)

// AvailableFunction represents a Snowflake function available for configuration.
// These are functions discovered in the Snowflake database that have not yet been
// configured as elementum_function resources.
type AvailableFunction struct {
	Name          string
	DatabaseName  string
	SchemaName    string
	Type          string // "PROCEDURE" or "USER_DEFINED_FUNCTION"
	Description   string
	MinArgs       int
	MaxArgs       int
	BuiltIn       bool
	Secure        bool
	TableFunction bool
	Parameters    []AvailableFunctionParameter
	ReturnType    string // Simple return type, or "TABLE" for table functions
}

// AvailableFunctionParameter represents an input parameter.
type AvailableFunctionParameter struct {
	Name string
	Type string
}

// convertAvailableFunctionData converts client.AvailableFunctionData to discovery.AvailableFunction
func convertAvailableFunctionData(af *client.AvailableFunctionData) AvailableFunction {
	result := AvailableFunction{
		Name:          af.Name,
		DatabaseName:  af.DatabaseName,
		SchemaName:    af.SchemaName,
		Type:          af.Type,
		Description:   af.Description,
		MinArgs:       af.MinArgs,
		MaxArgs:       af.MaxArgs,
		BuiltIn:       af.BuiltIn,
		Secure:        af.Secure,
		TableFunction: af.TableFunction,
	}

	// Convert parameters
	for _, param := range af.Parameters {
		result.Parameters = append(result.Parameters, AvailableFunctionParameter{
			Name: param.Name,
			Type: param.Type,
		})
	}

	// Convert return type
	if af.Output != nil {
		switch af.Output.OutputType {
		case "simple":
			result.ReturnType = af.Output.ReturnType
		case "table":
			result.ReturnType = "TABLE"
		}
	}

	return result
}

// ListAvailableFunctions retrieves all available Snowflake functions for a CloudLink.
// Unlike ListStoredFunctions, this returns functions in Snowflake that can be configured,
// not functions already configured in Elementum.
func ListAvailableFunctions(ctx context.Context, c *client.Client, cloudLinkID string, functionType string) ([]AvailableFunction, error) {
	// Convert function type string to GraphQL enum pointer
	var typePtr *client.SnowflakeFunctionType
	if functionType != "" {
		switch strings.ToLower(functionType) {
		case "procedure", "procedures":
			t := client.SnowflakeFunctionTypeProcedure
			typePtr = &t
		case "udf", "user_defined_function", "function", "functions":
			t := client.SnowflakeFunctionTypeUserDefinedFunction
			typePtr = &t
		default:
			return nil, fmt.Errorf("invalid function type: %s (use 'procedure' or 'udf')", functionType)
		}
	}

	result, err := client.ListAvailableFunctions(ctx, c.Genqlient(), cloudLinkID, typePtr)
	if err != nil {
		return nil, fmt.Errorf("failed to list available functions: %w", err)
	}

	afData := client.ExtractAvailableFunctions(result)
	var functions []AvailableFunction
	for _, af := range afData {
		functions = append(functions, convertAvailableFunctionData(&af))
	}

	return functions, nil
}

// CloudLinkFunctionsResult holds both configured (stored) and available functions for a CloudLink.
type CloudLinkFunctionsResult struct {
	StoredFunctions    []client.CloudLinkStoredFunction
	AvailableFunctions []AvailableFunction
}

// ListCloudLinkFunctions retrieves both stored (configured) and available functions for a CloudLink.
// Requires database and schema filters to scope the Snowflake query and avoid timeouts.
func ListCloudLinkFunctions(ctx context.Context, c *client.Client, cloudLinkID, database, schema, functionType string) (*CloudLinkFunctionsResult, error) {
	result := &CloudLinkFunctionsResult{}

	// Always fetch stored functions first (fast query, doesn't hit live Snowflake)
	storedResp, err := client.ListStoredFunctions(ctx, c.Genqlient(), cloudLinkID)
	if err != nil {
		return nil, fmt.Errorf("failed to list stored functions: %w", err)
	}
	storedFunctions := client.ExtractStoredFunctionsFromList(storedResp)
	for _, sf := range storedFunctions {
		result.StoredFunctions = append(result.StoredFunctions, client.CloudLinkStoredFunction{
			ID:          sf.ID,
			Name:        sf.Name,
			DisplayName: sf.DisplayName,
		})
	}

	// Build filter for available functions (required to avoid slow unfiltered queries)
	var children []map[string]any
	if database != "" {
		children = append(children, map[string]any{
			"type":  "EQUALS",
			"field": "database",
			"value": map[string]any{"type": "TEXT", "value": database},
		})
	}
	if schema != "" {
		children = append(children, map[string]any{
			"type":  "EQUALS",
			"field": "schema",
			"value": map[string]any{"type": "TEXT", "value": schema},
		})
	}
	var filter any
	if len(children) == 1 {
		filter = children[0]
	} else {
		filter = map[string]any{
			"type":     "AND",
			"children": children,
		}
	}
	filterJSON, err := json.Marshal(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to build filter: %w", err)
	}
	raw := json.RawMessage(filterJSON)
	filterPtr := &raw

	// Convert function type string to GraphQL enum pointer
	var typePtr *client.SnowflakeFunctionType
	if functionType != "" {
		switch strings.ToLower(functionType) {
		case "procedure", "procedures":
			t := client.SnowflakeFunctionTypeProcedure
			typePtr = &t
		case "udf", "user_defined_function", "function", "functions":
			t := client.SnowflakeFunctionTypeUserDefinedFunction
			typePtr = &t
		default:
			return nil, fmt.Errorf("invalid function type: %s (use 'procedure' or 'udf')", functionType)
		}
	}

	resp, err := client.ListCloudLinkFunctions(ctx, c.Genqlient(), cloudLinkID, filterPtr, typePtr)
	if err != nil {
		return nil, fmt.Errorf("failed to list available functions: %w", err)
	}

	extracted := client.ExtractCloudLinkFunctions(resp)
	for _, af := range extracted.AvailableFunctions {
		result.AvailableFunctions = append(result.AvailableFunctions, convertAvailableFunctionData(&af))
	}
	return result, nil
}

// ResolveCloudLinkID resolves a cloudlink name or ID to an ID.
// If the input looks like a UUID, it returns it directly.
// Otherwise, it searches for a cloudlink with a matching name (case-insensitive).
func ResolveCloudLinkID(ctx context.Context, c *client.Client, nameOrID string) (string, error) {
	// If it looks like a UUID (contains dashes and is the right length), use it directly
	if len(nameOrID) == 36 && strings.Count(nameOrID, "-") == 4 {
		return nameOrID, nil
	}

	// Otherwise, search by name
	cloudlinks, err := ListCloudLinks(ctx, c)
	if err != nil {
		return "", fmt.Errorf("failed to list cloudlinks: %w", err)
	}

	// Case-insensitive name match
	lowerName := strings.ToLower(nameOrID)
	for _, cl := range cloudlinks {
		if strings.ToLower(cl.Name) == lowerName {
			return cl.ID, nil
		}
	}

	return "", fmt.Errorf("cloudlink not found: %s", nameOrID)
}

// TypeDisplayName returns a human-friendly display name for the function type.
func (af *AvailableFunction) TypeDisplayName() string {
	switch af.Type {
	case "PROCEDURE":
		return "Procedure"
	case "USER_DEFINED_FUNCTION":
		return "UDF"
	default:
		return af.Type
	}
}

// FullyQualifiedName returns the database.schema.name format.
func (af *AvailableFunction) FullyQualifiedName() string {
	parts := []string{}
	if af.DatabaseName != "" {
		parts = append(parts, af.DatabaseName)
	}
	if af.SchemaName != "" {
		parts = append(parts, af.SchemaName)
	}
	parts = append(parts, af.Name)
	return strings.Join(parts, ".")
}

// ArgsDisplay returns a human-friendly argument count display.
func (af *AvailableFunction) ArgsDisplay() string {
	if af.MinArgs == af.MaxArgs {
		return fmt.Sprintf("%d", af.MinArgs)
	}
	return fmt.Sprintf("%d-%d", af.MinArgs, af.MaxArgs)
}

// ParametersDisplay returns a formatted parameter list.
func (af *AvailableFunction) ParametersDisplay() string {
	if len(af.Parameters) == 0 {
		return "()"
	}
	var params []string
	for _, p := range af.Parameters {
		if p.Name != "" {
			params = append(params, fmt.Sprintf("%s: %s", p.Name, p.Type))
		} else {
			params = append(params, p.Type)
		}
	}
	return "(" + strings.Join(params, ", ") + ")"
}
