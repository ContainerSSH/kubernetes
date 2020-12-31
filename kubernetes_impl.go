package kubernetes

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/containerssh/log"
	"github.com/containerssh/metrics"
	"github.com/containerssh/structutils"
	core "k8s.io/api/core/v1"
	kubeErrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/remotecommand"
	watchTools "k8s.io/client-go/tools/watch"
	"k8s.io/client-go/util/exec"
)

type kubeClientFactory struct {
	backendRequestsMetric metrics.SimpleCounter
	backendFailuresMetric metrics.SimpleCounter
}

func (f *kubeClientFactory) get(
	_ context.Context,
	config Config,
	logger log.Logger,
) (kubernetesClient, error) {
	connectionConfig := createConnectionConfig(config)

	cli, err := kubernetes.NewForConfig(&connectionConfig)
	if err != nil {
		return nil, err
	}

	restClient, err := restclient.RESTClientFor(&connectionConfig)
	if err != nil {
		return nil, err
	}

	return &kubeClient{
		client:                cli,
		restClient:            restClient,
		config:                config,
		logger:                logger,
		connectionConfig:      &connectionConfig,
		backendRequestsMetric: f.backendRequestsMetric,
		backendFailuresMetric: f.backendFailuresMetric,
	}, nil
}

func createConnectionConfig(config Config) restclient.Config {
	return restclient.Config{
		Host:    config.Connection.Host,
		APIPath: config.Connection.APIPath,
		ContentConfig: restclient.ContentConfig{
			GroupVersion:         &core.SchemeGroupVersion,
			NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
		},
		Username:        config.Connection.Username,
		Password:        config.Connection.Password,
		BearerToken:     config.Connection.BearerToken,
		BearerTokenFile: config.Connection.BearerTokenFile,
		Impersonate:     restclient.ImpersonationConfig{},
		TLSClientConfig: restclient.TLSClientConfig{
			Insecure:   config.Connection.Insecure,
			ServerName: config.Connection.ServerName,
			CertFile:   config.Connection.CertFile,
			KeyFile:    config.Connection.KeyFile,
			CAFile:     config.Connection.CAFile,
			CertData:   []byte(config.Connection.CertData),
			KeyData:    []byte(config.Connection.KeyData),
			CAData:     []byte(config.Connection.CAData),
		},
		UserAgent: "ContainerSSH",
		QPS:       config.Connection.QPS,
		Burst:     config.Connection.Burst,
		Timeout:   config.Timeouts.HTTP,
	}
}

type kubeClient struct {
	config                Config
	logger                log.Logger
	client                *kubernetes.Clientset
	restClient            *restclient.RESTClient
	connectionConfig      *restclient.Config
	backendRequestsMetric metrics.SimpleCounter
	backendFailuresMetric metrics.SimpleCounter
}

func (k *kubeClient) createPod(
	ctx context.Context,
	labels map[string]string,
	env map[string]string,
	tty *bool,
	cmd []string,
) (kubernetesPod, error) {
	podConfig, err := k.getPodConfig(tty, cmd, labels, env)
	if err != nil {
		return nil, err
	}

	var pod *core.Pod
	var lastError error
loop:
	for {
		k.backendRequestsMetric.Increment()
		pod, lastError = k.client.CoreV1().Pods(k.config.Pod.Metadata.Namespace).Create(
			ctx,
			&core.Pod{
				ObjectMeta: podConfig.Metadata,
				Spec:       podConfig.Spec,
			},
			meta.CreateOptions{},
		)
		if lastError == nil {
			createdPod := &kubePod{
				pod:                   pod,
				client:                k.client,
				restClient:            k.restClient,
				config:                k.config,
				logger:                k.logger,
				tty:                   tty,
				connectionConfig:      k.connectionConfig,
				backendRequestsMetric: k.backendRequestsMetric,
				backendFailuresMetric: k.backendFailuresMetric,
			}
			return createdPod.wait(ctx)
		}
		k.backendFailuresMetric.Increment()
		k.logger.Warninge(
			fmt.Errorf("failed to create pod, retrying in 10 seconds (%w)", lastError),
		)
		select {
		case <-ctx.Done():
			break loop
		case <-time.After(10 * time.Second):
		}
	}
	if lastError == nil {
		lastError = fmt.Errorf("timeout")
	}
	err = fmt.Errorf("failed to create pod, giving up (%w)", lastError)
	k.logger.Errore(
		err,
	)
	return nil, err
}

func (k *kubeClient) getPodConfig(tty *bool, cmd []string, labels map[string]string, env map[string]string) (
	PodConfig,
	error,
) {
	var podConfig PodConfig
	if err := structutils.Copy(&podConfig, k.config.Pod); err != nil {
		return PodConfig{}, err
	}

	if podConfig.Mode == ExecutionModeSession {
		if tty != nil {
			podConfig.Spec.Containers[k.config.Pod.ConsoleContainerNumber].TTY = *tty
		}
		if !podConfig.DisableAgent {
			podConfig.Spec.Containers[k.config.Pod.ConsoleContainerNumber].Command = append(
				[]string{
					podConfig.AgentPath,
					"console",
					"--wait",
					"--pid",
					"--",
				},
				cmd...,
			)
			podConfig.Spec.Containers[k.config.Pod.ConsoleContainerNumber].Stdin = true
			podConfig.Spec.Containers[k.config.Pod.ConsoleContainerNumber].StdinOnce = true
		} else {
			podConfig.Spec.Containers[k.config.Pod.ConsoleContainerNumber].Command = cmd
		}
		if podConfig.Spec.RestartPolicy == "" {
			podConfig.Spec.RestartPolicy = core.RestartPolicyNever
		}
	} else {
		podConfig.Spec.Containers[k.config.Pod.ConsoleContainerNumber].Command = k.config.Pod.IdleCommand
	}

	if podConfig.Metadata.Labels == nil {
		podConfig.Metadata.Labels = map[string]string{}
	}
	for k, v := range labels {
		podConfig.Metadata.Labels[k] = v
	}
	for key, value := range env {
		podConfig.Spec.Containers[k.config.Pod.ConsoleContainerNumber].Env = append(
			podConfig.Spec.Containers[k.config.Pod.ConsoleContainerNumber].Env,
			core.EnvVar{
				Name:  key,
				Value: value,
			},
		)
	}
	return podConfig, nil
}

type pushSizeQueue interface {
	remotecommand.TerminalSizeQueue

	Push(remotecommand.TerminalSize)
	Stop()
}

type sizeQueue struct {
	resizeChan chan remotecommand.TerminalSize
}

func (s *sizeQueue) Push(size remotecommand.TerminalSize) {
	s.resizeChan <- size
}

func (s *sizeQueue) Next() *remotecommand.TerminalSize {
	size, ok := <-s.resizeChan
	if !ok {
		return nil
	}
	return &size
}

func (s *sizeQueue) Stop() {
	close(s.resizeChan)
}

type kubePod struct {
	config                Config
	pod                   *core.Pod
	client                *kubernetes.Clientset
	restClient            *restclient.RESTClient
	logger                log.Logger
	tty                   *bool
	connectionConfig      *restclient.Config
	backendRequestsMetric metrics.SimpleCounter
	backendFailuresMetric metrics.SimpleCounter
}

type kubeExec struct {
	pod                   *kubePod
	exec                  remotecommand.Executor
	terminalSizeQueue     pushSizeQueue
	logger                log.Logger
	tty                   bool
	pid                   int
	env                   map[string]string
	backendRequestsMetric metrics.SimpleCounter
	backendFailuresMetric metrics.SimpleCounter
}

var cannotSendSignalError = errors.New("cannot send signal")

func (k *kubeExec) signal(ctx context.Context, sig string) error {
	if k.pid <= 0 {
		return cannotSendSignalError
	}
	return k.sendSignalToProcess(ctx, sig)
}

func (k *kubeExec) sendSignalToProcess(ctx context.Context, sig string) error {
	if k.pod.config.Pod.DisableAgent {
		return fmt.Errorf("cannot send signal")
	}
	k.logger.Debugf("Using the exec facility to send signal %s to pid %d...", sig, k.pid)
	podExec, err := k.pod.createExec(
		ctx, []string{
			k.pod.config.Pod.AgentPath,
			"signal",
			"--pid",
			strconv.Itoa(k.pid),
			"--signal",
			sig,
		}, map[string]string{}, false,
	)
	if err != nil {
		k.logger.Errorf(
			"cannot send %s signal to pod %s pid %d (%v)",
			sig, k.pod.pod.Name, k.pid, err,
		)
		return cannotSendSignalError
	}
	var stdoutBytes bytes.Buffer
	var stderrBytes bytes.Buffer
	stdin, stdinWriter := io.Pipe()
	done := make(chan struct{})
	podExec.run(
		&stdoutBytes, &stderrBytes, stdin, func(exitStatus int) {
			if exitStatus != 0 {
				k.backendFailuresMetric.Increment()
				err = cannotSendSignalError
				k.logger.Errorf(
					"cannot send %s signal to pod %s pid %d (%s)",
					sig, k.pod.pod.Name, k.pid, stderrBytes,
				)
			}
			done <- struct{}{}
		},
	)
	<-done
	_ = stdinWriter.Close()
	return err
}

func (k *kubeExec) resize(_ context.Context, height uint, width uint) error {
	//TODO handle ctx
	k.terminalSizeQueue.Push(
		remotecommand.TerminalSize{
			Width:  uint16(width),
			Height: uint16(height),
		},
	)
	return nil
}

func (k *kubeExec) run(stdout io.Writer, stderr io.Writer, stdin io.Reader, onExit func(exitStatus int)) {
	if !k.pod.config.Pod.DisableAgent {
		var stdinWriter io.WriteCloser
		var stdoutReader io.ReadCloser
		originalStdin := stdin
		originalStdout := stdout
		stdin, stdinWriter = io.Pipe()
		stdoutReader, stdout = io.Pipe()
		defer func() {
			_ = stdinWriter.Close()
			_ = stdoutReader.Close()
		}()
		go func() {
			if k.pod.config.Pod.Mode == ExecutionModeSession {
				// Start the program. See https://github.com/containerssh/agent for details.
				if _, err := stdinWriter.Write([]byte("\000")); err != nil {
					k.logger.Warningf("failed to start program (%v)", err)
				}
			}
			// Read the pid. See https://github.com/containerssh/agent for details.
			pidBytes := make([]byte, 4)
			if _, err := stdoutReader.Read(pidBytes); err != nil {
				k.logger.Warningf("failed to read PID from program (%v)", err)
			} else {
				k.pid = int(binary.LittleEndian.Uint32(pidBytes))
			}

			go func() {
				if _, err := io.Copy(originalStdout, stdoutReader); err != nil {
					if !errors.Is(err, io.ErrClosedPipe) {
						k.logger.Warningf("failed to read from stdout (%v)", err)
					}
				}
			}()
			go func() {
				if _, err := io.Copy(stdinWriter, originalStdin); err != nil {
					if !errors.Is(err, io.ErrClosedPipe) {
						k.logger.Warningf("failed to read from stdin (%v)", err)
					}
				}
			}()
		}()
	}
	k.handleStream(stdout, stderr, stdin, onExit)
}

func (k *kubeExec) handleStream(stdout io.Writer, stderr io.Writer, stdin io.Reader, onExit func(exitStatus int)) {
	var tty bool
	if k.pod.config.Pod.Mode == ExecutionModeSession {
		tty = *k.pod.tty
	} else {
		tty = k.tty
	}
	k.backendRequestsMetric.Increment()
	err := k.exec.Stream(
		remotecommand.StreamOptions{
			Stdin:             stdin,
			Stdout:            stdout,
			Stderr:            stderr,
			Tty:               tty,
			TerminalSizeQueue: k.terminalSizeQueue,
		},
	)

	k.terminalSizeQueue.Stop()
	if err != nil {
		exitErr := &exec.CodeExitError{}
		if errors.As(err, exitErr) {
			onExit(exitErr.Code)
		} else {
			k.backendFailuresMetric.Increment()
			k.pod.logger.Errore(err)
			onExit(137)
		}
	} else {
		onExit(0)
	}
}

func (k *kubePod) attach(_ context.Context) (kubernetesExecution, error) {
	req := k.restClient.Post().
		Namespace(k.pod.Namespace).
		Resource("pods").
		Name(k.pod.Name).
		SubResource("attach")
	req.VersionedParams(
		&core.PodAttachOptions{
			Container: k.pod.Spec.Containers[k.config.Pod.ConsoleContainerNumber].Name,
			Stdin:     true,
			Stdout:    true,
			Stderr:    !*k.tty,
			TTY:       *k.tty,
		}, scheme.ParameterCodec,
	)

	podExec, err := remotecommand.NewSPDYExecutor(k.connectionConfig, "POST", req.URL())
	if err != nil {
		return nil, err
	}

	return &kubeExec{
		pod:  k,
		exec: podExec,
		terminalSizeQueue: &sizeQueue{
			resizeChan: make(chan remotecommand.TerminalSize),
		},
		logger:                k.logger,
		tty:                   *k.tty,
		backendRequestsMetric: k.backendRequestsMetric,
		backendFailuresMetric: k.backendFailuresMetric,
	}, nil
}

func (k *kubePod) createExec(
	_ context.Context,
	program []string,
	env map[string]string,
	tty bool,
) (kubernetesExecution, error) {
	if !k.config.Pod.DisableAgent {
		newProgram := []string{
			k.config.Pod.AgentPath,
			"console",
			"--pid",
		}
		for envKey, envValue := range env {
			newProgram = append(newProgram, "--env", fmt.Sprintf("%s=%s", envKey, envValue))
		}
		newProgram = append(newProgram, "--")
		program = append(newProgram, program...)
	}

	req := k.restClient.Post().
		Resource("pods").
		Name(k.pod.Name).
		Namespace(k.pod.Namespace).
		SubResource("exec")
	req.VersionedParams(
		&core.PodExecOptions{
			Container: k.pod.Spec.Containers[k.config.Pod.ConsoleContainerNumber].Name,
			Command:   program,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       tty,
		},
		scheme.ParameterCodec,
	)

	podExec, err := remotecommand.NewSPDYExecutor(
		k.connectionConfig,
		"POST",
		req.URL(),
	)
	if err != nil {
		return nil, err
	}

	return &kubeExec{
		pod:  k,
		exec: podExec,
		terminalSizeQueue: &sizeQueue{
			resizeChan: make(chan remotecommand.TerminalSize),
		},
		logger:                k.logger,
		env:                   env,
		tty:                   tty,
		backendRequestsMetric: k.backendRequestsMetric,
		backendFailuresMetric: k.backendFailuresMetric,
	}, nil
}

func (k *kubePod) remove(ctx context.Context) error {
	var lastError error
loop:
	for {
		lastError = k.client.CoreV1().Pods(k.pod.Namespace).Delete(ctx, k.pod.Name, meta.DeleteOptions{})
		if lastError == nil {
			return nil
		}
		k.logger.Warninge(
			fmt.Errorf("failed to remove pod, retrying in 10 seconds (%w)", lastError),
		)
		select {
		case <-ctx.Done():
			break loop
		case <-time.After(10 * time.Second):
		}
	}
	if lastError == nil {
		lastError = fmt.Errorf("timeout")
	}
	err := fmt.Errorf("failed to remove pod, giving up (%w)", lastError)
	k.logger.Warninge(
		err,
	)
	return err
}

func (k *kubePod) wait(ctx context.Context) (kubernetesPod, error) {
	k.backendRequestsMetric.Increment()
	fieldSelector := fields.
		OneTermEqualSelector("metadata.name", k.pod.Name).
		String()
	listWatch := &cache.ListWatch{
		ListFunc: func(options meta.ListOptions) (runtime.Object, error) {
			options.FieldSelector = fieldSelector
			return k.client.
				CoreV1().
				Pods(k.pod.Namespace).
				List(ctx, options)
		},
		WatchFunc: func(options meta.ListOptions) (watch.Interface, error) {
			options.FieldSelector = fieldSelector
			return k.client.
				CoreV1().
				Pods(k.pod.Namespace).
				Watch(ctx, options)
		},
	}

	event, err := watchTools.UntilWithSync(
		ctx,
		listWatch,
		&core.Pod{},
		nil,
		k.isPodAvailableEvent,
	)
	if event != nil {
		k.pod = event.Object.(*core.Pod)
	}
	if err != nil {
		k.backendFailuresMetric.Increment()
	}
	return k, err
}

func (k *kubePod) isPodAvailableEvent(event watch.Event) (bool, error) {
	if event.Type == watch.Deleted {
		return false, kubeErrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "")
	}

	switch eventObject := event.Object.(type) {
	case *core.Pod:
		switch eventObject.Status.Phase {
		case core.PodFailed, core.PodSucceeded:
			return true, nil
		case core.PodRunning:
			conditions := eventObject.Status.Conditions
			for _, condition := range conditions {
				if condition.Type == core.PodReady &&
					condition.Status == core.ConditionTrue {
					return true, nil
				}
			}
		}
	}
	return false, nil
}
