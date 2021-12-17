package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"testing"
	"time"

	RosettaTypes "github.com/coinbase/rosetta-sdk-go/types"

	"github.com/maticnetwork/polygon-rosetta/configuration"
	"github.com/maticnetwork/polygon-rosetta/polygon"

	"github.com/stretchr/testify/assert"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")

func main() {
	blockNum := flag.Int64("block", -1, "the block number whose transactions to trace")
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	if *blockNum == -1 {
		log.Fatalf("Must specify a --block")
	}

	cfg, err := configuration.LoadConfiguration()
	if err != nil {
		log.Fatalf("Couldn't load configuration: %v", err)
	}

	c1, err := polygon.NewClient(&polygon.ClientConfig{
		URL:              cfg.BorURL,
		ChainConfig:      cfg.Params,
		SkipAdminCalls:   cfg.SkipGethAdmin,
		Headers:          cfg.GethHeaders,
		BurntContract:    cfg.BurntContract,
		LeanTraces:       false,
		CustomTracerPath: "../call_tracer.js",
	})
	if err != nil {
		log.Fatalf("Couldn't create default client: %v", err)
	}
	c2, err := polygon.NewClient(&polygon.ClientConfig{
		URL:              cfg.BorURL,
		ChainConfig:      cfg.Params,
		SkipAdminCalls:   cfg.SkipGethAdmin,
		Headers:          cfg.GethHeaders,
		BurntContract:    cfg.BurntContract,
		LeanTraces:       true,
		CustomTracerPath: "../call_tracer_lean.js",
	})
	if err != nil {
		log.Fatalf("Couldn't create client with legacy tracer: %v", err)
	}

	for i := 0; i < 10000; i++ {
		checkBlock(c1, c2, blockNum)
		*blockNum--
	}
}

var total1, total2 time.Duration

func checkBlock(c1, c2 *polygon.Client, blockNum *int64) {
	fmt.Printf("Checking block: %v\n", *blockNum)
	id := &RosettaTypes.PartialBlockIdentifier{
		Index: blockNum,
	}
	ctx := context.Background()

	start := time.Now()
	b1, err := c1.Block(ctx, id)
	if err != nil {
		log.Fatalf("Failed to get block with client 1: %v", err)
	}
	end := time.Now()
	elapsed1 := end.Sub(start)
	total1 += elapsed1

	b2, err := c2.Block(ctx, id)
	if err != nil {
		log.Fatalf("Failed to get block with client 2: %v", err)
	}
	elapsed2 := time.Now().Sub(end)
	total2 += elapsed2

	fmt.Printf("request times: %v %v ::: %v %v\n", elapsed1, elapsed2, total1, total2)
	if elapsed1 > time.Second*2 {
		fmt.Printf("WARNING: REQUEST TOOK MORE THAN TWO SECONDS FOR C1: %v\n", *blockNum)
	}
	if elapsed2 > time.Second*2 {
		fmt.Printf("WARNING: REQUEST TOOK MORE THAN TWO SECONDS FOR C2: %v\n", *blockNum)
	}

	if len(b1.Transactions) != len(b2.Transactions) {
		log.Fatalf("transaction counts differ")
	}

	t := &testing.T{}

	for i, tx1 := range b1.Transactions {
		tx2 := b2.Transactions[i]
		ops1 := tx1.Operations
		ops2 := tx2.Operations
		if len(ops1) != len(ops2) {
			log.Fatalf("op counts differ")
		}
		for j, op1 := range ops1 {
			op2 := ops2[j]
			if !assert.Equal(t, op1, op2) {
				log.Fatalf("ops not equal %v %v", op1.Amount.Value, op2.Amount.Value)
			}
		}
	}
}
