package kubernetes

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/containerssh/log"
	"github.com/containerssh/sshserver"
)

type networkHandler struct {
	sshserver.AbstractNetworkConnectionHandler

	mutex        *sync.Mutex
	client       net.TCPAddr
	connectionID string
	config       Config

	cli          kubernetesClient
	pod          kubernetesPod
	logger       log.Logger
	disconnected bool
	labels       map[string]string
}

func (n *networkHandler) OnAuthPassword(_ string, _ []byte) (response sshserver.AuthResponse, reason error) {
	return sshserver.AuthResponseUnavailable, fmt.Errorf("the backend handler does not support authentication")
}

func (n *networkHandler) OnAuthPubKey(_ string, _ string) (response sshserver.AuthResponse, reason error) {
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
	defer func() {
		cancelFunc()
		n.mutex.Unlock()
	}()

	spec := n.config.Pod.Spec

	spec.Containers[n.config.Pod.ConsoleContainerNumber].Command = n.config.Pod.IdleCommand
	n.labels = map[string]string{
		"containerssh_connection_id": n.connectionID,
		"containerssh_ip":            n.client.IP.String(),
		"containerssh_username":      username,
	}

	var err error
	if n.config.Pod.Mode == ExecutionModeConnection {
		if n.pod, err = n.cli.createPod(ctx, n.labels, nil, nil, nil); err != nil {
			return nil, err
		}
	}

	return &sshConnectionHandler{
		networkHandler: n,
		username:       username,
	}, nil
}

func (n *networkHandler) OnDisconnect() {
	n.disconnected = true
	ctx, cancelFunc := context.WithTimeout(context.Background(), n.config.Timeouts.PodStop)
	defer cancelFunc()
	n.mutex.Lock()
	if n.pod != nil {
		_ = n.pod.remove(ctx)
		n.pod = nil
		n.mutex.Unlock()
	}
}
