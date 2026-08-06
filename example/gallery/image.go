// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"embed"

	"github.com/guigui-gui/guigui/example/internal/resource"
)

//go:embed resource/*.png
var pngImages embed.FS

var theImageLoader = resource.NewImageLoader(pngImages)
