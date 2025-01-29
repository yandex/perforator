package zap

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type lazyCallField struct {
	key string
	fn  any
}

// MarshalLogObject implements zapcore.ObjectMarshaler to call function in a lazy manner
func (l lazyCallField) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	switch fn := l.fn.(type) {
	case zapcore.ObjectMarshalerFunc:
		return fn(encoder)
	case func() (any, error):
		val, err := fn()
		if err != nil {
			return err
		}
		return encoder.AddReflected(l.key, val)
	default:
		return fmt.Errorf("cannot encode lazy field of type %T", fn)
	}
}

// LazyCall creates a log field with lazy function
func LazyCall(key string, fn any) zap.Field {
	return zap.Field{
		Type:      zapcore.InlineMarshalerType,
		Interface: lazyCallField{key: key, fn: fn},
	}
}
