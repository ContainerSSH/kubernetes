package kubernetes

import (
	"context"
	"net"
	"sync"

	"github.com/containerssh/log"
	"github.com/containerssh/sshserver"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

func New(config Config, connectionID string, client net.TCPAddr, logger log.Logger) (sshserver.NetworkConnectionHandler, error) {
	connectionConfig := createConnectionConfig(config)

	cli, err := kubernetes.NewForConfig(&connectionConfig)
	if err != nil {
		return nil, err
	}

	restClient, err := restclient.RESTClientFor(&connectionConfig)
	if err != nil {
		return nil, err
	}

	return &networkHandler{
		restClientConfig: connectionConfig,
		mutex:            &sync.Mutex{},
		client:           client,
		connectionID:     connectionID,
		config:           config,
		onDisconnect:     map[uint64]func(){},
		onShutdown:       map[uint64]func(shutdownContext context.Context){},
		cli:              cli,
		restClient:       restClient,
		pod:              nil,
		cancelStart:      nil,
		labels:           nil,
		logger:           logger,
	}, nil
}
