package conflictwriter

import (
	"conflictSim/distribution"
	goSDK "conflictSim/gocbcore"
	"conflictSim/utils"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/couchbase/gocbcore/v10"
	xdcrBase "github.com/couchbase/goxdcr/v8/base"
	xdcrLog "github.com/couchbase/goxdcr/v8/log"
	xdcrUtils "github.com/couchbase/goxdcr/v8/utils"
)

func init() {
	mp := make(map[string]interface{})
	jsonLen := 100
	strLen := 10
	str := ""
	for range strLen {
		str += "a"
	}
	for i := range jsonLen {
		mp[fmt.Sprintf("%s:%v", str, i)] = str
	}
	Body, _ = json.Marshal(mp)
}

var Body []byte

type InOptions struct {
	Source       Bucket
	Target       Bucket
	Distribution string
	BatchSize    int
	Gap          int
	Workers      int
	NumConflicts int
	DebugMode    utils.DebugModeType
}

func (o InOptions) Validate() {
	o.Source.Validate(true)
	o.Target.Validate(false)

	if o.NumConflicts < 0 {
		panic("numConflicts is compulsory and must be positive")
	}
}

type Bucket struct {
	Name     string
	Username string
	Password string
	Url      string // ns_server url => input
	ConnStr  string // kv url => derived
}

func (b Bucket) Validate(source bool) {
	if len(b.Name) == 0 {
		panic(fmt.Sprintf("bucket name is compulsory, source=%v", source))
	}

	if len(b.Username) == 0 {
		panic(fmt.Sprintf("bucket username is compulsory, source=%v", source))
	}

	if len(b.Password) == 0 {
		panic(fmt.Sprintf("bucket password is compulsory, source=%v", source))
	}

	if len(b.Url) == 0 {
		panic(fmt.Sprintf("bucket url is compulsory, source=%v", source))
	}
}

type conflictWriter struct {
	opts              InOptions
	unitWorkPerWorker int
	distribution      distribution.Distribution
	work              []chan bool
	finCh             chan bool
	finAckCh          []chan bool
	logger            *xdcrLog.CommonLogger
	utils             xdcrUtils.UtilsIface
	failures          uint64
	success           uint64
	remaining         uint64
}

func NewConflictWriter(logger *xdcrLog.CommonLogger, opts InOptions, finCh chan bool) conflictWriter {
	return conflictWriter{
		opts:      opts,
		finCh:     finCh,
		work:      make([]chan bool, opts.Workers),
		finAckCh:  make([]chan bool, opts.Workers),
		logger:    logger,
		utils:     xdcrUtils.NewUtilities(),
		remaining: uint64(opts.NumConflicts),
		distribution: distribution.NewDistribution(
			float64(opts.Gap),
			distribution.RandomTypeFromStr(opts.Distribution),
		),
	}
}

func (cw *conflictWriter) worker(wid int, sourceWriter, targetWriter *gocbcore.Agent) {
	cw.logger.Infof("worker %v starting", wid)

	for {
		select {
		case <-cw.finCh:
			sourceWriter.Close()
			targetWriter.Close()
			cw.logger.Infof("worker %v exiting", wid)
			cw.finAckCh[wid] <- true
			return
		case <-cw.work[wid]:
			for i := range cw.unitWorkPerWorker {
				key := fmt.Sprintf("key-%v-%v", time.Now().UnixNano(), rand.Intn(1000000))
				var wg sync.WaitGroup
				var err1, err2 error

				wg.Add(1)
				go func() {
					defer wg.Done()
					ch := make(chan error)

					sourceWriter.Set(gocbcore.SetOptions{
						Key:            []byte(key),
						CollectionName: "_default",
						ScopeName:      "_default",
						Datatype:       uint8(gocbcore.JSONType),
						Value:          Body,
					}, func(sr *gocbcore.StoreResult, err error) {
						ch <- err
					})

					err1 = <-ch
					if err1 != nil {
						cw.logger.Debugf("Error writing to source bucket, err=%v", err1)
					}
				}()

				wg.Add(1)
				go func() {
					defer wg.Done()
					ch := make(chan error)

					targetWriter.Set(gocbcore.SetOptions{
						Key:            []byte(key),
						CollectionName: "_default",
						ScopeName:      "_default",
						Datatype:       uint8(gocbcore.JSONType),
						Value:          Body,
					}, func(sr *gocbcore.StoreResult, err error) {
						ch <- err
					})

					err2 = <-ch
					if err2 != nil {
						cw.logger.Debugf("Error writing to target bucket, err=%v", err2)
					}
				}()

				wg.Wait()
				cw.logger.Debugf("%v worker wrote a conflict in round %v of %v", wid, i+1, cw.unitWorkPerWorker)
				if err1 != nil || err2 != nil {
					atomic.AddUint64(&cw.failures, 1)
				} else {
					atomic.AddUint64(&cw.success, 1)
				}
			}
		}
	}
}

func (cw *conflictWriter) status() {
	t := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-t.C:
			cw.logger.Infof("success = %v, failures = %v, remaining = %v, eta = %v mins",
				atomic.LoadUint64(&cw.success),
				atomic.LoadUint64(&cw.failures),
				cw.remaining,
				float64(cw.remaining*uint64(cw.opts.Gap))/float64(uint64(cw.opts.BatchSize)*1000*60),
			)
		case <-cw.finCh:
			t.Stop()
			cw.logger.Infof("status() exiting")
			return
		}
	}
}

func (cw *conflictWriter) generate() {
	remainingConflicts := cw.opts.NumConflicts
	for remainingConflicts > 0 {
		atomic.StoreUint64(&cw.remaining, uint64(remainingConflicts))

		cw.logger.Debugf("Writing %v conflicts", cw.opts.BatchSize)
		for wid := range cw.opts.Workers {
			select {
			case <-cw.finCh:
				cw.logger.Infof("generate exits with remaining=%v", remainingConflicts)
				return
			case cw.work[wid] <- true:
			}
		}
		remainingConflicts -= cw.opts.BatchSize
		cw.logger.Debugf("Done writing %v more conflicts, remaining=%v", cw.opts.BatchSize, remainingConflicts)

		sleepDuration := cw.distribution.Next()
		cw.logger.Debugf("Sleeping %v ms", sleepDuration)
		time.Sleep(time.Duration(sleepDuration) * time.Millisecond)
	}

	close(cw.finCh)
	for wid := range cw.opts.Workers {
		<-cw.finAckCh[wid]
	}
	cw.logger.Infof("Done writing all conflicts")
}

func (cw *conflictWriter) Init() error {
	sourceKvVbMap, err := cw.initializeKVVBMap(cw.opts.Source.Url, cw.opts.Source.Username, cw.opts.Source.Password, cw.opts.Source.Name)
	if err != nil {
		cw.logger.Errorf("error initing sourceKvVbMap, err=%v", err)
		return fmt.Errorf("source initializeKVVBMap error")
	}

	targetKvVbMap, err := cw.initializeKVVBMap(cw.opts.Target.Url, cw.opts.Target.Username, cw.opts.Target.Password, cw.opts.Target.Name)
	if err != nil {
		cw.logger.Errorf("error initing targetKvVbMap, err=%v", err)
		return fmt.Errorf("target initializeKVVBMap error")
	}

	cw.opts.Source.ConnStr = utils.GetBucketConnStr(sourceKvVbMap, cw.opts.Source.Url, cw.logger)
	cw.logger.Infof("Using source bucketConnStr=%v", cw.opts.Source.ConnStr)

	cw.opts.Target.ConnStr = utils.GetBucketConnStr(targetKvVbMap, cw.opts.Target.Url, cw.logger)
	cw.logger.Infof("Using target bucketConnStr=%v", cw.opts.Target.ConnStr)

	cw.unitWorkPerWorker = cw.opts.BatchSize / cw.opts.Workers
	cw.logger.Infof("unitWorkPerWorker=%v", cw.unitWorkPerWorker)

	return nil
}

func (cw *conflictWriter) Start() {
	for wid := range cw.opts.Workers {
		work := make(chan bool, 10)
		finAck := make(chan bool)
		cw.work[wid] = work
		cw.finAckCh[wid] = finAck
		sourceWriter := goSDK.NewAgent(cw.opts.Source.Name, cw.opts.Source.ConnStr, cw.opts.Source.Username, cw.opts.Source.Password, cw.logger)
		targetWriter := goSDK.NewAgent(cw.opts.Target.Name, cw.opts.Target.ConnStr, cw.opts.Target.Username, cw.opts.Target.Password, cw.logger)
		go cw.worker(wid, sourceWriter, targetWriter)
	}

	go cw.status()
	cw.generate()
}

func (cw *conflictWriter) initializeKVVBMap(url, username, password, bucketName string) (map[string][]uint16, error) {
	var kvVbMap map[string][]uint16

	_, _, _, _, _, kvVbMap, err := cw.utils.BucketValidationInfo(url, bucketName, username,
		password, xdcrBase.HttpAuthMechPlain, []byte{}, false, []byte{}, []byte{}, cw.logger)
	if err != nil {
		cw.logger.Errorf("initializeKVVBMap error=%v", err)
		return nil, err
	}

	cw.logger.Infof("KvVbMap fetched for %v, %v: %v", url, bucketName, kvVbMap)
	return kvVbMap, nil
}
