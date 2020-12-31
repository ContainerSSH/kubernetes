package kubernetes_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/containerssh/geoip"
	"github.com/containerssh/log"
	"github.com/containerssh/metrics"
	"github.com/containerssh/service"
	"github.com/containerssh/sshserver"
	"github.com/containerssh/structutils"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ssh"

	"github.com/containerssh/kubernetes"
)

func TestFullSSHServer(t *testing.T) {
	t.Parallel()

	lock := &sync.Mutex{}
	for _, mode := range []kubernetes.ExecutionMode{
		kubernetes.ExecutionModeConnection,
		kubernetes.ExecutionModeSession,
	} {
		t.Run(fmt.Sprintf("mode=%s", mode), func(t *testing.T) {
			lock.Lock()
			defer lock.Unlock()

			lifecycle, listen, err := createSSHServer(mode)
			if !assert.NoError(t, err) {
				return
			}
			defer lifecycle.Stop(context.Background())

			running := make(chan struct{})
			lifecycle.OnRunning(
				func(s service.Service, l service.Lifecycle) {
					running <- struct{}{}
				},
			)
			go func() {
				_ = lifecycle.Run()
			}()
			<-running

			clientConfig := ssh.ClientConfig{
				User:            "test",
				Auth:            []ssh.AuthMethod{ssh.Password("")},
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			}
			sshConnection, err := ssh.Dial("tcp", listen, &clientConfig)
			if !assert.NoError(t, err) {
				return
			}
			defer func() {
				_ = sshConnection.Close()
			}()
			testCommandExecution(t, sshConnection)
			testShell(t, sshConnection)
		})
	}
}

func testShell(t *testing.T, sshConnection *ssh.Client) {
	session, err := sshConnection.NewSession()
	if !assert.NoError(t, err) {
		return
	}
	defer func() {
		_ = session.Close()
	}()

	stdin, err := session.StdinPipe()
	if !assert.NoError(t, err) {
		return
	}
	stdout, err := session.StdoutPipe()
	if !assert.NoError(t, err) {
		return
	}
	_, err = session.StderrPipe()
	if !assert.NoError(t, err) {
		return
	}
	assert.NoError(t, session.Setenv("MESSAGE", "Hello world!"))
	assert.NoError(t, session.RequestPty("xterm", 30, 120, ssh.TerminalModes{}))
	assert.NoError(t, session.Shell())

	output := bytes.Buffer{}

	// HACK: Sleep 1 second before querying cols and rows because Kubernetes processes terminal resizes asynchronously.
	_, err = stdin.Write([]byte("sleep 1 && echo \"$MESSAGE\" && echo \"COLS:$(tput cols)\" && echo \"ROWS:$(tput lines)\" && exit\n"))
	assert.NoError(t, err)
	for {
		buf := make([]byte, 1024)
		n, err := stdout.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			assert.NoError(t, err)
		}
		output.Write(buf[:n])
	}
	_ = stdin.Close()
	outputString := output.String()
	assert.True(t, strings.Contains(outputString, "Hello world!\r\n"))
	assert.True(t, strings.Contains(outputString, "COLS:120\r\n"))
	assert.True(t, strings.Contains(outputString, "ROWS:30\r\n"))
	assert.NoError(t, session.Wait())
}

func testCommandExecution(t *testing.T, sshConnection *ssh.Client) {
	session, err := sshConnection.NewSession()
	if !assert.NoError(t, err) {
		return
	}
	output, err := session.CombinedOutput("echo \"Hello world!\"")
	assert.NoError(t, err)
	assert.Equal(t, []byte("Hello world!\n"), output)
	_ = session.Close()
}

func createSSHServer(mode kubernetes.ExecutionMode) (service.Lifecycle, string, error) {
	logger, err := log.New(
		log.Config{
			Level:  log.LevelDebug,
			Format: log.FormatText,
		},
		"ssh",
		os.Stdout,
	)
	if err != nil {
		return nil, "", err
	}
	config := sshserver.Config{}
	structutils.Defaults(&config)
	if err := config.GenerateHostKey(); err != nil {
		return nil, "", err
	}
	geo, err := geoip.New(
		geoip.Config{
			Provider: geoip.DummyProvider,
		},
	)
	if err != nil {
		return nil, "", err
	}
	metricsCollector := metrics.New(geo)
	srv, err := sshserver.New(
		config,
		&fullHandler{
			mode,
			logger,
			metricsCollector.MustCreateCounter("backend_requests", "", ""),
			metricsCollector.MustCreateCounter("backend_errors", "", ""),
		},
		logger,
	)
	if err != nil {
		return nil, "", err
	}
	lifecycle := service.NewLifecycle(srv)
	listen := config.Listen
	return lifecycle, listen, err
}

type fullHandler struct {
	mode           kubernetes.ExecutionMode
	logger         log.Logger
	requestsMetric metrics.SimpleCounter
	errorsMetric   metrics.SimpleCounter
}

func (f *fullHandler) OnReady() error {
	return nil
}

func (f *fullHandler) OnShutdown(_ context.Context) {}

func (f *fullHandler) OnNetworkConnection(client net.TCPAddr, connectionID string) (
	sshserver.NetworkConnectionHandler,
	error,
) {
	config := kubernetes.Config{}
	structutils.Defaults(&config)
	if err := config.SetConfigFromKubeConfig(); err != nil {
		return nil, err
	}
	config.Pod.Mode = f.mode

	backend, err := kubernetes.New(
		client,
		connectionID,
		config,
		f.logger,
		f.requestsMetric,
		f.errorsMetric,
	)
	if err != nil {
		return nil, err
	}
	return &nullAuthenticator{
		backend: backend,
	}, nil
}

type nullAuthenticator struct {
	backend sshserver.NetworkConnectionHandler
}

func (n *nullAuthenticator) OnAuthPassword(_ string, _ []byte) (
	response sshserver.AuthResponse,
	reason error,
) {
	return sshserver.AuthResponseSuccess, nil
}

func (n *nullAuthenticator) OnAuthPubKey(_ string, _ string) (
	response sshserver.AuthResponse,
	reason error,
) {
	return sshserver.AuthResponseSuccess, nil
}

func (n *nullAuthenticator) OnHandshakeFailed(_ error) {

}

func (n *nullAuthenticator) OnHandshakeSuccess(username string) (
	connection sshserver.SSHConnectionHandler,
	failureReason error,
) {
	return n.backend.OnHandshakeSuccess(username)
}

func (n *nullAuthenticator) OnDisconnect() {
	n.backend.OnDisconnect()
}
