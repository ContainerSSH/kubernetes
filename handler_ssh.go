package kubernetes

import (
	"github.com/containerssh/sshserver"
)

type sshConnectionHandler struct {
	sshserver.AbstractSSHConnectionHandler

	networkHandler *networkHandler
	username       string
}

func (s *sshConnectionHandler) OnSessionChannel(channelID uint64, _ []byte, session sshserver.SessionChannel) (
	channel sshserver.SessionChannelHandler,
	failureReason sshserver.ChannelRejection,
) {
	return &channelHandler{
		session:        session,
		channelID:      channelID,
		networkHandler: s.networkHandler,
		username:       s.username,
		env:            map[string]string{},
	}, nil
}
