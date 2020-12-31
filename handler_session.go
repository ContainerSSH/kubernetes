package kubernetes

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/containerssh/sshserver"
	"github.com/containerssh/unixutils"
)

type channelHandler struct {
	channelID      uint64
	networkHandler *networkHandler
	username       string
	env            map[string]string
	pty            bool
	columns        uint32
	rows           uint32
	exitSent       bool
	exec           kubernetesExecution
}

func (c *channelHandler) OnUnsupportedChannelRequest(_ uint64, _ string, _ []byte) {}

func (c *channelHandler) OnFailedDecodeChannelRequest(_ uint64, _ string, _ []byte, _ error) {}

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
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	onExit func(exitStatus sshserver.ExitStatus),
) error {
	c.networkHandler.mutex.Lock()
	defer c.networkHandler.mutex.Unlock()

	var err error
	var realOnExit func(exitStatus sshserver.ExitStatus)
	var pod kubernetesPod
	switch c.networkHandler.config.Pod.Mode {
	case ExecutionModeConnection:
		realOnExit, err = c.handleExecModeConnection(ctx, program, onExit)
		if err != nil {
			return err
		}
	case ExecutionModeSession:
		realOnExit, pod, err = c.handleExecModeSession(ctx, program, onExit)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid execution mode: %s", c.networkHandler.config.Pod.Mode)
	}

	go c.exec.run(
		stdout, stderr, stdin, func(exitStatus int) {
			c.networkHandler.mutex.Lock()
			defer c.networkHandler.mutex.Unlock()
			if c.exitSent {
				return
			}
			c.exitSent = true
			realOnExit(sshserver.ExitStatus(exitStatus))
		},
	)

	if c.pty {
		err = c.exec.resize(ctx, uint(c.rows), uint(c.columns))
		if err != nil && c.networkHandler.config.Pod.Mode == ExecutionModeSession && pod != nil {
			c.removePod(pod)
			c.networkHandler.logger.Debugf("failed to set initial terminal size (%v)", err)
			return fmt.Errorf("failed to set terminal size")
		}
	}

	return nil
}

func (c *channelHandler) handleExecModeConnection(
	ctx context.Context,
	program []string,
	onExit func(exitStatus sshserver.ExitStatus),
) (func(exitStatus sshserver.ExitStatus), error) {
	exec, err := c.networkHandler.pod.createExec(ctx, program, c.env, c.pty)
	if err != nil {
		return nil, err
	}
	c.exec = exec
	return onExit, nil
}

func (c *channelHandler) handleExecModeSession(
	ctx context.Context,
	program []string,
	onExit func(exitStatus sshserver.ExitStatus),
) (func(exitStatus sshserver.ExitStatus), kubernetesPod, error) {
	pod, err := c.networkHandler.cli.createPod(
		ctx,
		c.networkHandler.labels,
		c.env,
		&c.pty,
		program,
	)
	if err != nil {
		return nil, nil, err
	}
	c.exec, err = pod.attach(ctx)
	if err != nil {
		c.removePod(pod)
		return nil, nil, err
	}
	onExitWrapper := func(exitStatus sshserver.ExitStatus) {
		onExit(exitStatus)
		c.removePod(pod)
	}
	return onExitWrapper, pod, nil
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
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	onExit func(exitStatus sshserver.ExitStatus),
) error {
	if c.networkHandler.config.Pod.disableCommand {
		return fmt.Errorf("command execution is disabled")
	}
	startContext, cancelFunc := context.WithTimeout(context.Background(), c.networkHandler.config.Timeouts.CommandStart)
	defer cancelFunc()

	return c.run(startContext, c.parseProgram(program), stdin, stdout, stderr, onExit)
}

func (c *channelHandler) OnShell(
	_ uint64,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	onExit func(exitStatus sshserver.ExitStatus),
) error {
	startContext, cancelFunc := context.WithTimeout(context.Background(), c.networkHandler.config.Timeouts.CommandStart)
	defer cancelFunc()

	return c.run(startContext, c.networkHandler.config.Pod.ShellCommand, stdin, stdout, stderr, onExit)
}

func (c *channelHandler) OnSubsystem(
	_ uint64,
	subsystem string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	onExit func(exitStatus sshserver.ExitStatus),
) error {
	startContext, cancelFunc := context.WithTimeout(context.Background(), c.networkHandler.config.Timeouts.CommandStart)
	defer cancelFunc()

	if binary, ok := c.networkHandler.config.Pod.Subsystems[subsystem]; ok {
		return c.run(startContext, []string{binary}, stdin, stdout, stderr, onExit)
	}
	return fmt.Errorf("subsystem not supported")
}

func (c *channelHandler) OnSignal(_ uint64, signal string) error {
	c.networkHandler.mutex.Lock()
	defer c.networkHandler.mutex.Unlock()
	if c.exec == nil {
		return fmt.Errorf("program not running")
	}
	ctx, cancelFunc := context.WithTimeout(context.Background(), c.networkHandler.config.Timeouts.Signal)
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
