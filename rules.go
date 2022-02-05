package gorules

import (
	"github.com/quasilyte/go-ruleguard/dsl"
)

// Use xerrors everywhere! It provides additional stacktrace info!
//nolint:unused,deadcode,varnamelen
func xerrors(m dsl.Matcher) {
	m.Import("errors")
	m.Import("fmt")
	m.Import("golang.org/x/xerrors")

	m.Match("fmt.Errorf($*args)").
		Suggest("xerrors.New($args)")

	m.Match("fmt.Errorf($*args)").
		Suggest("xerrors.Errorf($args)")

	m.Match("errors.New($msg)").
		Where(m["msg"].Type.Is("string")).
		Suggest("xerrors.New($msg)")
}
