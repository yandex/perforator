package empty_interface

//go:generate mockgen -package empty_interface -destination mock.go -source input.go -write_package_comment=false

type Empty interface{}
