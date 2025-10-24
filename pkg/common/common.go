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
	"os"

	log "github.com/sirupsen/logrus"
)

func GetEnv(key, fallback string) string {
	log.WithFields(log.Fields{
		"key":      key,
		"fallback": fallback,
	}).Trace("start")
	defer log.Trace("end")

	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// Get returns the value pointed to by `ptr` if it is not nil,
// otherwise it returns the provided `fallback` value.
//
// This function is useful for handling optional pointer values safely,
// without needing explicit nil checks.
//
// Parameters:
//   - ptr:      A pointer to a value of type T (may be nil).
//   - fallback: Default value to return if ptr is nil.
//
// Example:
//
//	var flag *bool
//	result := Get(flag, true) // returns true since flag is nil
func Get[T any](ptr *T, fallback T) T {
	log.WithFields(log.Fields{
		"ptr":      ptr,
		"fallback": fallback,
	}).Trace("start")
	defer log.Trace("end")

	if ptr != nil {
		return *ptr
	}
	return fallback
}
