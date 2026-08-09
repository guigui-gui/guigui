// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"embed"

	"github.com/guigui-gui/guigui/example/internal/resource"
)

//go:embed resource/*.png
var imageResource embed.FS

var theImageLoader = resource.NewImageLoader(imageResource)
