package kubernetes

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/containerssh/log"
	"github.com/containerssh/sshserver"
	"github.com/creasty/defaults"
	"github.com/stretchr/testify/assert"
	v1Api "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func TestSuccessfulHandshakeShouldCreatePod(t *testing.T) {
	config := Config{}
	err := defaults.Set(&config)
	assert.Nil(t, err, "failed to set defaults (%v)", err)

	config.Pod.Spec.Containers[0].Image = "docker.io/library/busybox"

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

	kr, err := New(config, connectionID, net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 2222,
		Zone: "",
	}, logger)
	assert.Nil(t, err, "failed to create handler (%v)", err)
	defer kr.OnDisconnect()

	_, err = kr.OnHandshakeSuccess("test")
	assert.Nil(t, err, "failed to create handshake handler (%v)", err)

	k8sConfig := createConnectionConfig(config)
	cli, err := kubernetes.NewForConfig(&k8sConfig)
	assert.Nil(t, err, "failed to create k8s client (%v)", err)

	podList, err := cli.CoreV1().Pods(config.Pod.Metadata.Namespace).List(context.Background(), v1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", "containerssh_connection_id", connectionID),
	})
	assert.Nil(t, err, "failed to list k8s pods (%v)", err)
	assert.Equal(t, 1, len(podList.Items))
	assert.Equal(t, v1Api.PodRunning, podList.Items[0].Status.Phase)
	assert.Equal(t, true, *podList.Items[0].Status.ContainerStatuses[0].Started)
}

func TestSingleSessionShouldRunProgram(t *testing.T) {
	config := Config{}
	err := defaults.Set(&config)
	assert.Nil(t, err, "failed to set defaults (%v)", err)

	config.Pod.Spec.Containers[0].Image = "docker.io/library/busybox"

	err = setConfigFromKubeConfig(&config)
	assert.Nil(t, err, "failed to set up kube config (%v)", err)

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

	kr, err := New(config, connectionID, net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 2222,
		Zone: "",
	}, logger)
	assert.Nil(t, err, "failed to create handler (%v)", err)
	defer kr.OnDisconnect()

	ssh, err := kr.OnHandshakeSuccess("test")
	assert.Nil(t, err, "failed to create handshake handler (%v)", err)

	channel, err := ssh.OnSessionChannel(0, []byte{})
	assert.Nil(t, err, "failed to to create session channel (%v)", err)

	stdin := bytes.NewReader([]byte{})
	var stdoutBytes bytes.Buffer
	stdout := bufio.NewWriter(&stdoutBytes)
	var stderrBytes bytes.Buffer
	stderr := bufio.NewWriter(&stderrBytes)
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
	assert.Nil(t, stdout.Flush())
	assert.Equal(t, "Hello world!\n", stdoutBytes.String())
	assert.Equal(t, "", stderrBytes.String())
	assert.Equal(t, 0, status)
}

func TestCommandExecutionShouldReturnStatusCode(t *testing.T) {
	config := Config{}
	err := defaults.Set(&config)
	assert.Nil(t, err, "failed to set defaults (%v)", err)

	config.Pod.Spec.Containers[0].Image = "docker.io/library/busybox"

	err = setConfigFromKubeConfig(&config)
	assert.Nil(t, err, "failed to set up kube config (%v)", err)

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
	kr, err := New(config, connectionID, net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 2222,
		Zone: "",
	}, logger)
	assert.Nil(t, err, "failed to create handler (%v)", err)
	defer kr.OnDisconnect()

	ssh, err := kr.OnHandshakeSuccess("test")
	assert.Nil(t, err, "failed to create handshake handler (%v)", err)

	channel, err := ssh.OnSessionChannel(0, []byte{})
	assert.Nil(t, err, "failed to to create session channel (%v)", err)

	stdin := bytes.NewReader([]byte{})
	var stdoutBytes bytes.Buffer
	stdout := bufio.NewWriter(&stdoutBytes)
	var stderrBytes bytes.Buffer
	stderr := bufio.NewWriter(&stderrBytes)
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
	assert.Nil(t, stdout.Flush())
	assert.Equal(t, 42, status)
}
