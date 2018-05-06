package snowflake

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

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
	TimeBits, MachineIDBits, SequenceBits uint8
}

type bitmask struct {
	time    uint64
	machine uint32
	seq     uint16
}

// Node struct
type Node struct {
	mutex     *sync.Mutex
	startTime int64
	prevTime  int64 // most recent time when this snowflake was used
	seq       uint16
	machineID uint32
	mask      MaskConfig
	timeMask  uint64
	bmask     bitmask
	shiftTime uint8
	maxSeq    uint32
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
		sf.startTime = workerTime(time.Date(2017, 11, 29, 0, 0, 0, 0, time.UTC))
	} else {
		sf.startTime = workerTime(st.StartTime)
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
	sf.bmask.machine = uint32(1)<<mask.MachineIDBits - 1
	sf.bmask.seq = uint16(1)<<mask.SequenceBits - 1
	sf.shiftTime = mask.MachineIDBits + mask.SequenceBits
	sf.seq = sf.bmask.seq // why is it set to max value ?

	return sf, nil
}

// NextID generates a next unique ID.
// After the Node time overflows, NextID returns an error.
func (sf *Node) NextID() (uint64, error) {
	sf.mutex.Lock()
	defer sf.mutex.Unlock()
	sf.validateTime()
	return sf.toID()
}

// NextIDs returns block of IDs, with len 2^sequence-bits
func (sf *Node) NextIDs() ([]uint64, error) {
	sf.mutex.Lock()
	defer sf.mutex.Unlock()
	sf.validateTime()
	sf.seq = 0
	idList := make([]uint64, 0, sf.bmask.seq+1)
	for sf.seq <= sf.bmask.seq {
		id, err := sf.toID()
		if err != nil {
			return nil, err
		}
		idList = append(idList, id)
		sf.seq = (sf.seq + 1)
	}
	return idList, nil
}

// ---- private -----

const scaleFactor = 1e6

func validMask(mc MaskConfig) bool {
	s := mc.MachineIDBits + mc.SequenceBits + mc.TimeBits
	if mc.MachineIDBits > 32 || mc.SequenceBits > 16 || s > 63 {
		return false
	}
	return true
}

func (sf *Node) toID() (uint64, error) {
	if sf.prevTime >= 1<<sf.mask.TimeBits {
		return 0, errors.New("over the time limit")
	}
	// Time-MachineID-Sequence
	// return uint64(sf.prevTime)<<(sf.shiftTime) |
	// 	uint64(sf.seq)<<sf.mask.MachineIDBits |
	// 	uint64(sf.machineID), nil

	// Time-MachineID-Sequence
	return uint64(sf.prevTime)<<(sf.shiftTime) |
		uint64(sf.machineID)<<sf.mask.SequenceBits |
		uint64(sf.seq), nil
}

func (sf *Node) validateTime() {
	current := currentElapsedTime(sf.startTime)
	if sf.prevTime < current {
		// this is only executed the first time
		// this will be executed if the elapsedTime is not set correctly to current time
		sf.prevTime = current
		sf.seq = 0
	} else if sf.prevTime == current {
		sf.seq = (sf.seq + 1) & sf.bmask.seq
		if sf.seq == 0 {
			sf.prevTime++
			overtime := sf.prevTime - current
			time.Sleep(sleepTime((overtime)))
		}
	} else {
		log.Fatal("recent time can never be greater than current time")
	}
}

func workerTime(t time.Time) int64 {
	return t.UTC().UnixNano() / scaleFactor
}

func currentElapsedTime(startTime int64) int64 {
	return workerTime(time.Now()) - startTime
}

func sleepTime(overtime int64) time.Duration {
	return time.Duration(overtime)*10*time.Millisecond -
		time.Duration(time.Now().UTC().UnixNano()%scaleFactor)*time.Nanosecond
}
