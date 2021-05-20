package kubernetes

import (
	"net"

	"github.com/containerssh/log"
	"github.com/containerssh/metrics"
	"github.com/containerssh/sshserver/v2"
	"github.com/containerssh/structutils"
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
	if err := validateKubeRunConfig(logger, oldConfig); err != nil {
		return nil, err
	}

	config := Config{}
	structutils.Defaults(&config)
	config.Pod = PodConfig{
		Metadata: metav1.ObjectMeta{
			Namespace:    oldConfig.Pod.Namespace,
			GenerateName: "containerssh-",
		},
		ConsoleContainerNumber: oldConfig.Pod.ConsoleContainerNumber,
		Spec:                   oldConfig.Pod.Spec,
		Subsystems:             oldConfig.Pod.Subsystems,
		DisableAgent:           !oldConfig.Pod.EnableAgent,
		AgentPath:              oldConfig.Pod.AgentPath,
		IdleCommand:            nil,
		ShellCommand:           oldConfig.Pod.ShellCommand,
		Mode:                   ExecutionModeSession,
		disableCommand:         oldConfig.Pod.DisableCommand,
	}
	config.Connection = oldConfig.Connection.ConnectionConfig
	config.Connection.insecure = oldConfig.Connection.Insecure
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

//goland:noinspection GoDeprecation
func validateKubeRunConfig(logger log.Logger, oldConfig KubeRunConfig) error {
	logger.Warning(
		log.NewMessage(
			EKubeRun,
			"You are using the kuberun backend deprecated since ContainerSSH 0.4. This backend will be removed "+
				"in the future. Please switch to the new kubernetes backend as soon as possible. "+
				"See https://containerssh.io/deprecations/kuberun for details.",
		),
	)

	if err := oldConfig.Validate(); err != nil {
		return err
	}

	if oldConfig.Connection.Insecure {
		logger.Warning(
			log.NewMessage(
				EInsecure,
				"You are connecting to your Kubernetes cluster in insecure mode. This is dangerous and highly discouraged.",
			),
		)
	}
	return nil
}
