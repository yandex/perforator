package subpkg

type Quxer interface {
	Qux()
}

type quuxerUnexported interface{
	Quux(Quxer) Quxer
}

type Quuxer = quuxerUnexported
