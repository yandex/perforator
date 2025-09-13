package empty_interface

//go:generate mockgen -package empty_interface -destination mock.go -source input.go -copyright_file=mock_copyright_header

type Empty interface{} // migrating interface{} -> any does not resolve to an interface type dropping test generation added in b391ab3
