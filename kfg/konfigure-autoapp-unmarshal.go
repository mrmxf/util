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

// AutoAppUnmarshal automatically unmarshals the application configuration using the provided
// KonfigureOpt settings. This function provides defaults for unspecified AutoApp options.
//
// How it works:
//   - 1. Applies defaults for any unspecified AutoApp options:
//   - AutoAppStruct defaults to &AppConfig
//   - AutoAppKey defaults to AppKey ("clog")
//   - AutoAppLabel defaults to AppLabel ("koanf")
//   - 2. Uses the generic Unmarshal function to populate the destination structure
//   - 3. Logs the success or failure of the unmarshaling operation
//
// Parameters:
//   - opt: KonfigureOpt containing AutoApp configuration settings
//
// Returns an error if the configuration cannot be unmarshaled, nil if successful.
func AutoAppUnmarshal(opt *KonfigureOpt) error {
	var (
		appStruct any
		appKey    string
		appTag    string
	)

	// Check if Konfigure() has been called first
	if Raw == nil {
		return errors.New("configuration not initialized: call Konfigure() before using AutoAppUnmarshal()")
	}

	// Apply defaults for AutoApp options
	if appStruct = opt.AutoAppStruct; appStruct == nil {
		slog.Debug("AutoAppUnmarshal: AutoAppStruct is nil - nothing to do.")
		return nil
	}

	// if there is no AutoAppKey then there is no configuration to do
	if appKey = opt.AutoAppKey; appKey == "" {
		slog.Debug("AutoAppUnmarshal: AutoAppKey is \"\" - nothing to do.")
		return nil
	}

	// This might cause issues - ho hum. works for me
	if val := reflect.ValueOf(opt.AutoAppStruct); val.Kind() != reflect.Ptr {
		slog.Debug("AutoAppUnmarshal: converting opt.AutoAppStruct to a pointer")
		appStruct = &opt.AutoAppStruct
	}

	if appTag = opt.AutoAppAnnotationLabel; appTag == "" {
		appTag = KonfigDefaultAppTag
		slog.Debug("AutoAppUnmarshal: using default AppLabel", "label", appTag)
	}

	slog.Debug("AutoAppUnmarshal: starting automatic unmarshal", "key", appKey, "label", appTag)

	// Use UnmarshalWithConf directly with the provided struct pointer
	// This works with any struct type at runtime without requiring compile-time generics
	err := (*Raw).UnmarshalWithConf(appKey, appStruct, koanf.UnmarshalConf{Tag: appTag})
	if err != nil {
		slog.Debug("AutoAppUnmarshal: failed to unmarshal configuration", "key", appKey, "error", err)
		return err
	}

	slog.Debug("AutoAppUnmarshal: successfully populated destination struct")

	slog.Debug("AutoAppUnmarshal: completed automatic unmarshal successfully")
	return nil
}
