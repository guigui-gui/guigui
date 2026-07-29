// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

//go:build android || ios || (!js && !unix && !windows)

package clipboard

func readContents() (Contents, error) {
	return Contents{}, nil
}

func writeContents(contents Contents) error {
	return nil
}
