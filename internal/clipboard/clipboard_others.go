// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

//go:build android || ios || (!js && !windows && !unix)

package clipboard

func readAll() ([]byte, error) {
	return nil, nil
}

func writeAll(text []byte) error {
	return nil
}
