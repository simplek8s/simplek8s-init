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

package common

import (
	"slices"

	log "github.com/sirupsen/logrus"
)

// UpdateOrAppend returns a new slice where `newItem` replaces an existing element
// that matches according to the `equals` function, or is appended if no match is found.
//
// This function makes a shallow copy of the input slice before modifying it,
// ensuring the original slice remains unchanged.
//
// Parameters:
//   - items:    Original slice of elements.
//   - newItem:  Element to insert or update.
//   - equals:   Function used to determine equality between elements.
//
// Example:
//
//	newList := UpdateOrAppend(users, newUser, func(a, b User) bool {
//	    return a.Name == b.Name
//	})
func UpdateOrAppend[T any](items []T, newItem T, equals func(a, b T) bool) []T {
	log.WithFields(log.Fields{
		"items":   items,
		"newItem": newItem,
		"equals":  equals,
	}).Trace("start")
	defer log.Trace("end")

	newItems := slices.Clone(items)

	if i := slices.IndexFunc(items, func(x T) bool {
		return equals(x, newItem)
	}); i >= 0 {
		newItems[i] = newItem
		return newItems
	}

	return append(newItems, newItem)
}
