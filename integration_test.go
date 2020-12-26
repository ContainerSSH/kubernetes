package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/containerssh/log"
	"github.com/containerssh/sshserver"
	"github.com/containerssh/structutils"
	"github.com/creasty/defaults"
	"github.com/stretchr/testify/assert"
	v1Api "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

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

				connectionID := sshserver.GenerateConnectionID()

				logger, err := log.New(
					log.Config{
						Level:  log.LevelDebug,
						Format: log.FormatText,
					},
					"kubernetes",
					os.Stdout,
				)
				assert.NoError(t, err)

				kr, err := New(
					config, connectionID, net.TCPAddr{
						IP:   net.ParseIP("127.0.0.1"),
						Port: 2222,
						Zone: "",
					}, logger,
				)
				assert.Nil(t, err, "failed to create handler (%v)", err)
				defer kr.OnDisconnect()

				_, err = kr.OnHandshakeSuccess("test")
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

				config := Config{}
				err := defaults.Set(&config)
				assert.Nil(t, err, "failed to set defaults (%v)", err)

				err = setConfigFromKubeConfig(&config)
				assert.Nil(t, err, "failed to set up kube config (%v)", err)

				config.Pod.Mode = mode

				connectionID := sshserver.GenerateConnectionID()
				logger, err := log.New(
					log.Config{
						Level:  log.LevelDebug,
						Format: log.FormatText,
					},
					"kubernetes",
					os.Stdout,
				)
				assert.NoError(t, err)

				kr, err := New(
					config, connectionID, net.TCPAddr{
						IP:   net.ParseIP("127.0.0.1"),
						Port: 2222,
						Zone: "",
					}, logger,
				)
				assert.Nil(t, err, "failed to create handler (%v)", err)
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
			})
	}
}

func TestCommandExecutionShouldReturnStatusCode(t *testing.T) {
	t.Parallel()

	modes := []ExecutionMode{ExecutionModeSession, ExecutionModeConnection}

	for _, mode := range modes {
		t.Run(fmt.Sprintf("mode=%s", mode), func(t *testing.T) {
			t.Parallel()

			config := Config{}
			structutils.Defaults(&config)

			err := setConfigFromKubeConfig(&config)
			assert.Nil(t, err, "failed to set up kube config (%v)", err)

			config.Pod.Mode = mode

			connectionID := sshserver.GenerateConnectionID()
			logger, err := log.New(
				log.Config{
					Level:  log.LevelDebug,
					Format: log.FormatText,
				},
				"kubernetes",
				os.Stdout,
			)
			assert.NoError(t, err)
			kr, err := New(
				config, connectionID, net.TCPAddr{
					IP:   net.ParseIP("127.0.0.1"),
					Port: 2222,
					Zone: "",
				}, logger,
			)
			assert.Nil(t, err, "failed to create handler (%v)", err)
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
