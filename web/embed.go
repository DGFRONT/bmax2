// Package web embeds the mini app front end.
package web

import (
	"embed"
	"io/fs"
)

//go:embed index.html css js
var files embed.FS

// Files returns the front end rooted at the web directory.
func Files() fs.FS { return files }
