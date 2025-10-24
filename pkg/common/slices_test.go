// Copyright 2022 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
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

package common_test

import (
	"slices"
	"testing"

	"simplek8s/pkg/common"
)

// A simple struct for testing purposes.

func TestUpdateOrAppend(t *testing.T) {
	type user struct {
		Name string
		Age  int
	}

	tests := []struct {
		name     string
		input    []user
		newItem  user
		equals   func(a, b user) bool
		expected []user
	}{
		{
			name: "replace existing element",
			input: []user{
				{Name: "Alice", Age: 20},
				{Name: "Bob", Age: 30},
			},
			newItem: user{Name: "Alice", Age: 25},
			equals: func(a, b user) bool {
				return a.Name == b.Name
			},
			expected: []user{
				{Name: "Alice", Age: 25},
				{Name: "Bob", Age: 30},
			},
		},
		{
			name: "append when not found",
			input: []user{
				{Name: "Alice", Age: 20},
			},
			newItem: user{Name: "Bob", Age: 30},
			equals: func(a, b user) bool {
				return a.Name == b.Name
			},
			expected: []user{
				{Name: "Alice", Age: 20},
				{Name: "Bob", Age: 30},
			},
		},
		{
			name:     "append to empty slice",
			input:    []user{},
			newItem:  user{Name: "Alice", Age: 20},
			equals:   func(a, b user) bool { return a.Name == b.Name },
			expected: []user{{Name: "Alice", Age: 20}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := slices.Clone(tt.input)
			result := common.UpdateOrAppend(tt.input, tt.newItem, tt.equals)

			if !slices.Equal(result, tt.expected) {
				t.Errorf("expected %+v, got %+v", tt.expected, result)
			}

			// Ensure original slice was not modified
			if !slices.Equal(tt.input, original) {
				t.Errorf("original slice was modified: expected %+v, got %+v", original, tt.input)
			}
		})
	}
}
