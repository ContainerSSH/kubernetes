package kubernetes

import (
	"io/ioutil"
	"net"
	"os"
	"testing"

	"github.com/containerssh/geoip"
	"github.com/containerssh/log"
	"github.com/containerssh/metrics"
	"github.com/containerssh/sshserver"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestV03Compatibility(t *testing.T) {
	t.Parallel()

	fh, err := os.Open("testdata/config-0.3.yaml")
	if !assert.NoError(t, err) {
		return
	}
	defer func() {
		assert.NoError(t, fh.Close())
	}()

	data, err := ioutil.ReadAll(fh)
	if !assert.NoError(t, err) {
		return
	}

	//goland:noinspection GoDeprecation
	config := KubeRunConfig{}
	if !assert.NoError(t, yaml.Unmarshal(data, &config)) {
		return
	}

	logger, err := log.New(
		log.Config{
			Level:  log.LevelDebug,
			Format: log.FormatText,
		},
		"kuberun",
		os.Stdout,
	)
	if !assert.NoError(t, err) {
		return
	}
	logger.Noticef("The deprecation notice in this test is intentional.")

	geoipProvider, err := geoip.New(geoip.Config{
		Provider: geoip.DummyProvider,
	})
	must(t, assert.NoError(t, err))
	collector := metrics.New(geoipProvider)
	//goland:noinspection GoDeprecation
	_, err = NewKubeRun(
		net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 2222,
		},
		sshserver.GenerateConnectionID(),
		config,
		logger,
		collector.MustCreateCounter("backend_requests", "", ""),
		collector.MustCreateCounter("backend_errors", "", ""),
	)
	if !assert.NoError(t, err) {
		return
	}
}
