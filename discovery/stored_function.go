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

	"github.com/elementumltd/elementum-cli/internal/client"
)

// convertStoredFunctionData converts client.StoredFunctionData to discovery.StoredFunction
func convertStoredFunctionData(sf *client.StoredFunctionData) StoredFunction {
	result := StoredFunction{
		ID:            sf.ID,
		Name:          sf.Name,
		DisplayName:   sf.DisplayName,
		Description:   sf.Description,
		DatabaseName:  sf.DatabaseName,
		SchemaName:    sf.SchemaName,
		Type:          sf.Type,
		Configured:    sf.Configured,
		CloudLinkID:   sf.CloudLinkID,
		CloudLinkName: sf.CloudLinkName,
	}

	// Convert parameters
	for _, param := range sf.Parameters {
		result.Parameters = append(result.Parameters, StoredFunctionParameter{
			Name: param.Name,
			Type: param.Type,
		})
	}

	// Convert return type
	if sf.Output != nil {
		if sf.Output.OutputType == "simple" {
			result.ReturnType = sf.Output.ReturnType
		} else if sf.Output.OutputType == "table" {
			result.ReturnType = "TABLE"
		}
	}

	return result
}

// GetStoredFunction retrieves a stored function by its ID and CloudLink ID
func GetStoredFunction(ctx context.Context, c *client.Client, cloudLinkID, functionID string) (*StoredFunction, error) {
	result, err := client.GetStoredFunction(ctx, c.Genqlient(), cloudLinkID, functionID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stored function: %w", err)
	}

	sfData := client.ExtractStoredFunctionFromGetStoredFunction(result)
	if sfData == nil {
		return nil, fmt.Errorf("stored function not found: %s", functionID)
	}

	sf := convertStoredFunctionData(sfData)
	return &sf, nil
}

// ListStoredFunctions retrieves all stored functions for a CloudLink
func ListStoredFunctions(ctx context.Context, c *client.Client, cloudLinkID string) ([]StoredFunction, error) {
	result, err := client.ListStoredFunctions(ctx, c.Genqlient(), cloudLinkID)
	if err != nil {
		return nil, fmt.Errorf("failed to list stored functions: %w", err)
	}

	sfData := client.ExtractStoredFunctionsFromList(result)
	var functions []StoredFunction
	for _, sf := range sfData {
		functions = append(functions, convertStoredFunctionData(&sf))
	}

	return functions, nil
}

// ListAllStoredFunctions retrieves all stored functions across all CloudLinks
func ListAllStoredFunctions(ctx context.Context, c *client.Client) ([]StoredFunction, error) {
	result, err := client.ListAllStoredFunctions(ctx, c.Genqlient())
	if err != nil {
		return nil, fmt.Errorf("failed to list all stored functions: %w", err)
	}

	sfData := client.ExtractAllStoredFunctions(result)
	var functions []StoredFunction
	for _, sf := range sfData {
		functions = append(functions, convertStoredFunctionData(&sf))
	}

	return functions, nil
}

// GetStoredFunctionByID finds a stored function by ID across all CloudLinks
func GetStoredFunctionByID(ctx context.Context, c *client.Client, functionID string) (*StoredFunction, error) {
	functions, err := ListAllStoredFunctions(ctx, c)
	if err != nil {
		return nil, err
	}

	for i := range functions {
		if functions[i].ID == functionID {
			return &functions[i], nil
		}
	}

	return nil, fmt.Errorf("stored function not found: %s", functionID)
}

// GetUUIDMappings returns UUID to Terraform reference mappings for a StoredFunction
func (sf *StoredFunction) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	m[sf.ID] = "data.elementum_stored_function." + resourceName + ".id"
	return m
}
