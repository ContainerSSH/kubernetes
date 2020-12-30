package kubernetes

import (
	"context"
	"net"
	"sync"

	"github.com/containerssh/log"
	"github.com/containerssh/metrics"
	"github.com/containerssh/sshserver"
)

func New(
	client net.TCPAddr,
	connectionID string,
	config Config,
	logger log.Logger,
	backendRequestsMetric metrics.SimpleCounter,
	backendFailuresMetric metrics.SimpleCounter,
) (sshserver.NetworkConnectionHandler, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	if config.Pod.DisableAgent {
		logger.Warningf("You are using the Kubernetes backend without the ContainerSSH Guest Agent. Several features will not work as expected.")
	}
	if config.Connection.Insecure {
		logger.Warningf("You are connecting to your Kubernetes cluster in insecure mode. This is dangerous and highly discouraged.")
	}

	var clientFactory kubernetesClientFactory = &kubeClientFactory{
		backendRequestsMetric: backendRequestsMetric,
		backendFailuresMetric: backendFailuresMetric,
	}

	cli, err := clientFactory.get(
		context.Background(),
		config,
		logger,
	)
	if err != nil {
		return nil, err
	}

	return &networkHandler{
		mutex:        &sync.Mutex{},
		client:       client,
		connectionID: connectionID,
		config:       config,
		cli:          cli,
		pod:          nil,
		labels:       nil,
		logger:       logger,
		disconnected: false,
	}, nil
}
