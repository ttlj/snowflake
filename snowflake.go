package snowflake

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// custom epoch
var epoch = time.Date(2017, 11, 29, 0, 0, 0, 0, time.UTC).UnixNano()

// Settings configures Node
//
// StartTime is the time since which the SnowFlake time is defined as the elapsed time.
// If StartTime is 0, the start time of the SnowFlake is set to "2014-09-01 00:00:00 +0000 UTC".
// If StartTime is ahead of the now time, SnowFlake is not created.
//
// WorkerID returns the unique ID of the SnowFlake instance.
// If WorkerID returns an error, SnowFlake is not created.
// If WorkerID is nil, default WorkerID is used.
// Default WorkerID returns the lower 16 bits of the private IP address.
type Settings struct {
	StartTime time.Time
	WorkerID  func() (uint16, error)
}

// MaskConfig configures the structure of the generated ID.
// The sum of all bits must not exceed 63 bits.
type MaskConfig struct {
	TimeBits, WorkerBits, SequenceBits uint8
}

type bitmask struct {
	time    uint64
	machine uint32
	seq     uint16
}

// Node struct
type Node struct {
	mutex     *sync.Mutex
	epoch     int64
	tslast    int64 // most recent time when this snowflake was used
	timeMask  uint64
	machineID uint32
	maxSeq    uint32
	mask      MaskConfig
	seq       uint16
	shiftTime uint8
	bmask     bitmask
}

// NewNode creates new SnowFlake worker
func NewNode(st Settings, mc ...MaskConfig) (*Node, error) {
	mask := MaskConfig{39, 16, 8} // default
	if len(mc) > 0 {
		mask = mc[0]
		if !validMask(mask) {
			return nil, fmt.Errorf("invalid mask-config")
		}
	}

	sf := new(Node)
	sf.mutex = new(sync.Mutex)

	if st.StartTime.After(time.Now()) {
		return nil, fmt.Errorf("invalid start-time")
	}
	if st.StartTime.IsZero() {
		sf.epoch = epoch
	} else {
		sf.epoch = st.StartTime.UTC().UnixNano() / scaleFactor
	}

	var err error
	var val uint16
	if st.WorkerID == nil {
		val, err = Lower16BitPrivateIP()
	} else {
		val, err = st.WorkerID()
	}
	if err != nil {
		return nil, fmt.Errorf("cannot generate worker ID")
	}

	sf.machineID = uint32(val)
	sf.mask = mask
	sf.bmask.time = uint64(1)<<mask.TimeBits - 1
	sf.bmask.machine = uint32(1)<<mask.WorkerBits - 1
	sf.bmask.seq = uint16(1)<<mask.SequenceBits - 1
	sf.shiftTime = mask.WorkerBits + mask.SequenceBits
	sf.seq = sf.bmask.seq // why is it set to max value ?

	return sf, nil
}

// NextID generates a next unique ID.
// After the Node time overflows, NextID returns an error.
func (sf *Node) NextID() (uint64, error) {
	sf.mutex.Lock()
	defer sf.mutex.Unlock()
	sf.validateTime()
	if (sf.tslast - sf.epoch) >= 1<<sf.mask.TimeBits {
		return 0, errors.New("over the time limit")
	}
	// Time-MachineID-Sequence
	id := uint64(sf.tslast-sf.epoch)<<(sf.shiftTime) |
		uint64(sf.machineID)<<sf.mask.SequenceBits |
		uint64(sf.seq)
	return id, nil
}

// NextIDs returns block of IDs, with len 2^sequence-bits
func (sf *Node) NextIDs() ([]uint64, error) {
	sf.mutex.Lock()
	defer sf.mutex.Unlock()
	sf.validateTime()
	if (sf.tslast - sf.epoch) >= 1<<sf.mask.TimeBits {
		return nil, errors.New("over the time limit")
	}

	sf.seq = 0
	lst := make([]uint64, 0, sf.bmask.seq+1)
	base := uint64(sf.tslast-sf.epoch)<<(sf.shiftTime) |
		uint64(sf.machineID)<<sf.mask.SequenceBits |
		uint64(sf.seq)
	for sf.seq <= sf.bmask.seq {
		lst = append(lst, base+uint64(sf.seq))
		sf.seq = sf.seq + 1
	}
	return lst, nil
}

// ---- private -----

const scaleFactor = 1e6

func milliseconds() int64 {
	return time.Now().UnixNano() / scaleFactor
}

func validMask(mc MaskConfig) bool {
	s := mc.WorkerBits + mc.SequenceBits + mc.TimeBits
	if mc.WorkerBits > 32 || mc.SequenceBits > 16 || s > 63 {
		return false
	}
	return true
}

func (sf *Node) validateTime() {
	ts := milliseconds()
	if sf.tslast == ts {
		sf.seq = (sf.seq + 1) & sf.bmask.seq
		if sf.seq == 0 {
			for ts <= sf.tslast {
				ts = milliseconds()
			}
		}
	} else {
		sf.seq = 0
	}

	if ts < sf.tslast {
		log.Fatalf("time is moving backwards, waiting until %d\n", sf.tslast)
	}

	sf.tslast = ts
}
