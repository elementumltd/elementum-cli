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
	"fmt"
	"net/url"
	"strings"

	"github.com/elementumltd/elementum-cli/logger"
)

// ResourceInfo contains the parsed resource type and identifier from a URL
type ResourceInfo struct {
	Type       string // "App", "Element", "Task", "Table", "CloudLink"
	Identifier string // namespace, handle, or name depending on resource type
}

// ParseResourceFromURL parses an Elementum URL and returns the resource type and identifier.
// Supports various URL patterns for Apps, Elements, Tasks, Tables, and CloudLinks.
//
// Supported URL patterns:
//   - App:       /app/<namespace>/..., /apps/<namespace>/..., /admin/apps/<namespace>/...
//   - Element:   /element/<handle>/..., /elements/<handle>/..., /admin/elements/<handle>/...
//   - Task:      /task/<handle>/..., /tasks/<handle>/..., /admin/tasks/<handle>/...
//   - Table:     /admin/tables/<handle>/..., /tables/<handle>/...
//   - CloudLink: /admin/cloudlinks/<name>/..., /cloudlinks/<name>/...
func ParseResourceFromURL(input string) (*ResourceInfo, error) {
	logger.Debug("parsing resource from URL", "input", input)

	// If it doesn't look like a URL, return an error
	if !strings.Contains(input, "/") {
		return nil, fmt.Errorf("input does not appear to be a URL: %s", input)
	}

	// Normalize the input - handle both full URLs and path-only inputs
	var path string
	if strings.Contains(input, "://") {
		// Full URL - parse it
		u, err := url.Parse(input)
		if err != nil {
			return nil, fmt.Errorf("failed to parse URL: %w", err)
		}
		path = u.Path
	} else {
		// Path-only input
		path = input
	}

	// Clean the path
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")

	logger.Debug("URL parsed into parts", "parts", parts, "path", path)

	// Look for resource type markers in the path
	for i := 0; i < len(parts); i++ {
		part := strings.ToLower(parts[i])

		// Check if this part is a resource type marker and there's an identifier after it
		if i+1 >= len(parts) {
			continue
		}

		identifier := parts[i+1]
		if identifier == "" {
			continue
		}

		switch part {
		// App patterns
		case "app", "apps":
			logger.Debug("found app identifier", "identifier", identifier)
			return &ResourceInfo{Type: "App", Identifier: identifier}, nil

		// Element patterns
		case "element", "elements":
			logger.Debug("found element identifier", "identifier", identifier)
			return &ResourceInfo{Type: "Element", Identifier: identifier}, nil

		// Task patterns
		case "task", "tasks":
			logger.Debug("found task identifier", "identifier", identifier)
			return &ResourceInfo{Type: "Task", Identifier: identifier}, nil

		// Table patterns
		case "table", "tables":
			logger.Debug("found table identifier", "identifier", identifier)
			return &ResourceInfo{Type: "Table", Identifier: identifier}, nil

		// CloudLink patterns
		case "cloudlink", "cloudlinks":
			logger.Debug("found cloudlink identifier", "identifier", identifier)
			return &ResourceInfo{Type: "CloudLink", Identifier: identifier}, nil

		// Admin section may prefix resource types
		case "admin":
			// Check the next part for the actual resource type
			continue
		}
	}

	return nil, fmt.Errorf("could not determine resource type from URL: %s", input)
}
