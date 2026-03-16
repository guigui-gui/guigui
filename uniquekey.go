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

// EnvKey is a unique identifier for an environment value.
type EnvKey uniqueKey

// GenerateEnvKey generates a new EnvKey.
func GenerateEnvKey() EnvKey {
	return EnvKey(generateUniqueKey())
}

// DataKey is a deprecated alias for [EnvKey].
//
// Deprecated: Use [EnvKey] instead.
type DataKey = EnvKey

// GenerateDataKey is a deprecated alias for [GenerateEnvKey].
//
// Deprecated: Use [GenerateEnvKey] instead.
func GenerateDataKey() DataKey {
	return GenerateEnvKey()
}

// EventKey is a unique identifier for an event.
type EventKey uniqueKey

// GenerateEventKey generates a new EventKey.
func GenerateEventKey() EventKey {
	return EventKey(generateUniqueKey())
}
