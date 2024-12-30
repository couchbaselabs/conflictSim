package utils

import (
	"fmt"

	"github.com/couchbase/gocbcore/v10"
	xdcrBase "github.com/couchbase/goxdcr/v8/base"
	xdcrLog "github.com/couchbase/goxdcr/v8/log"
)

func GetBucketConnStr(kvVbMap map[string][]uint16, url string, logger *xdcrLog.CommonLogger) string {
	kvHostAddr := ""
	for connStr := range kvVbMap {
		kvHostAddr = connStr
		break
	}
	hostname := xdcrBase.GetHostName(url)
	var kvPort uint16
	var err error
	kvPort, err = xdcrBase.GetPortNumber(kvHostAddr)
	if err != nil {
		logger.Warnf("Error getting kv port. Will use 11210. kvVbMap=%v, url=%v", kvVbMap, url)
		kvPort = 11210
	}

	return fmt.Sprintf("%v%v:%v", CouchbasePrefix, hostname, kvPort)
}

const CouchbasePrefix = "couchbase://"

type DebugModeType int

const (
	NoDebugMode DebugModeType = iota
	XdcrDebugMode
	XdcrAndSDKDebugMode
)

func ProcessDebugMode(n DebugModeType) *xdcrLog.LoggerContext {
	logCtx := xdcrLog.DefaultLoggerContext
	switch n {
	case XdcrAndSDKDebugMode:
		gocbcore.SetLogger(gocbcore.VerboseStdioLogger())
		fallthrough
	case XdcrDebugMode:
		logCtx.Log_level = xdcrLog.LogLevelDebug
	}

	return logCtx
}
