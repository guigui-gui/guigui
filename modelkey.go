// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package guigui

import (
	"sync/atomic"
)

// ModelKey is a unique identifier for a model.
type ModelKey struct {
	v int64
}

var theModelKey atomic.Int64

// GenerateModelKey generates a new ModelKey.
func GenerateModelKey() ModelKey {
	return ModelKey{theModelKey.Add(1)}
}
