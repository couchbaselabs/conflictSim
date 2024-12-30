package goSDK

import (
	"crypto/x509"
	"fmt"
	"time"

	"github.com/couchbase/gocbcore/v10"
	xdcrLog "github.com/couchbase/goxdcr/v8/log"
)

func NewAgent(bucketName, bucketConnStr, username, password string, logger *xdcrLog.CommonLogger) *gocbcore.Agent {
	return createSDKAgent(&gocbcore.AgentConfig{
		SeedConfig: gocbcore.SeedConfig{MemdAddrs: []string{bucketConnStr}},
		BucketName: bucketName,
		UserAgent:  fmt.Sprintf("conflictSim_%v/%v", bucketConnStr, bucketName),
		SecurityConfig: gocbcore.SecurityConfig{
			UseTLS: false,
			TLSRootCAProvider: func() *x509.CertPool {
				return nil
			},
			Auth: gocbcore.PasswordAuthProvider{
				Username: username,
				Password: password,
			},
			AuthMechanisms: []gocbcore.AuthMechanism{gocbcore.ScramSha1AuthMechanism,
				gocbcore.ScramSha256AuthMechanism,
				gocbcore.ScramSha512AuthMechanism,
			},
		},
		KVConfig: gocbcore.KVConfig{
			ConnectTimeout: 10 * time.Second,
			MaxQueueSize:   100000 * 50,
		},
		CompressionConfig: gocbcore.CompressionConfig{Enabled: true},
		HTTPConfig:        gocbcore.HTTPConfig{ConnectTimeout: 10 * time.Second},
		IoConfig:          gocbcore.IoConfig{UseCollections: true},
	}, logger)
}

func createSDKAgent(agentConfig *gocbcore.AgentConfig, logger *xdcrLog.CommonLogger) *gocbcore.Agent {
	agent, err := gocbcore.CreateAgent(agentConfig)
	if err != nil {
		logger.Errorf("CreateAgent err=%v", err)
	}

	signal := make(chan error)
	_, err = agent.WaitUntilReady(time.Now().Add(15*time.Second),
		gocbcore.WaitUntilReadyOptions{
			DesiredState: gocbcore.ClusterStateOnline,
			ServiceTypes: []gocbcore.ServiceType{gocbcore.MemdService},
		}, func(res *gocbcore.WaitUntilReadyResult, err error) {
			signal <- err
		})

	if err == nil {
		err = <-signal
		if err != nil {
			logger.Errorf("Waited 15 seconds for bucket to be ready, err=%v\n", err)
			return nil
		}
	}

	return agent
}
