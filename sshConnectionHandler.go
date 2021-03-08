package kubernetes

import (
	"context"

	"github.com/containerssh/sshserver"
)

type sshConnectionHandler struct {
	networkHandler *networkHandler
	username       string
}

func (s *sshConnectionHandler) OnUnsupportedGlobalRequest(_ uint64, _ string, _ []byte) {
}

func (s *sshConnectionHandler) OnUnsupportedChannel(_ uint64, _ string, _ []byte) {
}

func (s *sshConnectionHandler) OnShutdown(_ context.Context) {
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
