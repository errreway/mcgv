//go:build tools
// +build tools

package tools

import (
	_ "github.com/jstemmer/go-junit-report/v2"
	_ "golang.org/x/tools/cmd/stringer"
	_ "honnef.co/go/tools/cmd/staticcheck"
)
