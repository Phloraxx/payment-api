package v4web

import (
	"embed"
	"io/fs"
)

// dist is populated by the v4 frontend build before the production v4 binary
// is compiled. A placeholder index is committed so Go tests can compile
// without Node being installed.
//
//go:embed all:dist
var embedded embed.FS

func Assets() fs.FS {
	sub, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
