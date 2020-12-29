package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/containerssh/geoip"
	"github.com/containerssh/log"
	"github.com/containerssh/metrics"
	"github.com/containerssh/sshserver"
	"github.com/containerssh/structutils"
	"github.com/creasty/defaults"
	"github.com/stretchr/testify/assert"
	v1Api "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func must(t *testing.T, arg bool) {
	if !arg {
		t.FailNow()
	}
}

func getKube(t *testing.T, config Config) (sshserver.NetworkConnectionHandler, string) {
	connectionID := sshserver.GenerateConnectionID()
	geoipProvider, err := geoip.New(geoip.Config{
		Provider: geoip.DummyProvider,
	})
	must(t, assert.NoError(t, err))
	collector := metrics.New(geoipProvider)
	logger := getLogger(t)
	kr, err := New(
		net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 2222,
			Zone: "",
		}, connectionID, config, logger,
		collector.MustCreateCounter("backend_requests", "", ""),
		collector.MustCreateCounter("backend_failures", "", ""),
	)
	must(t, assert.NoError(t, err, "failed to create handler (%v)", err))
	return kr, connectionID
}

func TestSuccessfulHandshakeShouldCreatePod(t *testing.T) {
	t.Parallel()

	modes := []ExecutionMode{ExecutionModeSession, ExecutionModeConnection}

	for _, mode := range modes {
		t.Run(
			fmt.Sprintf("mode=%s", mode), func(t *testing.T) {
				t.Parallel()
				config := Config{}
				structutils.Defaults(&config)

				if err := setConfigFromKubeConfig(&config); err != nil {
					assert.FailNow(t, "failed to create configuration from the current users kubeconfig (%v)", err)
				}

				kr, connectionID := getKube(t, config)
				defer kr.OnDisconnect()

				_, err := kr.OnHandshakeSuccess("test")
				assert.Nil(t, err, "failed to create handshake handler (%v)", err)

				k8sConfig := createConnectionConfig(config)
				cli, err := kubernetes.NewForConfig(&k8sConfig)
				assert.Nil(t, err, "failed to create k8s client (%v)", err)

				podList, err := cli.CoreV1().Pods(config.Pod.Metadata.Namespace).List(
					context.Background(), v1.ListOptions{
						LabelSelector: fmt.Sprintf("%s=%s", "containerssh_connection_id", connectionID),
					},
				)
				assert.Nil(t, err, "failed to list k8s pods (%v)", err)
				assert.Equal(t, 1, len(podList.Items))
				assert.Equal(t, v1Api.PodRunning, podList.Items[0].Status.Phase)
				assert.Equal(t, true, *podList.Items[0].Status.ContainerStatuses[0].Started)
			})
	}
}

func TestSingleSessionShouldRunProgram(t *testing.T) {
	t.Parallel()

	modes := []ExecutionMode{ExecutionModeSession, ExecutionModeConnection}

	for _, mode := range modes {
		t.Run(
			fmt.Sprintf("mode=%s", mode), func(t *testing.T) {
				t.Parallel()

				config := getKubernetesConfig(t)

				config.Pod.Mode = mode

				kr, _ := getKube(t, config)
				defer kr.OnDisconnect()

				ssh, err := kr.OnHandshakeSuccess("test")
				assert.Nil(t, err, "failed to create handshake handler (%v)", err)

				channel, err := ssh.OnSessionChannel(0, []byte{})
				assert.Nil(t, err, "failed to to create session channel (%v)", err)

				stdin := bytes.NewReader([]byte{})
				stdout := &bytes.Buffer{}
				stderr := &bytes.Buffer{}
				done := make(chan struct{})
				status := 0
				err = channel.OnExecRequest(
					0,
					"echo \"Hello world!\"",
					stdin,
					stdout,
					stderr,
					func(exitStatus sshserver.ExitStatus) {
						status = int(exitStatus)
						done <- struct{}{}
					},
				)
				assert.Nil(t, err)
				<-done
				stdoutBytes := stdout.Bytes()
				stderrBytes := stderr.Bytes()
				assert.Equal(t, "Hello world!\n", string(stdoutBytes))
				assert.Equal(t, "", string(stderrBytes))
				assert.Equal(t, 0, status)
			},
		)
	}
}

func getLogger(t *testing.T) log.Logger {
	logger, err := log.New(
		log.Config{
			Level:  log.LevelDebug,
			Format: log.FormatText,
		},
		"kubernetes",
		os.Stdout,
	)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	return logger
}

func getKubernetesConfig(t *testing.T) Config {
	config := Config{}
	err := defaults.Set(&config)
	if !assert.Nil(t, err, "failed to set defaults (%v)", err) {
		t.FailNow()
	}

	err = setConfigFromKubeConfig(&config)
	if !assert.Nil(t, err, "failed to set up kube config (%v)", err) {
		t.FailNow()
	}

	return config
}

func TestCommandExecutionShouldReturnStatusCode(t *testing.T) {
	t.Parallel()

	modes := []ExecutionMode{ExecutionModeSession, ExecutionModeConnection}

	for _, mode := range modes {
		t.Run(fmt.Sprintf("mode=%s", mode), func(t *testing.T) {
			t.Parallel()

			config := getKubernetesConfig(t)

			config.Pod.Mode = mode

			kr, _ := getKube(t, config)
			defer kr.OnDisconnect()

			ssh, err := kr.OnHandshakeSuccess("test")
			assert.Nil(t, err, "failed to create handshake handler (%v)", err)

			channel, err := ssh.OnSessionChannel(0, []byte{})
			assert.Nil(t, err, "failed to to create session channel (%v)", err)

			stdin := bytes.NewReader([]byte{})
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			done := make(chan struct{})
			status := 0
			err = channel.OnExecRequest(
				0,
				"exit 42",
				stdin,
				stdout,
				stderr,
				func(exitStatus sshserver.ExitStatus) {
					status = int(exitStatus)
					done <- struct{}{}
				},
			)
			assert.Nil(t, err)
			<-done
			assert.Equal(t, 42, status)
		})
	}
}
