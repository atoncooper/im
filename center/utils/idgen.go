package utils

import (
	"errors"
	"sync"
	"time"
)

const (
	timeBits    = 41
	bizBits     = 4
	dcBits      = 3
	machineBits = 6
	seqBits     = 9
	maxSeq      = 1<<seqBits - 1
	workerBits  = bizBits + dcBits + machineBits
	epoch       = 1704067200000
)

type GeneratorID struct {
	mu       sync.Mutex
	lastTs   int64
	sequence int64
	workerID int64
}

func NewGeneratorID(biz, dc, machine int) (*GeneratorID, error) {
	worker := (int64(biz)&((1<<bizBits)-1))<<(dcBits+machineBits) |
		(int64(dc)&((1<<dcBits)-1))<<machineBits |
		(int64(machine) & ((1 << machineBits) - 1))
	if worker >= 1<<workerBits {
		return nil, errors.New("workerID overflow")
	}
	return &GeneratorID{workerID: worker}, nil
}

func (g *GeneratorID) Next() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now().UnixMilli()
	if now < g.lastTs {
		time.Sleep(time.Duration(g.lastTs-now) * time.Millisecond)
		now = time.Now().UnixMilli()
	}

	if now == g.lastTs {
		g.sequence = (g.sequence + 1) & maxSeq
		if g.sequence == 0 {
			for now <= g.lastTs {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		g.sequence = 0
	}
	g.lastTs = now

	return (now-epoch)<<22 | g.workerID<<9 | g.sequence
}
