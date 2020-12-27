package kubernetes_test

import (
	"io/ioutil"
	"net"
	"os"
	"testing"

	"github.com/containerssh/log"
	"github.com/containerssh/sshserver"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"

	"github.com/containerssh/kubernetes"
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
	config := kubernetes.KubeRunConfig{}
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

	//goland:noinspection GoDeprecation
	_, err = kubernetes.NewKubeRun(
		config,
		sshserver.GenerateConnectionID(),
		net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 2222,
		},
		logger,
	)
	if !assert.NoError(t, err) {
		return
	}
}
