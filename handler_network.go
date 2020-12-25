package kubernetes

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/containerssh/log"
	"github.com/containerssh/sshserver"
	core "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

type networkHandler struct {
	mutex        *sync.Mutex
	client       net.TCPAddr
	connectionID string
	config       Config

	// onDisconnect contains a per-channel disconnect handler
	onDisconnect map[uint64]func()
	onShutdown   map[uint64]func(shutdownContext context.Context)

	cli              *kubernetes.Clientset
	restClient       *restclient.RESTClient
	pod              *core.Pod
	cancelStart      func()
	labels           map[string]string
	logger           log.Logger
	restClientConfig restclient.Config
}

func (n *networkHandler) OnAuthPassword(_ string, _ []byte) (response sshserver.AuthResponse, reason error) {
	return sshserver.AuthResponseUnavailable, fmt.Errorf("the backend handler does not support authentication")
}

func (n *networkHandler) OnAuthPubKey(_ string, _ []byte) (response sshserver.AuthResponse, reason error) {
	return sshserver.AuthResponseUnavailable, fmt.Errorf("the backend handler does not support authentication")
}

func (n *networkHandler) OnHandshakeFailed(_ error) {
}

func (n *networkHandler) OnHandshakeSuccess(username string) (connection sshserver.SSHConnectionHandler, failureReason error) {
	n.mutex.Lock()
	if n.pod != nil {
		n.mutex.Unlock()
		return nil, fmt.Errorf("handshake already complete")
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), n.config.Timeouts.PodStart)
	n.cancelStart = cancelFunc
	defer func() {
		n.cancelStart = nil
		n.mutex.Unlock()
	}()

	spec := n.config.Pod.Spec

	spec.Containers[n.config.Pod.ConsoleContainerNumber].Command = n.config.Pod.IdleCommand
	n.labels = map[string]string{
		"containerssh_connection_id": n.connectionID,
		"containerssh_ip":            n.client.IP.String(),
		"containerssh_username":      username,
	}

	return &sshConnectionHandler{
		networkHandler: n,
		username:       username,
	}, nil
}

func (n *networkHandler) OnDisconnect() {

}
