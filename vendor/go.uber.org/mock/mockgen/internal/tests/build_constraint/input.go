package empty_interface

//go:generate mockgen -package empty_interface -destination mock.go -source input.go "-build_constraint=(linux && 386) || (darwin && !cgo) || usertag"

type Empty interface{}
