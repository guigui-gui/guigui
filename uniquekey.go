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

// DataKey is a unique identifier for a model.
type DataKey uniqueKey

// GenerateDataKey generates a new DataKey.
func GenerateDataKey() DataKey {
	return DataKey(generateUniqueKey())
}

// EventKey is a unique identifier for an event.
type EventKey uniqueKey

// GenerateEventKey generates a new EventKey.
func GenerateEventKey() EventKey {
	return EventKey(generateUniqueKey())
}
