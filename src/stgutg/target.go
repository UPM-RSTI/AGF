// --- stgutg/target.go ---
package stgutg

import "sync"

var (
	supiToTarget = make(map[string][]byte)
	tgtMu        sync.RWMutex
)

// SetTargetGnb guarda el targetGnbIdBytes para una SUPI
func SetTargetGnb(supi string, target []byte) {
	tgtMu.Lock()
	supiToTarget[supi] = target
	tgtMu.Unlock()
}

// GetTargetGnb recupera el targetGnbIdBytes para una SUPI
func GetTargetGnb(supi string) ([]byte, bool) {
	tgtMu.RLock()
	t, ok := supiToTarget[supi]
	tgtMu.RUnlock()
	return t, ok
}
