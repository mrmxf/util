//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

//go:embed stash-schema.json
var stashSchemaJSON []byte

var stashSchema *jsonschema.Schema

// init compiles the embedded schema once at startup
func init() {
	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft2020

	// Add the schema to the compiler
	if err := compiler.AddResource("stash-schema.json", bytes.NewReader(stashSchemaJSON)); err != nil {
		slog.Error("failed to add stash schema resource", "error", err)
		return
	}

	// Compile the schema
	schema, err := compiler.Compile("stash-schema.json")
	if err != nil {
		slog.Error("failed to compile stash schema", "error", err)
		return
	}

	stashSchema = schema
}

// LoadValidateStash loads the stash file and validates it against the JSON schema.
// Returns the loaded stash and any validation errors.
func LoadValidateStash() (Stash, error) {
	stash := LoadStash()

	// If schema compilation failed during init, return without validation
	if stashSchema == nil {
		slog.Warn("stash schema not available, skipping validation")
		return stash, nil
	}

	// Convert stash to JSON for validation (jsonschema works with interface{})
	stashJSON, err := json.Marshal(stash)
	if err != nil {
		return stash, fmt.Errorf("failed to marshal stash for validation: %w", err)
	}

	// Unmarshal back to interface{} for validation
	var stashData interface{}
	if err := json.Unmarshal(stashJSON, &stashData); err != nil {
		return stash, fmt.Errorf("failed to unmarshal stash data: %w", err)
	}

	// Validate against schema
	if err := stashSchema.Validate(stashData); err != nil {
		// Format validation error nicely
		if ve, ok := err.(*jsonschema.ValidationError); ok {
			return stash, fmt.Errorf("stash validation failed: %s", formatValidationError(ve))
		}
		return stash, fmt.Errorf("stash validation failed: %w", err)
	}

	slog.Debug("stash validated successfully")
	return stash, nil
}

// formatValidationError formats a jsonschema.ValidationError into a readable string
func formatValidationError(ve *jsonschema.ValidationError) string {
	// Collect all errors
	var errors []string
	collectErrors(ve, &errors)

	if len(errors) == 0 {
		return "unknown validation error"
	}

	// Format as a single string
	result := ""
	for i, err := range errors {
		if i > 0 {
			result += "; "
		}
		result += err
	}
	return result
}

// collectErrors recursively collects all validation error messages
func collectErrors(ve *jsonschema.ValidationError, errors *[]string) {
	if ve == nil {
		return
	}

	// Add this error's message
	if ve.Message != "" {
		*errors = append(*errors, fmt.Sprintf("%s: %s", ve.InstanceLocation, ve.Message))
	}

	// Recursively collect from causes
	for _, cause := range ve.Causes {
		collectErrors(cause, errors)
	}
}

// ValidateStashYAML validates a YAML byte slice against the stash schema.
// This is useful for validating external YAML files before loading them.
func ValidateStashYAML(yamlData []byte) error {
	if stashSchema == nil {
		return fmt.Errorf("stash schema not available")
	}

	// Parse YAML to interface{}
	var data interface{}
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate
	if err := stashSchema.Validate(data); err != nil {
		if ve, ok := err.(*jsonschema.ValidationError); ok {
			return fmt.Errorf("validation failed: %s", formatValidationError(ve))
		}
		return err
	}

	return nil
}
