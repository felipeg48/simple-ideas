package cdp

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// Slots are machine-wide: every si process and every test package on the
// machine shares the same Slots lock files in the temp folder, so a full
// go test ./... (one process per package) starts no more one-shot Chromes
// at once than one si does. A slot is an exclusive lock on one file (flock
// on Unix, LockFileEx on Windows); the system drops it when the holder
// exits, so a crashed process never keeps a slot. Long-lived DevTools
// Chromes (Launch) take no slot, so a filming run never waits on itself.
var Slots = max(2, runtime.NumCPU()/4)

// slotDir is where the lock files live (a variable for the tests).
var slotDir = filepath.Join(os.TempDir(), "si-chrome-slots")

// slot is one held lock; release frees it.
type slot struct{ f *os.File }

func (s slot) release() {
	if s.f != nil {
		unlockFile(s.f)
		s.f.Close()
	}
}

// acquire waits for a free slot and reports how long it waited. Waiters
// try every slot in turn and sleep a short, random while between rounds,
// so no one waiter is starved by the others. When the folder cannot be
// used (a read-only temp), the run goes ahead without a slot.
func acquire() (slot, time.Duration) {
	start := time.Now()
	if err := os.MkdirAll(slotDir, 0o777); err != nil {
		return slot{}, 0
	}
	files := make([]*os.File, Slots)
	defer func() {
		for _, f := range files {
			if f != nil {
				f.Close()
			}
		}
	}()
	for {
		opened := 0
		for i := range files {
			if files[i] == nil {
				f, err := os.OpenFile(filepath.Join(slotDir, fmt.Sprintf("slot-%d.lock", i)), os.O_RDWR|os.O_CREATE, 0o666)
				if err != nil {
					continue
				}
				files[i] = f
			}
			opened++
			if tryLockFile(files[i]) {
				f := files[i]
				files[i] = nil
				return slot{f}, time.Since(start)
			}
		}
		if opened == 0 {
			return slot{}, 0
		}
		time.Sleep(50*time.Millisecond + rand.N(100*time.Millisecond))
	}
}

// busySlots counts the slots held right now (by this process or others).
func busySlots() int {
	n := 0
	for i := range Slots {
		f, err := os.OpenFile(filepath.Join(slotDir, fmt.Sprintf("slot-%d.lock", i)), os.O_RDWR|os.O_CREATE, 0o666)
		if err != nil {
			continue
		}
		if tryLockFile(f) {
			unlockFile(f)
		} else {
			n++
		}
		f.Close()
	}
	return n
}
