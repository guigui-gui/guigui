// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package guigui

import "math/rand/v2"

// ModelKey is a unique identifier for a model.
type ModelKey int64

// GenerateModelKey generates a new ModelKey.
func GenerateModelKey() ModelKey {
	// A key doesn't have to be cryptographically random. It just needs to be unique.
	return ModelKey(rand.Int64())
}
