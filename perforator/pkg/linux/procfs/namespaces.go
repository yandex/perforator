package procfs

import (
	"io/fs"
	"syscall"

	"github.com/yandex/perforator/perforator/pkg/linux"
)

type namespaces struct {
	p *Process
}

func (n *namespaces) GetPidInode() (linux.PIDNamespaceInode, error) {
	return n.getNsInode("pid")
}

func (n *namespaces) getNsInode(ns string) (linux.PIDNamespaceInode, error) {
	path := n.p.child("ns/" + ns)

	stat, err := fs.Stat(n.p.fs, path)
	if err != nil {
		return linux.PIDNamespaceInode(0), err
	}

	return linux.PIDNamespaceInode(stat.Sys().(*syscall.Stat_t).Ino), nil
}
