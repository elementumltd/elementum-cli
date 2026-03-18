// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package client provides the GraphQL client for the Elementum API.
//
// # Type-Safe GraphQL Client
//
// This package uses genqlient to generate type-safe GraphQL client code from
// the operations defined in the operations/*.graphql files, validated against
// schema.graphql.
//
// To regenerate the client code after modifying .graphql files:
//
//	make generate-graphql
//
// Or directly:
//
//	cd internal/client && go generate
//
// # Using the Generated Code
//
// The generated code provides type-safe functions for each GraphQL operation.
// Use the Client.Genqlient() method to get a genqlient-compatible client:
//
//	// Before (old way - still works but less type-safe):
//	var result struct { ... }
//	err := client.ExecuteInto(ctx, GetUserByIDQuery, vars, &result)
//
//	// After (new way - type-safe):
//	resp, err := GetUserByID(ctx, client.Genqlient(), userId)
//	// resp is fully typed GetUserByIDResponse
//
// # Files
//
//   - genqlient.yaml: Configuration for genqlient code generation
//   - operations/*.graphql: GraphQL query and mutation definitions
//   - generated.go: Auto-generated type-safe client code (DO NOT EDIT)
//   - graphql_client.go: Adapter implementing genqlient's graphql.Client interface
//   - queries.go, mutations.go: Legacy string-based queries (deprecated, being migrated)
package client

//go:generate go run github.com/Khan/genqlient
