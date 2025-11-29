// Copyright 2025 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package procfs provides functions to interact with procfs filesystem.
package procfs

import (
	"errors"
	"strconv"
	"strings"
)

type scalarParser func(string) (any, error)

var scalarParsers = map[string]scalarParser{
	"string": func(v string) (any, error) { return v, nil },
	"bool": func(v string) (any, error) {
		return strconv.ParseBool(v)
	},
	"int": func(v string) (any, error) {
		return strconv.Atoi(v)
	},
	"float32": func(v string) (any, error) {
		f, err := strconv.ParseFloat(v, 32)
		return float32(f), err
	},
	"float64": func(v string) (any, error) {
		return strconv.ParseFloat(v, 64)
	},
}

func scalarTypeName[T any]() string {
	var zero T
	switch any(zero).(type) {
	case string:
		return "string"
	case bool:
		return "bool"
	case int:
		return "int"
	case float32:
		return "float32"
	case float64:
		return "float64"
	}
	return ""
}

// parseScalar parses int, float32, float64, bool, or string.
func parseScalar[T any](v string, defaultValue T) (T, error) {
	t := scalarTypeName[T]()
	parser, ok := scalarParsers[t]
	if !ok {
		return defaultValue, errors.New("unsupported scalar type")
	}

	val, err := parser(v)
	if err != nil {
		return defaultValue, err
	}
	return any(val).(T), nil
}

func sliceElemTypeName[T any]() string {
	var zero T

	switch any(zero).(type) {
	case []string:
		return "string"
	case []bool:
		return "bool"
	case []int:
		return "int"
	case []float32:
		return "float32"
	case []float64:
		return "float64"
	}
	return ""
}

var sliceBuilders = map[string]func([]any) any{
	"string": func(v []any) any {
		out := make([]string, len(v))
		for i := range v {
			out[i] = v[i].(string)
		}
		return out
	},
	"bool": func(v []any) any {
		out := make([]bool, len(v))
		for i := range v {
			out[i] = v[i].(bool)
		}
		return out
	},
	"int": func(v []any) any {
		out := make([]int, len(v))
		for i := range v {
			out[i] = v[i].(int)
		}
		return out
	},
	"float32": func(v []any) any {
		out := make([]float32, len(v))
		for i := range v {
			out[i] = v[i].(float32)
		}
		return out
	},
	"float64": func(v []any) any {
		out := make([]float64, len(v))
		for i := range v {
			out[i] = v[i].(float64)
		}
		return out
	},
}

// parseSlice parses slices of string, int, float32, float64, bool.
func parseSlice[T any](v string, defaultValue T) (T, error) {
	elemType := sliceElemTypeName[T]()

	parser := scalarParsers[elemType]
	builder := sliceBuilders[elemType]

	parts := strings.Split(v, ",")
	tmp := make([]any, len(parts))

	for i, p := range parts {
		val, err := parser(strings.TrimSpace(p))
		if err != nil {
			return defaultValue, err
		}
		tmp[i] = val
	}

	return any(builder(tmp)).(T), nil
}

// parseValue parse a string into the given defaultValue type.
func parseValue[T any](v string, defaultValue T) (T, error) {
	if sliceElemTypeName[T]() != "" {
		return parseSlice(v, defaultValue)
	}
	return parseScalar(v, defaultValue)
}

// handleNoValue handle the case when a key has no value.
// If the default value is a bool, return true.
// Otherwise, return the default value.
func handleNoValue[T any](defaultValue T) (T, error) {
	var zero T
	switch any(zero).(type) {
	case bool:
		return any(true).(T), nil
	default:
		return defaultValue, nil
	}
}

// splitKV split a key-value pair into key and value.
func splitKV(s string) (key, value string, hasValue bool) {
	parts := strings.SplitN(s, "=", 2)
	if len(parts) == 1 {
		return parts[0], "", false
	}
	return parts[0], parts[1], true
}

// GetCmdlineValue returns the value of a cmdline property.
//
// T could be one of the following types:
//
//	var T string
//	var T []string
//	var T int
//	var T []int
//	var T float32
//	var T []float32
//	var T float64
//	var T []float64
//	var T bool
//	var T []bool
//
// Behavior:
//   - If no cmdline file is found, it returns an error.
//   - If the key is not found, it returns the default value.
//   - If the key is found and its value is not empty, it returns the value.
//   - If the key is found, defaultValue type is boolean, and its value is
//     empty, it returns true.
//   - If the key is found, defaultValue type is not boolean, and its value is
//     empty, it returns the default value.
//   - If the key is found and its value is not a valid type, it returns an error.
func GetCmdlineValue[T any](cmdline string, key string, defaultValue T) (T, error) {
	for p := range strings.FieldsSeq(string(cmdline)) {
		k, v, hasValue := splitKV(p)

		if k != key {
			continue
		}

		// Case: key without value.
		if !hasValue {
			return handleNoValue(defaultValue)
		}

		// Case: key=value.
		parsed, err := parseValue(v, defaultValue)
		if err != nil {
			return defaultValue, err
		}
		return parsed, nil
	}

	// Case: key not found.
	return defaultValue, nil
}

const CmdlineFilepath = "/proc/cmdline"
