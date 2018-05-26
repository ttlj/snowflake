// Package snowflake implements Twitter's SnowFlake algorithm.
package snowflake

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// default custom epoch
var epoch = time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano() / scaleFactor

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
// The sum of all bits must not exceed 63.
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
	mutex       *sync.Mutex
	epoch       int64
	tslast      int64 // most recent time when this snowflake was used
	timeMask    uint64
	machineID   uint32
	maxSeq      uint32
	mask        MaskConfig
	seq         uint16
	shiftTime   uint8
	shiftWorker uint8
	bmask       bitmask
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
	sf.shiftWorker = mask.SequenceBits
	sf.seq = sf.bmask.seq // why is it set to max value ?

	return sf, nil
}

// NextID generates a next unique ID.
// Returns an error when Node's time overflows.
// It terminates if it detects backwards moving time.
func (sf *Node) NextID() (uint64, error) {
	sf.mutex.Lock()
	defer sf.mutex.Unlock()
	sf.validateTime()
	return sf.toID()
}

type interval struct {
	lower, upper uint64
}

func (sf *Node) intervals(size uint16) ([]interval, error) {
	sf.mutex.Lock()
	defer sf.mutex.Unlock()
	top := sf.bmask.seq
	bulk := size
	if size == 0 {
		bulk = top
	}
	var count uint16
	lst := make([]interval, 0, 4)
	for count < bulk {
		var err error
		var ii interval
		sf.validateTime()

		rest := bulk - count
		ii.lower, err = sf.toID()
		if err != nil {
			return nil, err
		}

		avail := top - sf.seq
		if avail >= rest {
			sf.seq += rest - 1
		} else {
			sf.seq = top
		}
		ii.upper, err = sf.toID()
		if err != nil {
			return nil, err
		}

		count += uint16(ii.upper - ii.lower + 1)
		lst = append(lst, ii)
		if size == 0 {
			break
		}
	}
	return lst, nil
}

// NextIDRange returns lower and upper identifiers of a range.
// The size is up to SequenceBits^2 - 1.
func (sf *Node) NextIDRange() (uint64, uint64, error) {
	sf.mutex.Lock()
	defer sf.mutex.Unlock()
	sf.validateTime()
	lower, err := sf.toID()
	if err != nil {
		return 0, 0, err
	}
	sf.seq = sf.bmask.seq
	upper, err := sf.toID()
	if err != nil {
		return 0, 0, err
	}
	return lower, upper, nil
}

// NextIDRangeFill returns range as list of identifiers
func (sf *Node) NextIDRangeFill() ([]uint64, error) {
	lower, upper, err := sf.NextIDRange()
	if err != nil {
		return nil, err
	}
	lst := make([]uint64, 0, sf.bmask.seq+1)
	for i := lower; i <= upper; i++ {
		lst = append(lst, i)
	}
	return lst, nil
}

// NextIDBatch return list of identifiers of requested size
func (sf *Node) NextIDBatch(size int) ([]uint64, error) {
	lst := make([]uint64, 0, sf.bmask.seq+1)
	for i := 0; i < size; i++ {
		id, err := sf.NextID()
		if err != nil {
			return nil, err
		}
		lst = append(lst, id)
	}
	return lst, nil
}

// ---- private -----

const scaleFactor = 1e6

func milliseconds() int64 {
	return time.Now().UnixNano() / scaleFactor
}

func (sf *Node) toID() (uint64, error) {
	if (sf.tslast - sf.epoch) >= 1<<sf.mask.TimeBits {
		return 0, errors.New("over the time limit")
	}
	// Time-MachineID-Sequence
	return uint64(sf.tslast-sf.epoch)<<(sf.shiftTime) |
		uint64(sf.machineID)<<sf.shiftWorker |
		uint64(sf.seq), nil
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
	if ts < sf.tslast {
		log.Fatalf("time is moving backwards, waiting until %d\n", sf.tslast)
	}

	if ts == sf.tslast {
		sf.seq = (sf.seq + 1) & sf.bmask.seq
		if sf.seq == 0 {
			for ts <= sf.tslast {
				ts = milliseconds()
			}
		}
	} else {
		sf.seq = 0
	}

	sf.tslast = ts
}
