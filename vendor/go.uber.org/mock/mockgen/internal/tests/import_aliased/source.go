package import_aliased

import (
	definitionAlias "context"
)

//go:generate mockgen -package import_aliased -destination source_mock.go -source=source.go -imports definitionAlias=context

type S interface {
	M(ctx definitionAlias.Context)
}
