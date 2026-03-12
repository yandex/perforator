package jvmbindings

type ObjectPtr struct {
	addr uintptr
	j    *JVM
}

func (op ObjectPtr) JVM() *JVM {
	return op.j
}

func (op ObjectPtr) Addr() uintptr {
	return op.addr
}

type CodeHeap ObjectPtr

type HeapBlock ObjectPtr

type CodeBlob ObjectPtr

type Method ObjectPtr

type ConstMethod ObjectPtr

type Zstring ObjectPtr

type ConstantPool ObjectPtr

type Symbol ObjectPtr

type Klass ObjectPtr

type StubQueue ObjectPtr

type CodeHeapArray ObjectPtr

type VirtualSpace ObjectPtr
