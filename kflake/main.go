package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ttlj/snowflake"
	"github.com/ttlj/snowflake/restful"
)

const port = ":3080"

type intslice []uint8

var maskints intslice

func main() {

	// TODO: choice-flag
	var wid string
	flag.StringVar(&wid, "t", "", "worker-id type: {podid|podip|random}; default: podid")
	flag.Var(&maskints, "m", "comma separated MaskConfig values {time,worker,sequence} bits; default: 41,10,12")
	flag.Parse()

	node := initFlake(wid)

	// Start engine
	e := &restful.Env{Flake: node}
	r := restful.NewEngine(e)

	srv := &http.Server{
		Addr:    port,
		Handler: r,
	}

	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	log.Println("Server exiting")
}

func initFlake(wid string) *snowflake.Node {
	st := snowflake.Settings{}
	switch wid {
	case "podid":
		st.WorkerID = snowflake.K8sPodID
	case "podip":
		st.WorkerID = snowflake.EnvVarIPWorkerID
	case "random":
		st.WorkerID = randomWorkerID
	default:
		st.WorkerID = snowflake.K8sPodID
	}
	var mc snowflake.MaskConfig
	mc = snowflake.MaskConfig{TimeBits: 41, WorkerBits: 10, SequenceBits: 12}
	if len(maskints) > 0 {
		mc = snowflake.MaskConfig{
			TimeBits: maskints[0], WorkerBits: maskints[1], SequenceBits: maskints[2]}
	}
	node, err := snowflake.NewNode(st, mc)
	if node == nil {
		log.Fatal("failed to initialize snowflake: ", err)
	}
	return node
}

func randomWorkerID() (uint16, error) {
	rval, err := rand.Int(rand.Reader, big.NewInt(10))
	n := rval.Int64()
	return uint16(n), err
}

func (i *intslice) String() string {
	return fmt.Sprintf("%d", *i)
}

func (i *intslice) Set(value string) error {
	parts := strings.Split(value, ",")
	if len(parts) != 3 {
		return fmt.Errorf("Invalid MaskConfig")
	}
	for _, item := range parts {
		tmp, err := strconv.ParseUint(item, 10, 8)
		if err != nil {
			return err
		}
		*i = append(*i, uint8(tmp))
	}
	return nil
}
