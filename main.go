package main

import (
	"conflictSim/utils"
	"flag"
	"fmt"
	"os"
	"os/signal"

	conflictwriter "conflictSim/conflictWriter"

	xdcrLog "github.com/couchbase/goxdcr/v8/log"
)

func main() {
	opts := parseArgs()
	finCh := make(chan bool)
	go monitorForInterrupt(finCh)
	logCtx := utils.ProcessDebugMode(opts.DebugMode)
	logger := xdcrLog.NewLogger("ConflictWriter", logCtx)

	cw := conflictwriter.NewConflictWriter(logger, opts, finCh)
	err := cw.Init()
	if err != nil {
		close(finCh)
		return
	}

	cw.Start()
}

func parseArgs() conflictwriter.InOptions {
	// compulsory input
	source := flag.String("source", "", "Source host address (compulsory)")
	target := flag.String("target", "", "Target host address (compulsory)")
	srcBucket := flag.String("srcBucket", "", "Source bucket name (compulsory)")
	tgtBucket := flag.String("tgtBucket", "", "Target bucket name (compulsory)")
	srcPassword := flag.String("srcPassword", "", "Source password (compulsory)")
	srcUsername := flag.String("srcUsername", "", "Source username (compulsory)")
	tgtPassword := flag.String("tgtPassword", "", "Target password (compulsory)")
	tgtUsername := flag.String("tgtUsername", "", "Target username (compulsory)")
	numConflicts := flag.Int("numConflicts", -1, "Number of conflicts to be generated (compulsory)")

	// optional inputs with default values
	distribution := flag.String("distribution", "step", "Distribution of conflict writes: step, normrandom, exprandom, pseudorandom  (optional, default=step)")
	batchSize := flag.Int("batchSize", 1, "Batch size of conflict writes i.e. number of conflicts to be simulated for a single conflict generation (optional, default=1)")
	gap := flag.Int("gap", 2000, "Gap in milliseconds between conflict batch writes (optional, default=2000)")
	workers := flag.Int("workers", 10, "Number of workers per cluster. Each worker simulates an application connecting to a cluster (optional, default=10)")
	debugMode := flag.Int("debugMode", 0, "Turn on xdcr debug logging and SDK verbose logging. 0 is for off. 1 is for xdcr debug logging. 2 is for xdcr debug logging + SDK versbose logging (optional, default=0 or off)")

	flag.Parse()

	options := conflictwriter.InOptions{
		Source: conflictwriter.Bucket{
			Url:      *source,
			Name:     *srcBucket,
			Username: *srcUsername,
			Password: "xxx",
		},
		Target: conflictwriter.Bucket{
			Url:      *target,
			Name:     *tgtBucket,
			Username: *tgtUsername,
			Password: "xxx",
		},
		Distribution: *distribution,
		BatchSize:    *batchSize,
		Gap:          *gap,
		Workers:      *workers,
		NumConflicts: *numConflicts,
		DebugMode:    utils.DebugModeType(*debugMode),
	}

	fmt.Printf("Options: %+v\n", options)

	options.Source.Password = *srcPassword
	options.Target.Password = *tgtPassword

	options.Validate()

	return options
}

func monitorForInterrupt(finCh chan bool) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	for sig := range c {
		if sig.String() == "interrupt" {
			fmt.Println("Received interrupt. Closing all workers")
			close(finCh)
		}
	}
}
