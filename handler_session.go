package kubernetes

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/containerssh/sshserver"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/remotecommand"
	execUtil "k8s.io/client-go/util/exec"
	"k8s.io/kubectl/pkg/scheme"
)

type channelHandler struct {
	networkHandler    *networkHandler
	env               map[string]string
	sshHandler        *sshConnectionHandler
	running           bool
	pty               bool
	columns           uint32
	rows              uint32
	channelID         uint64
	terminalSizeQueue pushSizeQueue
}

func (c *channelHandler) OnUnsupportedChannelRequest(_ uint64, _ string, _ []byte) {}

func (c *channelHandler) OnFailedDecodeChannelRequest(_ uint64, _ string, _ []byte, _ error) {}

func (c *channelHandler) OnEnvRequest(_ uint64, name string, value string) error {
	c.sshHandler.mutex.Lock()
	defer c.sshHandler.mutex.Unlock()
	if c.running {
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
	c.sshHandler.mutex.Lock()
	defer c.sshHandler.mutex.Unlock()
	if c.running {
		return fmt.Errorf("program already running")
	}
	c.env["TERM"] = term
	c.pty = true
	c.columns = columns
	c.rows = rows
	return nil
}

func (c *channelHandler) parseProgram(program string) []string {
	programParts, err := shellwords.Parse(program)
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
	program []string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	onExit func(exitStatus sshserver.ExitStatus),
) error {
	c.networkHandler.mutex.Lock()
	defer c.networkHandler.mutex.Unlock()

	container := c.networkHandler.pod.Spec.Containers[c.networkHandler.config.Pod.ConsoleContainerNumber]

	go c.streamIO(program, stdin, stdout, stderr, container, onExit)

	return nil
}

func (c *channelHandler) streamIO(
	program []string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	container corev1.Container,
	exit func(exitStatus sshserver.ExitStatus),
) {
	c.initTerminalSizeQueue()

	c.stream(program, container, stdin, stdout, stderr, exit)
}

func (c *channelHandler) stream(
	program []string,
	container corev1.Container,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	exit func(exitStatus sshserver.ExitStatus),
) {
	req := c.networkHandler.restClient.Post().
		Resource("pods").
		Name(c.networkHandler.pod.Name).
		Namespace(c.networkHandler.pod.Namespace).
		SubResource("exec")
	req.VersionedParams(
		&corev1.PodExecOptions{
			Container: container.Name,
			Command:   program,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       c.pty,
		},
		scheme.ParameterCodec,
	)

	exec, err := remotecommand.NewSPDYExecutor(
		&c.networkHandler.restClientConfig,
		"POST",
		req.URL(),
	)
	if err != nil {
		exit(137)
		c.networkHandler.logger.Warningf("failed to stream IO (%v)", err)
		return
	}
	if err := exec.Stream(
		remotecommand.StreamOptions{
			Stdin:             stdin,
			Stdout:            stdout,
			Stderr:            stderr,
			Tty:               c.pty,
			TerminalSizeQueue: c.terminalSizeQueue,
		},
	); err != nil {
		exitErr := execUtil.CodeExitError{}
		if errors.As(err, &exitErr) {
			exit(sshserver.ExitStatus(exitErr.Code))
			return
		}
		c.networkHandler.logger.Warningf("failed to stream IO (%v)", err)
		exit(137)
		return
	} else {
		exit(0)
	}
}

func (c *channelHandler) initTerminalSizeQueue() {
	if c.pty {
		c.terminalSizeQueue.Push(
			remotecommand.TerminalSize{
				Width:  uint16(c.columns),
				Height: uint16(c.rows),
			},
		)
	}
}

func (c *channelHandler) OnExecRequest(
	_ uint64,
	program string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	onExit func(exitStatus sshserver.ExitStatus),
) error {
	return c.run(c.parseProgram(program), stdin, stdout, stderr, onExit)
}

func (c *channelHandler) OnShell(
	_ uint64,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	onExit func(exitStatus sshserver.ExitStatus),
) error {
	return c.run(nil, stdin, stdout, stderr, onExit)
}

func (c *channelHandler) OnSubsystem(
	_ uint64,
	subsystem string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	onExit func(exitStatus sshserver.ExitStatus),
) error {
	if binary, ok := c.networkHandler.config.Pod.Subsystems[subsystem]; ok {
		return c.run([]string{binary}, stdin, stdout, stderr, onExit)
	}
	return fmt.Errorf("subsystem not supported")
}

func (c *channelHandler) OnSignal(_ uint64, _ string) error {
	return fmt.Errorf("signals are not supported")
}

func (c *channelHandler) OnWindow(_ uint64, columns uint32, rows uint32, _ uint32, _ uint32) error {
	c.sshHandler.mutex.Lock()
	defer c.sshHandler.mutex.Unlock()
	if !c.running {
		return fmt.Errorf("program not running")
	}

	c.terminalSizeQueue.Push(remotecommand.TerminalSize{
		Width:  uint16(columns),
		Height: uint16(rows),
	})

	return nil
}
