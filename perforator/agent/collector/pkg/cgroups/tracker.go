package cgroups

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
)

const (
	CgroupScanPeriod = time.Minute
)

type LockType int

const (
	RLock LockType = iota
	WLock
	NoLock
)

type cgroupIDs struct {
	id   uint64
	name string
}

type cgroup struct {
	cgroupIDs
	event CgroupEventListener
	mutex sync.RWMutex
}

type TrackedCgroup struct {
	// in freezer hierarchy
	Name  string
	Event CgroupEventListener
}

// Tracks some event ...
// For example if cgroup was deleted in freezer hierarchy
//    and a new one was created with the same name. We continue to track it
// Also tracked events and reopened when IsStale() returns true

// One usecase for this is if we want to track perf events on cgroups.
// 	We want to reopen perf event when either perf event cgroup was recreated
// 	or freezer hierarchy cgroup was recreated

type Tracker struct {
	l         log.Logger
	nameCache nameCache

	mutex         sync.RWMutex
	cgroupsByID   map[uint64]*cgroup
	cgroupsByName map[string]*cgroup
}

// this function is used in tests with custom NameCache
func newTrackerImpl(l log.Logger, nameCache nameCache) (*Tracker, error) {
	return &Tracker{
		l:             l.WithName("CgroupTracker"),
		cgroupsByID:   make(map[uint64]*cgroup),
		cgroupsByName: make(map[string]*cgroup),
		nameCache:     nameCache,
	}, nil
}

type CgroupHint struct {
	Version    CgroupVersion `yaml:"version"`
	MountPoint string        `yaml:"mount_point"`
}

type CgroupHints struct {
	Strong *CgroupHint  `yaml:"strong,omitempty"`
	Weak   []CgroupHint `yaml:"weak,omitempty"`
}

type TrackerConfig struct {
	CgroupHints *CgroupHints `yaml:"cgroupfs_hints,omitempty"`
}

func NewTracker(l log.Logger, config *TrackerConfig) (*Tracker, error) {
	fs, err := findCgroups(config.CgroupHints)
	if err != nil {
		return nil, fmt.Errorf("cgroupfs not found: %w", err)
	}
	nameCache, err := newCgroupNameCache(fs)
	if err != nil {
		return nil, fmt.Errorf("failed to create name cache: %w", err)
	}

	return newTrackerImpl(l, nameCache)
}

func (t *Tracker) CgroupVersion() CgroupVersion {
	return t.nameCache.cgroupVersion()
}

func (t *Tracker) updateCgroupID(oldID uint64, newID uint64, lock bool) *cgroup {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	cgrp := t.cgroupsByID[oldID]
	if cgrp == nil {
		return nil
	}

	delete(t.cgroupsByID, oldID)
	t.cgroupsByID[newID] = cgrp

	cgrp.mutex.Lock()
	if !lock {
		defer cgrp.mutex.Unlock()
	}

	cgrp.id = newID
	return cgrp
}

func (t *Tracker) getCgroupByID(id uint64, lock LockType) *cgroup {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	cgrp := t.cgroupsByID[id]
	if cgrp != nil {
		if lock == WLock {
			cgrp.mutex.Lock()
		} else if lock == RLock {
			cgrp.mutex.RLock()
		}
	}
	return cgrp
}

func (t *Tracker) reopenCgroup(oldID uint64, newID uint64) error {
	var cgrp *cgroup

	if oldID != newID {
		cgrp = t.updateCgroupID(oldID, newID, true)
	} else {
		cgrp = t.getCgroupByID(oldID, WLock)
	}

	if cgrp == nil {
		return fmt.Errorf("no cgroup with id %d found", oldID)
	}
	defer cgrp.mutex.Unlock()

	t.l.Info(
		"Reopen cgroup",
		log.String("name", cgrp.name),
		log.UInt64("new_id", newID),
		log.UInt64("old_id", oldID),
	)

	cgrp.event.Close()
	return cgrp.event.Open(cgrp.name, cgrp.id)
}

func (t *Tracker) checkUpdateCgroup(cgrp *cgroup) (needsUpdate bool, newID uint64, err error) {
	cgrp.mutex.RLock()
	defer cgrp.mutex.RUnlock()

	newCgroupID, err := t.nameCache.addCgroup(cgrp.name)
	if err != nil {
		return false, cgrp.id, err
	}

	if newCgroupID == cgrp.id && !cgrp.event.IsStale() {
		return false, cgrp.id, nil
	}

	return true, newCgroupID, nil
}

type cgroupUpdate struct {
	oldID uint64
	newID uint64
}

func (t *Tracker) getCgroupsForUpdate() ([]cgroupUpdate, error) {
	result := []cgroupUpdate{}

	t.mutex.RLock()
	defer t.mutex.RUnlock()

	for cgroupID, cgrp := range t.cgroupsByID {
		update, newID, err := t.checkUpdateCgroup(cgrp)
		if err != nil {
			return nil, err
		}

		if update {
			result = append(
				result,
				cgroupUpdate{
					oldID: cgroupID,
					newID: newID,
				},
			)
		}
	}

	return result, nil
}

func (t *Tracker) checkUpdatedCgroups() error {
	updatedCgroups, err := t.getCgroupsForUpdate()
	if err != nil {
		return err
	}

	for _, cgroupids := range updatedCgroups {
		err := t.reopenCgroup(cgroupids.oldID, cgroupids.newID)
		if err != nil {
			t.l.Warn(
				"Failed to reopen cgroup",
				log.UInt64("old_id", cgroupids.oldID),
				log.UInt64("new_id", cgroupids.newID),
				log.Error(err),
			)
		}
	}

	return nil
}

func (t *Tracker) runTracker(ctx context.Context) error {
	tick := time.NewTicker(CgroupScanPeriod)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}

		t.l.Info("Check updated cgroups")
		err := t.checkUpdatedCgroups()
		if err != nil {
			t.l.Error("Failed to check update cgroups", log.Error(err))
		}
	}
}

func (t *Tracker) runNameCacheUpdater(ctx context.Context) error {
	tick := time.NewTicker(30 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}

		err := t.nameCache.populate()
		if err != nil {
			t.l.Error("Failed to update cgroupfs cache", log.Error(err))
		}
	}
}

func (t *Tracker) RunPoller(ctx context.Context) error {
	g, newCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return t.runTracker(newCtx)
	})

	g.Go(func() error {
		return t.runNameCacheUpdater(newCtx)
	})

	return g.Wait()
}

func (t *Tracker) CgroupFullName(id uint64) string {
	return t.nameCache.cgroupFullName(id)
}

func (t *Tracker) CgroupBaseName(id uint64) string {
	return t.nameCache.cgroupBaseName(id)
}

// Cgroup name must be in /sys/fs/cgroup/freezer hierarchy
// Thread-safe
func (t *Tracker) AddCgroup(newCgroup *TrackedCgroup, reopenEventIfExists bool) error {
	if newCgroup.Event == nil {
		return errors.New("tracked event must be non nil")
	}

	cgroupID, err := t.nameCache.addCgroup(newCgroup.Name)
	if err != nil {
		return err
	}

	mapsLocked := true
	t.mutex.Lock()
	defer func() {
		if mapsLocked {
			t.mutex.Unlock()
		}
	}()

	cgrp := t.cgroupsByName[newCgroup.Name]
	if cgrp != nil {
		if !reopenEventIfExists {
			return nil
		}

		cgrp.mutex.Lock()
		defer cgrp.mutex.Unlock()

		if cgroupID != cgrp.id {
			delete(t.cgroupsByID, cgrp.id)
			t.cgroupsByID[cgroupID] = cgrp
			cgrp.id = cgroupID
		}

		mapsLocked = false
		t.mutex.Unlock()

		err = newCgroup.Event.Open(cgrp.name, cgrp.id)
		if err != nil {
			return err
		}
		cgrp.event.Close()
		cgrp.event = newCgroup.Event

		return nil
	}

	err = newCgroup.Event.Open(newCgroup.Name, cgroupID)
	if err != nil {
		return err
	}

	cgrp = &cgroup{
		cgroupIDs: cgroupIDs{
			name: newCgroup.Name,
			id:   cgroupID,
		},
		event: newCgroup.Event,
	}
	t.cgroupsByID[cgroupID] = cgrp
	t.cgroupsByName[newCgroup.Name] = cgrp

	return nil
}

// Cgroup name must be in /sys/fs/cgroup/freezer hierarchy
// Thread-safe
func (t *Tracker) Delete(name string) error {
	t.mutex.Lock()
	locked := true
	defer func() {
		if locked {
			t.mutex.Unlock()
		}
	}()

	cgroup := t.cgroupsByName[name]
	if cgroup == nil {
		return nil
	}

	delete(t.cgroupsByName, name)
	delete(t.cgroupsByID, cgroup.id)

	locked = false
	t.mutex.Unlock()

	cgroup.mutex.Lock()
	defer cgroup.mutex.Unlock()

	cgroup.event.Close()

	return nil
}

// Track only passed cgrps.
// Must not be called concurrently
func (t *Tracker) TrackCgroups(cgrps []*TrackedCgroup) error {
	newCgroups := map[string]*TrackedCgroup{}
	for _, cgrp := range cgrps {
		newCgroups[cgrp.Name] = cgrp
	}

	deletedCgroups := []string{}
	t.mutex.RLock()
	for name := range t.cgroupsByName {
		if newCgroups[name] == nil {
			deletedCgroups = append(deletedCgroups, name)
		}
	}
	t.mutex.RUnlock()

	for _, name := range deletedCgroups {
		err := t.Delete(name)
		if err != nil {
			return err
		}
	}

	for _, cgrp := range cgrps {
		err := t.AddCgroup(cgrp, false)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Tracker) GetTrackedEvent(id uint64) CgroupEventListener {
	cgrp := t.getCgroupByID(id, RLock)
	if cgrp == nil {
		return nil
	}
	defer cgrp.mutex.RUnlock()
	return cgrp.event
}

func (t *Tracker) ForEachCgroup(callback func(event CgroupEventListener) error) error {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	for _, cgroup := range t.cgroupsByID {
		err := callback(cgroup.event)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Tracker) NumCgroupNames() int {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return int(len(t.cgroupsByName))
}

func (t *Tracker) NumCgroupIDs() int {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return int(len(t.cgroupsByID))
}
