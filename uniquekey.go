// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package guigui

import (
	"sync/atomic"
)

type uniqueKey struct {
	v int64
}

var theUniqueKey atomic.Int64

func generateUniqueKey() uniqueKey {
	return uniqueKey{theUniqueKey.Add(1)}
}

// ModelKey is a unique identifier for a model.
type ModelKey uniqueKey

// GenerateModelKey generates a new ModelKey.
func GenerateModelKey() ModelKey {
	return ModelKey(generateUniqueKey())
}

// EventKey is a unique identifier for an event.
type EventKey uniqueKey

// GenerateEventKey generates a new EventKey.
func GenerateEventKey() EventKey {
	return EventKey(generateUniqueKey())
}
