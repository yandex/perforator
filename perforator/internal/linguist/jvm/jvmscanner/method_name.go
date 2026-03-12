package jvmscanner

import (
	"fmt"

	"github.com/yandex/perforator/perforator/internal/linguist/jvm/jvmbindings"
)

func readMethodName(method jvmbindings.Method, limit int) ([]byte, error) {
	j := jvmbindings.ObjectPtr(method).JVM()

	constMethod, err := method.ConstMethod()
	if err != nil {
		return nil, fmt.Errorf("failed to get constMethod: %w", err)
	}
	constantPool, err := constMethod.Consts()
	if err != nil {
		return nil, fmt.Errorf("failed to get constantPool: %w", err)
	}
	nameIndex, err := constMethod.NameIndex()
	if err != nil {
		return nil, fmt.Errorf("failed to get nameIndex: %w", err)
	}
	selfNameSym, err := constantPool.At(nameIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to get method name symbol: %w", err)
	}
	poolHolder, err := constantPool.PoolHolder()
	if err != nil {
		return nil, fmt.Errorf("failed to get method class: %w", err)
	}
	klassNameSym, err := poolHolder.Name()
	if err != nil {
		return nil, fmt.Errorf("failed to get class name symbol: %w", err)
	}

	selfNameLen, err := selfNameSym.Length()
	if err != nil {
		return nil, fmt.Errorf("failed to get method name length: %w", err)
	}
	selfNamePtr := selfNameSym.Body()
	selfName, err := j.ReadBytes(selfNamePtr, min(int(selfNameLen), limit))
	if err != nil {
		return nil, fmt.Errorf("failed to read method name: %w", err)
	}
	if int(selfNameLen)+1 >= limit {
		return selfName, nil
	}
	klassNameLen, err := klassNameSym.Length()
	if err != nil {
		return nil, fmt.Errorf("failed to get class name length: %w", err)
	}
	klassNamePtr := klassNameSym.Body()
	// TODO: we can be more flexible with limit here (i.e. read much more and then compress like j.u.c.Executor etc)
	klassName, err := j.ReadBytes(klassNamePtr, min(int(klassNameLen), limit-1-int(selfNameLen)))
	if err != nil {
		return nil, fmt.Errorf("failed to read class name: %w", err)
	}
	res := append(klassName, byte('.'))
	return append(res, selfName...), nil
}
