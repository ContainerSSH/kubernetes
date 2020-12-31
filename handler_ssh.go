package kubernetes

import (
	"github.com/containerssh/sshserver"
)

type sshConnectionHandler struct {
	networkHandler *networkHandler
	username       string
}

func (s *sshConnectionHandler) OnUnsupportedGlobalRequest(_ uint64, _ string, _ []byte) {}

func (s *sshConnectionHandler) OnUnsupportedChannel(_ uint64, _ string, _ []byte) {}

func (s *sshConnectionHandler) OnSessionChannel(channelID uint64, _ []byte) (
	channel sshserver.SessionChannelHandler,
	failureReason sshserver.ChannelRejection,
) {
	return &channelHandler{
		channelID:      channelID,
		networkHandler: s.networkHandler,
		username:       s.username,
		env:            map[string]string{},
	}, nil
}
