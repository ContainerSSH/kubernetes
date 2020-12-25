package kubernetes

import (
	"net"

	"github.com/containerssh/log"
	"github.com/containerssh/sshserver"
)

// NewKubeRun creates a handler based on the legacy "kuberun" configuration.
// Deprecated: Use New instead.
//goland:noinspection GoDeprecation
func NewKubeRun(oldConfig KubeRunConfig, connectionID string, client net.TCPAddr, logger log.Logger) (sshserver.NetworkConnectionHandler, error) {
	config := Config{}
	config.Pod = PodConfig{
		Namespace:              oldConfig.Pod.Namespace,
		ConsoleContainerNumber: oldConfig.Pod.ConsoleContainerNumber,
		Spec:                   oldConfig.Pod.Spec,
		Subsystems:             oldConfig.Pod.Subsystems,
		IdleCommand:            nil,
		ShellCommand:           nil,
		Mode:                   ExecutionModeSession,
		disableCommand:         oldConfig.Pod.DisableCommand,
	}
	config.Connection = oldConfig.Connection.ConnectionConfig
	config.Timeouts = TimeoutConfig{
		HTTP:     oldConfig.Connection.Timeout,
		PodStart: oldConfig.Timeout,
		PodStop:  oldConfig.Timeout,
	}

	return New(
		config,
		connectionID,
		client,
		logger,
	)
}
