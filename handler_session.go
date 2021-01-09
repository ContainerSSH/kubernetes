package kubernetes

import (
	"context"
	"fmt"
	"strings"

	"github.com/containerssh/sshserver"
	"github.com/containerssh/unixutils"
)

type channelHandler struct {
	sshserver.AbstractSessionChannelHandler

	channelID      uint64
	networkHandler *networkHandler
	username       string
	env            map[string]string
	pty            bool
	columns        uint32
	rows           uint32
	exec           kubernetesExecution
	session        sshserver.SessionChannel
	pod            kubernetesPod
}

func (c *channelHandler) OnEnvRequest(_ uint64, name string, value string) error {
	if c.exec != nil {
		return fmt.Errorf("program already running")
	}
	c.env[name] = value
	return nil
}

func (c *channelHandler) OnPtyRequest(
	_ uint64,
	term string,
	columns uint32,
	rows uint32,
	_ uint32,
	_ uint32,
	_ []byte,
) error {
	if c.exec != nil {
		return fmt.Errorf("program already running")
	}
	c.env["TERM"] = term
	c.pty = true
	c.columns = columns
	c.rows = rows
	return nil
}

func (c *channelHandler) parseProgram(program string) []string {
	programParts, err := unixutils.ParseCMD(program)
	if err != nil {
		return []string{"/bin/sh", "-c", program}
	} else {
		if strings.HasPrefix(programParts[0], "/") || strings.HasPrefix(
			programParts[0],
			"./",
		) || strings.HasPrefix(programParts[0], "../") {
			return programParts
		} else {
			return []string{"/bin/sh", "-c", program}
		}
	}
}

func (c *channelHandler) run(
	ctx context.Context,
	program []string,
) error {
	c.networkHandler.mutex.Lock()
	defer c.networkHandler.mutex.Unlock()

	var err error
	switch c.networkHandler.config.Pod.Mode {
	case ExecutionModeConnection:
		err = c.handleExecModeConnection(ctx, program)
	case ExecutionModeSession:
		c.pod, err = c.handleExecModeSession(ctx, program)
	default:
		return fmt.Errorf("invalid execution mode: %s", c.networkHandler.config.Pod.Mode)
	}
	if err != nil {
		return err
	}

	go c.exec.run(
		c.session.Stdin(),
		c.session.Stdout(),
		c.session.Stderr(),
		c.session.CloseWrite,
		func(exitStatus int) {
			c.session.ExitStatus(uint32(exitStatus))
			_ = c.session.Close()
		},
	)

	if c.pty {
		err = c.exec.resize(ctx, uint(c.rows), uint(c.columns))
		if err != nil && c.networkHandler.config.Pod.Mode == ExecutionModeSession && c.pod != nil {
			c.removePod(c.pod)
			c.networkHandler.logger.Debugf("failed to set initial terminal size (%v)", err)
			return fmt.Errorf("failed to set terminal size")
		}
	}

	return nil
}

func (c *channelHandler) handleExecModeConnection(
	ctx context.Context,
	program []string,
) error {
	exec, err := c.networkHandler.pod.createExec(ctx, program, c.env, c.pty)
	if err != nil {
		return err
	}
	c.exec = exec
	return nil
}

func (c *channelHandler) handleExecModeSession(
	ctx context.Context,
	program []string,
) (kubernetesPod, error) {
	pod, err := c.networkHandler.cli.createPod(
		ctx,
		c.networkHandler.labels,
		c.env,
		&c.pty,
		program,
	)
	if err != nil {
		return nil, err
	}
	c.exec, err = pod.attach(ctx)
	if err != nil {
		c.removePod(pod)
		return nil, err
	}
	return pod, nil
}

func (c *channelHandler) removePod(pod kubernetesPod) {
	ctx, cancelFunc := context.WithTimeout(
		context.Background(), c.networkHandler.config.Timeouts.PodStop,
	)
	defer cancelFunc()
	_ = pod.remove(ctx)
}

func (c *channelHandler) OnExecRequest(
	_ uint64,
	program string,
) error {
	if c.networkHandler.config.Pod.disableCommand {
		return fmt.Errorf("command execution is disabled")
	}
	startContext, cancelFunc := context.WithTimeout(
		context.Background(),
		c.networkHandler.config.Timeouts.CommandStart,
	)
	defer cancelFunc()

	return c.run(startContext, c.parseProgram(program))
}

func (c *channelHandler) OnShell(
	_ uint64,
) error {
	startContext, cancelFunc := context.WithTimeout(
		context.Background(),
		c.networkHandler.config.Timeouts.CommandStart,
	)
	defer cancelFunc()

	return c.run(startContext, c.networkHandler.config.Pod.ShellCommand)
}

func (c *channelHandler) OnSubsystem(
	_ uint64,
	subsystem string,
) error {
	startContext, cancelFunc := context.WithTimeout(
		context.Background(),
		c.networkHandler.config.Timeouts.CommandStart,
	)
	defer cancelFunc()

	if binary, ok := c.networkHandler.config.Pod.Subsystems[subsystem]; ok {
		return c.run(startContext, []string{binary})
	}
	return fmt.Errorf("subsystem not supported")
}

func (c *channelHandler) OnSignal(_ uint64, signal string) error {
	c.networkHandler.mutex.Lock()
	defer c.networkHandler.mutex.Unlock()
	if c.exec == nil {
		return fmt.Errorf("program not running")
	}
	ctx, cancelFunc := context.WithTimeout(
		context.Background(),
		c.networkHandler.config.Timeouts.Signal,
	)
	defer cancelFunc()

	return c.exec.signal(ctx, signal)
}

func (c *channelHandler) OnWindow(_ uint64, columns uint32, rows uint32, _ uint32, _ uint32) error {
	c.networkHandler.mutex.Lock()
	defer c.networkHandler.mutex.Unlock()
	if c.exec == nil {
		return fmt.Errorf("program not running")
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), c.networkHandler.config.Timeouts.Window)
	defer cancelFunc()

	return c.exec.resize(ctx, uint(rows), uint(columns))
}

func (c *channelHandler) OnClose() {
	if c.exec != nil {
		select {
		case <-c.exec.done():
			return
		default:
		}
	}
	if c.networkHandler.config.Pod.Mode == ExecutionModeSession {
		if c.pod != nil {
			c.removePod(c.pod)
		}
	} else if c.exec != nil {
		c.exec.kill()
	}
}

func (c *channelHandler) OnShutdown(shutdownContext context.Context) {
	if c.exec != nil {
		c.exec.term(shutdownContext)
		select {
		case <-shutdownContext.Done():
			if c.networkHandler.config.Pod.Mode == ExecutionModeSession {
				c.removePod(c.pod)
			} else {
				c.exec.kill()
			}
		case <-c.exec.done():
		}
	}
}
