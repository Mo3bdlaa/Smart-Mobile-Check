// Package smartcheck exposes the files that are compiled into the executable:
// the device specification database and the web user interface.
package smartcheck

import "embed"

// Specs holds the built-in device specification database (specs/**.json).
//
//go:embed specs
var Specs embed.FS

// Web holds the user interface (HTML/CSS/JS) served by the local app server.
//
//go:embed web
var Web embed.FS
