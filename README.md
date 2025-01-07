# conflictSim

A docloader to generate and simulate conflicting writes between two given couchbase buckets. It can used to generate conflicts in an interleaving fashion to simulate many real world applications. It is used to test the performance of XDCR active-active conflict logging feature.

## Installation on macOS
Prerequisites: Golang

```
$ make clean # for cleaning previous installations, if any
$ make deps  # to install the golang dependencies
$ make
```

## Usage
Either install the binary from source as mentioned above for macOS or download the binary from the latest release and give the binary the executable permissions for linux/amd64 systems.
```
$ ./conflictSim -h
Usage of ./conflictSim:
  -batchSize int
    	Batch size of conflict writes i.e. number of conflicts to be simulated for a single conflict generation (optional, default=1) (default 1)
  -debugMode int
    	Turn on xdcr debug logging and SDK verbose logging. 0 is for off. 1 is for xdcr debug logging. 2 is for xdcr debug logging + SDK versbose logging (optional, default=0 or off)
  -distribution string
    	Distribution of conflict writes: step, normrandom, exprandom, pseudorandom  (optional, default=step) (default "step")
  -gap int
    	Gap in milliseconds between conflict batch writes (optional, default=2000) (default 2000)
  -numConflicts int
    	Number of conflicts to be generated (compulsory) (default -1)
  -source string
    	Source host address (compulsory)
  -srcBucket string
    	Source bucket name (compulsory)
  -srcPassword string
    	Source password (compulsory)
  -srcUsername string
    	Source username (compulsory)
  -target string
    	Target host address (compulsory)
  -tgtBucket string
    	Target bucket name (compulsory)
  -tgtPassword string
    	Target password (compulsory)
  -tgtUsername string
    	Target username (compulsory)
  -workers int
    	Number of workers per cluster. Each worker simulates an application connecting to a cluster (optional, default=10) (default 10)
```

## Example
```
$ ./conflictSim -source 172.100.23.100:8091 -target 172.100.23.105:8091 -srcBucket bucket-1 -tgtBucket bucket-2 -srcUsername Administrator -tgtUsername Administrator -srcPassword wewewe -tgtPassword wewewe -numConflicts 1000000 -batchSize 1000 -gap 3000 -workers 500
```
The above command means that a total of `1000000` conflicts will be generated between `bucket-1` of couchbase cluster `172.100.23.100:8091` and `bucket-2` of couchbase cluster `172.100.23.101:8091`. The conflicts will be generated in such a way that `1000` conflicts will be generated every `3000` milliseconds or 3 seconds by `500` application threads.
