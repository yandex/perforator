package procfs

import (
	"io/fs"
	"syscall"
)

type namespaces struct {
	p *process
}

func (n *namespaces) GetPidInode() (uint64, error) {
	return n.getNsInode("pid")
}

func (n *namespaces) getNsInode(ns string) (uint64, error) {
	path := n.p.child("ns/" + ns)

	stat, err := fs.Stat(n.p.fs, path)
	if err != nil {
		return 0, err
	}

	return stat.Sys().(*syscall.Stat_t).Ino, nil
}
