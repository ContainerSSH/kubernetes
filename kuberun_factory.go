package kubernetes

import (
	"net"

	"github.com/containerssh/log"
	"github.com/containerssh/metrics"
	"github.com/containerssh/sshserver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewKubeRun creates a handler based on the legacy "kuberun" configuration.
// Deprecated: Use New instead.
//goland:noinspection GoDeprecation,GoUnusedExportedFunction
func NewKubeRun(
	client net.TCPAddr,
	connectionID string,
	oldConfig KubeRunConfig,
	logger log.Logger,
	backendRequestsMetric metrics.SimpleCounter,
	backendFailuresMetric metrics.SimpleCounter,
) (sshserver.NetworkConnectionHandler, error) {
	logger.Warningf(
		"You are using the kuberun backend deprecated since ContainerSSH 0.4. This backend will be removed " +
			"in the future. Please switch to the new kubernetes backend as soon as possible. " +
			"See https://containerssh.io/deprecations/kuberun for details.",
	)

	if err := oldConfig.Validate(); err != nil {
		return nil, err
	}

	config := Config{}
	config.Pod = PodConfig{
		Metadata: metav1.ObjectMeta{
			Namespace: oldConfig.Pod.Namespace,
			GenerateName: "containerssh-",
		},
		ConsoleContainerNumber: oldConfig.Pod.ConsoleContainerNumber,
		Spec:                   oldConfig.Pod.Spec,
		Subsystems:             oldConfig.Pod.Subsystems,
		DisableAgent:           true,
		IdleCommand:            nil,
		ShellCommand:           nil,
		Mode:                   ExecutionModeSession,
		disableCommand:         oldConfig.Pod.DisableCommand,
	}
	config.Connection = oldConfig.Connection.ConnectionConfig
	config.Timeouts = TimeoutConfig{
		PodStart:     oldConfig.Timeout,
		PodStop:      oldConfig.Timeout,
		CommandStart: oldConfig.Timeout,
		Signal:       oldConfig.Timeout,
		Window:       oldConfig.Timeout,
		HTTP:         oldConfig.Connection.Timeout,
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return New(
		client,
		connectionID,
		config,
		logger,
		backendRequestsMetric,
		backendFailuresMetric,
	)
}
