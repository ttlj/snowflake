package snowflake_test

import (
	"fmt"
	"testing"

	"github.com/ttlj/snowflake"
)

var testMasks = []snowflake.MaskConfig{
	{TimeBits: 45, WorkerBits: 10, SequenceBits: 8},
	{TimeBits: 44, WorkerBits: 10, SequenceBits: 9},
	{TimeBits: 43, WorkerBits: 10, SequenceBits: 10},
	{TimeBits: 42, WorkerBits: 10, SequenceBits: 11},
	{TimeBits: 41, WorkerBits: 10, SequenceBits: 12},
}

func setBench(mc snowflake.MaskConfig) (*snowflake.Node, string) {
	sf := getFlake(mc)
	name := fmt.Sprintf("seq_%d", mc.SequenceBits)
	return sf, name
}

func BenchmarkNextID(b *testing.B) {
	for _, tc := range testMasks {
		sf, name := setBench(tc)
		b.ResetTimer()
		b.Run(name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				sf.NextID()
			}
		})
	}
}

func BenchmarkNextIDs(b *testing.B) {
	for _, tc := range testMasks {
		sf, name := setBench(tc)
		b.ResetTimer()
		b.Run(name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				sf.NextIDs()
			}
		})
	}
}
