package kubernetes

import (
	"context"
	"net"
	"sync"

	"github.com/containerssh/log"
	"github.com/containerssh/sshserver"
)

func New(config Config, connectionID string, client net.TCPAddr, logger log.Logger) (sshserver.NetworkConnectionHandler, error) {
	if config.Pod.DisableAgent {
		logger.Warningf("You are using the Kubernetes backend without the ContainerSSH Guest Agent. Several features will not work as expected.")
	}
	if config.Connection.Insecure {
		logger.Warningf("You are connecting to your Kubernetes cluster in insecure mode. This is dangerous and highly discouraged.")
	}

	var clientFactory kubernetesClientFactory = &kubeClientFactory{}

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
