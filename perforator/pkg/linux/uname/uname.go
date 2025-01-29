package uname

import (
	"bytes"
	"syscall"
)

func stringFromBytes(ints []int8) string {
	b := make([]byte, len(ints))
	for i, value := range ints {
		b[i] = byte(value)
	}
	b = bytes.TrimRight(b, "\x00")
	return string(b)
}

type Uname struct {
	SystemName string
	NodeName   string
	Release    string
	Version    string
	Machine    string
	DomainName string
}

func Load() (*Uname, error) {
	utsname := syscall.Utsname{}
	err := syscall.Uname(&utsname)
	if err != nil {
		return nil, err
	}

	return &Uname{
		SystemName: stringFromBytes(utsname.Sysname[:]),
		NodeName:   stringFromBytes(utsname.Nodename[:]),
		Release:    stringFromBytes(utsname.Release[:]),
		Version:    stringFromBytes(utsname.Version[:]),
		Machine:    stringFromBytes(utsname.Machine[:]),
		DomainName: stringFromBytes(utsname.Domainname[:]),
	}, nil
}

func SystemRelease() (string, error) {
	uname, err := Load()
	if err != nil {
		return "", err
	}
	return uname.Release, nil
}
