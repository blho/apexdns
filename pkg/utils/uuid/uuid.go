package uuid

import (
	"sync"

	"github.com/pborman/uuid"
)

var lock sync.Mutex
var lastUUID uuid.UUID

func Get() string {
	lock.Lock()
	defer lock.Unlock()
	result := uuid.NewUUID()
	// The UUID package is naive and can generate identical UUIDs if the
	// time interval is quick enough.
	// The UUID uses 100 ns increments so it's short enough to actively
	// wait for a new value.
	for uuid.Equal(lastUUID, result) {
		result = uuid.NewUUID()
	}
	lastUUID = result
	return result.String()
}

func IsValid(uid string) bool {
	return uuid.Parse(uid) != nil
}
