package jvmscanner

type bookmark struct {
	raw        string
	HeapIndex  int
	BlockIndex uint32
	CompileID  int32
}
