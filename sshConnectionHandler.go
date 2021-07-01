package kubernetes

import (
	"context"
	"github.com/containerssh/sshserver/v2"
)

type sshConnectionHandler struct {
	networkHandler *networkHandler
	username       string
	env            map[string]string
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
	env := map[string]string{}
	for k, v := range s.env {
		env[k] = v
	}
	return &channelHandler{
		session:        session,
		channelID:      channelID,
		networkHandler: s.networkHandler,
		username:       s.username,
		env:            env,
	}, nil
}
