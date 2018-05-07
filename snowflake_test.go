package snowflake_test

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/deckarep/golang-set"
	"github.com/stretchr/testify/assert"
	"github.com/ttlj/snowflake"
)

func oneWorker() (uint16, error) {
	return 1, nil
}

func getFlake() *snowflake.Node {
	var settings snowflake.Settings
	settings.StartTime = time.Now() // startTime is the current time
	//settings.StartTime = time.Date(2014, 1, 1, 0, 0, 0, 0, time.UTC)
	settings.WorkerID = oneWorker
	//var sf *SnowFlake
	sf, _ := snowflake.NewNode(settings)
	if sf == nil {
		panic("SnowFlake not created")
	}
	// startTime = toSnowFlakeTime(settings.StartTime)
	// //ip, _ := lower16BitPrivateIP()
	// machineID = uint64(321)
	return sf
}

func nextID(t *testing.T, sf *snowflake.Node) uint64 {
	id, err := sf.NextID()
	if err != nil {
		t.Fatalf("id not generated: %s", err)
	}
	return id
}

func TestInvalidMaskConfig(t *testing.T) {
	var parameters = []struct {
		t, m, s uint8
	}{
		{38, 8, 17},  // too big seq
		{21, 33, 8},  // too big worker id
		{39, 16, 16}, // too big overall
	}
	var st snowflake.Settings
	for _, tt := range parameters {
		mc := snowflake.MaskConfig{TimeBits: tt.t, WorkerBits: tt.m, SequenceBits: tt.s}
		fmt.Println(mc)
		_, err := snowflake.NewNode(st, mc)
		if err == nil {
			t.Errorf("expected error, got nil for %+v", mc)
		}
	}
}

func TestFlakeOnce(t *testing.T) {
	sf := getFlake()
	sleepTime := uint64(50)
	time.Sleep(time.Duration(sleepTime) * 10 * time.Millisecond)
	nextID(t, sf)
}

func TestFlakeTwice(t *testing.T) {
	sf := getFlake()
	id1, _ := sf.NextID()
	id2, _ := sf.NextID()
	assert.Equal(t, true, (id1 < id2), "ID Order Mismatch")
}

func TestSnowFlakeFor2Sec(t *testing.T) {
	sf := getFlake()
	var numID uint32
	var lastID uint64

	initial := time.Now().UnixNano()
	current := initial
	for current-initial < 2*1e9 {
		id, _ := sf.NextID()

		numID++

		if id <= lastID {
			t.Fatal("duplicated id")
		}
		lastID = id

		current = time.Now().UnixNano()
	}
	fmt.Println("number of id:", numID)
}

func TestFlakeList(t *testing.T) {
	sf := getFlake()
	idList, err := sf.NextIDs()
	if err != nil {
		t.Fatal("id list not generated")
	}
	lower := idList[0]
	upper := idList[255]
	assert.Equal(t, 256, len(idList), "Length of ID List should be 256")
	assert.Equal(t, 256, cap(idList), "Capacity of ID List should be 256")
	assert.Equal(t, true, (lower < upper), "ID Order Mismatch")
}

func TestFlakeInParallel(t *testing.T) {
	sf := getFlake()
	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU)
	fmt.Println("number of cpu:", numCPU)

	consumer := make(chan uint64)

	const numID = 10000
	generate := func() {
		for i := 0; i < numID; i++ {
			consumer <- nextID(t, sf)
		}
	}

	const numGenerator = 10
	for i := 0; i < numGenerator; i++ {
		go generate()
	}

	set := mapset.NewSet()
	for i := 0; i < numID*numGenerator; i++ {
		id := <-consumer
		if set.Contains(id) {
			t.Fatal("duplicated id")
		} else {
			set.Add(id)
		}
	}
	fmt.Println("number of id:", set.Cardinality())
}
