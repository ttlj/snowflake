package snowflake

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

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
	// mask := MaskConfig{35, 16, 12} // default
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
		sf.epoch = epoch // workerTime(time.Date(2017, 11, 29, 0, 0, 0, 0, time.UTC))
	} else {
		sf.epoch = workerTime(st.StartTime)
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

func milliseconds() int64 {
	return time.Now().UnixNano() / 1e6
}

func validMask(mc MaskConfig) bool {
	s := mc.WorkerBits + mc.SequenceBits + mc.TimeBits
	if mc.WorkerBits > 32 || mc.SequenceBits > 16 || s > 63 {
		return false
	}
	return true
}

func (sf *Node) toID() (uint64, error) {
	ts := sf.tslast - sf.epoch
	if ts >= 1<<sf.mask.TimeBits {
		return 0, errors.New("over the time limit")
	}
	// Time-MachineID-Sequence
	// return uint64(sf.tslast)<<(sf.shiftTime) |
	// 	uint64(sf.seq)<<sf.mask.WorkerBits |
	// 	uint64(sf.machineID), nil

	// Time-MachineID-Sequence
	return uint64(ts)<<(sf.shiftTime) |
		uint64(sf.machineID)<<sf.mask.SequenceBits |
		uint64(sf.seq), nil
}

func (sf *Node) validateTime() {
	ts := milliseconds()
	if sf.tslast < ts {
		// this is only executed the first time
		// this will be executed if the elapsedTime is not set correctly to ts time
		sf.tslast = ts
		sf.seq = 0
	} else if sf.tslast == ts {
		sf.seq = (sf.seq + 1) & sf.bmask.seq
		if sf.seq == 0 {
			sf.tslast++
			overtime := sf.tslast - ts
			time.Sleep(sleepTime((overtime)))
		}
	} else {
		log.Fatal("recent time can never be greater than ts time")
	}
}

func (sf *Node) toID2() (uint64, error) {
	if sf.tslast >= 1<<sf.mask.TimeBits {
		return 0, errors.New("over the time limit")
	}
	// Time-MachineID-Sequence
	// return uint64(sf.tslast)<<(sf.shiftTime) |
	// 	uint64(sf.seq)<<sf.mask.WorkerBits |
	// 	uint64(sf.machineID), nil

	// Time-MachineID-Sequence
	return uint64(sf.tslast)<<(sf.shiftTime) |
		uint64(sf.machineID)<<sf.mask.SequenceBits |
		uint64(sf.seq), nil
}

func (sf *Node) validateTime2() {
	// fmt.Println("vt")
	ts := currentElapsedTime(sf.epoch)
	if sf.tslast < ts {
		// this is only executed the first time
		// this will be executed if the elapsedTime is not set correctly to ts time
		sf.tslast = ts
		sf.seq = 0
	} else if sf.tslast == ts {
		sf.seq = (sf.seq + 1) & sf.bmask.seq
		if sf.seq == 0 {
			sf.tslast++
			overtime := sf.tslast - ts
			// fmt.Println("sleeping", overtime)
			time.Sleep(sleepTime((overtime)))
		}
	} else {
		log.Fatal("recent time can never be greater than ts time")
	}
}

func workerTime(t time.Time) int64 {
	return t.UnixNano() / scaleFactor
}

func currentElapsedTime(epoch int64) int64 {
	return workerTime(time.Now()) - epoch
}

func sleepTime(overtime int64) time.Duration {
	return time.Duration(overtime)*time.Millisecond -
		time.Duration(time.Now().UnixNano()%scaleFactor)*time.Nanosecond
}

// func init() {
// 	defer profile.Start().Stop()
// }
