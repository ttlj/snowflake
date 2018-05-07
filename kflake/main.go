package main

import (
	"crypto/rand"
	"flag"
	"log"
	"math/big"

	"github.com/ttlj/snowflake"
	"github.com/ttlj/snowflake/restful"
)

const port = ":3080"

func main() {

	// TODO: choice-flag
	var wid string
	flag.StringVar(&wid, "workerid", "", "worker-id type: {podid|podip}")
	flag.Parse()

	// Init snowflake
	st := snowflake.Settings{}
	switch wid {
	case "podid":
		st.WorkerID = snowflake.K8sPodID
	case "podip":
		st.WorkerID = snowflake.EnvVarIPWorkerID
	default:
		st.WorkerID = randomWorkerID
	}
	mc := snowflake.MaskConfig{TimeBits: 41, WorkerBits: 10, SequenceBits: 12}
	node, err := snowflake.NewNode(st, mc)
	if node == nil {
		log.Fatal("failed to initialize snowflake: ", err)
	}

	// Start engine
	e := &restful.Env{Flake: node}
	r := restful.NewEngine(e)
	if err := r.Run(port); err != nil {
		log.Fatal("failed to run server: ", err)
	}
}

func randomWorkerID() (uint16, error) {
	rval, err := rand.Int(rand.Reader, big.NewInt(10))
	n := rval.Int64()
	return uint16(n), err
}
