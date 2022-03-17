package threadsafe_ulid

import (
	"encoding/binary"
	"github.com/oklog/ulid/v2"
"math/rand"
"sync"
"time"
)

type ThreadSafeUlid struct {
	Time       time.Time
	safe       safeMonotonicReader
}

func NewThreadSafeUlid () ThreadSafeUlid {
	t := time.Now()
	return ThreadSafeUlid{
		Time:      t,
		safe:      safeMonotonicReader{MonotonicReader: ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)},
	}
}

func (u *ThreadSafeUlid) NewUlid () (ulid.ULID, error) {
	return ulid.New(ulid.Timestamp(u.Time), u.safe)
}

func (u *ThreadSafeUlid) NewSeededEntropy (seed string) {
	seedBytes := []byte(seed)
	eightByteHash := seedBytes[0:7]
	seedInt, _ := binary.Varint(eightByteHash)
	u.safe = safeMonotonicReader{MonotonicReader: ulid.Monotonic(rand.New(rand.NewSource(seedInt)), 0)}
}

type safeMonotonicReader struct {
	mtx sync.Mutex
	ulid.MonotonicReader
}

func (r *safeMonotonicReader) MonotonicRead(ms uint64, p []byte) (err error) {
	r.mtx.Lock()
	err = r.MonotonicReader.MonotonicRead(ms, p)
	r.mtx.Unlock()
	return err
}
