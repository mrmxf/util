//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package kfg

import (
	"errors"
	"log/slog"
	"reflect"

	"github.com/knadh/koanf/v2"
)

// Unmarshal is a generic function that extracts configuration data from the loaded YAML
// and converts it into a Go struct of type T. It returns a pointer to the populated struct.
//
// Generic type parameter:
// - T: The type of struct you want to unmarshal into (e.g., ClogConfig)
//
// Parameters:
// - key: The configuration key to extract (e.g., "clog" for the clog section)
// - lbl: The struct tag to use for field mapping (e.g., "koanf" to use `koanf:"field-name"` tags)
//
// Returns:
// - *T: A pointer to a new struct of type T with fields populated from the configuration
// - error: Any error that occurred during unmarshaling, or nil if successful
//
// Example usage:
//
//	config, err := Unmarshal[ClogConfig]("clog", "koanf")
//	if err != nil {
//	    // handle error
//	}
//	fmt.Println(config.Log.Level) // Access the loaded configuration
//
// How it works:
// 1. Checks if Konfigure() has been called (Kfg != nil)
// 2. Creates a new instance of type T using Go's reflection system
// 3. Handles both pointer and non-pointer types correctly
// 4. Uses koanf's UnmarshalWithConf to map YAML data to struct fields
// 5. The struct tag (lbl) tells koanf which field tags to use for mapping
func Unmarshal[T any](key string, label string) (*T, error) {
	// Check if Konfigure() has been called first
	if Raw == nil {
		return nil, errors.New("configuration not initialized: call Konfigure() before using Unmarshal()")
	}

	var result T

	// Use reflection to examine the type T and create appropriate instances
	// reflect.TypeOf gets the type information of our generic type T
	resultType := reflect.TypeOf(result)
	slog.Debug("Unmarshaling struct of type " + resultType.Name())

	if resultType.Kind() == reflect.Ptr {
		// If T is already a pointer type (like *SomeStruct), create new instance
		// reflect.New creates a new instance of the underlying type
		result = reflect.New(resultType.Elem()).Interface().(T)
	} else {
		// If T is not a pointer (like SomeStruct), we need to get its address for unmarshaling
		// because koanf.Unmarshal needs a pointer to modify the struct
		resultPtr := reflect.New(resultType).Interface()

		// Unmarshal the configuration data into our struct pointer
		// UnmarshalWithConf allows us to specify which struct tag to use
		err := Raw.UnmarshalWithConf(key, resultPtr, koanf.UnmarshalConf{Tag: label})
		if err != nil {
			return nil, err
		}

		// Extract the actual value from the pointer and convert back to type T
		// then return its address to match our return type of *T
		resultValue := reflect.ValueOf(resultPtr).Elem().Interface().(T)
		return &resultValue, nil
	}

	// For pointer types, unmarshal directly into the result
	err := Raw.UnmarshalWithConf(key, &result, koanf.UnmarshalConf{Tag: label})
	if err != nil {
		return nil, err
	}

	return &result, nil
}
