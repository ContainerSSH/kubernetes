package kubernetes

import (
	"net"

	"github.com/containerssh/log"
	"github.com/containerssh/sshserver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewKubeRun creates a handler based on the legacy "kuberun" configuration.
// Deprecated: Use New instead.
//goland:noinspection GoDeprecation,GoUnusedExportedFunction
func NewKubeRun(oldConfig KubeRunConfig, connectionID string, client net.TCPAddr, logger log.Logger) (sshserver.NetworkConnectionHandler, error) {
	logger.Warningf(
		"You are using the kuberun backend deprecated since ContainerSSH 0.4. This backend will be removed " +
			"in the future. Please switch to the new kubernetes backend as soon as possible. " +
			"See https://containerssh.io/deprecations/kuberun for details.",
	)

	config := Config{}
	config.Pod = PodConfig{
		Metadata: metav1.ObjectMeta{
			Namespace: oldConfig.Pod.Namespace,
		},
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
