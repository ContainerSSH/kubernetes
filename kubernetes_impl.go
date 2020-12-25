package kubernetes

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/containerssh/log"
	"github.com/containerssh/structutils"
	core "k8s.io/api/core/v1"
	errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
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
)

type kubeClientFactory struct {
}

func (f *kubeClientFactory) get(
	ctx context.Context,
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
		client:           cli,
		restClient:       restClient,
		config:           config,
		logger:           logger,
		connectionConfig: &connectionConfig,
	}, nil
}

func createConnectionConfig(config Config) restclient.Config {
	return restclient.Config{
		Host:    config.Connection.Host,
		APIPath: config.Connection.APIPath,
		ContentConfig: restclient.ContentConfig{
			GroupVersion:         &v1.SchemeGroupVersion,
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
	config           Config
	logger           log.Logger
	client           *kubernetes.Clientset
	restClient       *restclient.RESTClient
	connectionConfig *restclient.Config
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
				pod:              pod,
				client:           k.client,
				restClient:       k.restClient,
				config:           k.config,
				logger:           k.logger,
				tty:              tty,
				connectionConfig: k.connectionConfig,
			}
			return createdPod.wait(ctx)
		}
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
	if err := structutils.Copy(podConfig, k.config.Pod); err != nil {
		return PodConfig{}, err
	}

	if tty != nil {
		podConfig.Spec.Containers[k.config.Pod.ConsoleContainerNumber].Command = cmd
	} else {
		podConfig.Spec.Containers[k.config.Pod.ConsoleContainerNumber].Command = k.config.Pod.IdleCommand
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

type kubePod struct {
	config           Config
	pod              *core.Pod
	client           *kubernetes.Clientset
	restClient       *restclient.RESTClient
	logger           log.Logger
	tty              *bool
	connectionConfig *restclient.Config
}

type kubeExec struct {
	pod               *kubePod
	exec              remotecommand.Executor
	terminalSizeQueue pushSizeQueue
}

func (k *kubeExec) resize(ctx context.Context, height uint, width uint) error {
	//TODO handle ctx
	k.terminalSizeQueue.Push(remotecommand.TerminalSize{
		Width:  uint16(width),
		Height: uint16(height),
	})
	return nil
}

func (k *kubeExec) run(stdout io.Writer, stderr io.Writer, stdin io.Reader, onExit func(exitStatus int)) {
	err := k.exec.Stream(remotecommand.StreamOptions{
		Stdin:             stdin,
		Stdout:            stdout,
		Stderr:            stderr,
		Tty:               *k.pod.tty,
		TerminalSizeQueue: k.terminalSizeQueue,
	})
	if err != nil {
		k.pod.logger.Errore(err)
		onExit(137)
	}
}

func (k *kubePod) attach(_ context.Context) (kubernetesExecution, error) {
	req := k.restClient.Post().
		Namespace(k.pod.Namespace).
		Resource("pods").
		Name(k.pod.Name).
		SubResource("attach")
	req.VersionedParams(&core.PodAttachOptions{
		Container: k.pod.Spec.Containers[k.config.Pod.ConsoleContainerNumber].Name,
		Stdin:     true,
		Stdout:    true,
		Stderr:    !*k.tty,
		TTY:       *k.tty,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(k.connectionConfig, "POST", req.URL())
	if err != nil {
		return nil, err
	}

	return &kubeExec{
		pod:  k,
		exec: exec,
		terminalSizeQueue: &sizeQueue{
			resizeChan: make(chan remotecommand.TerminalSize),
		},
	}, nil
}

func (k *kubePod) createExec(
	_ context.Context,
	program []string,
	env map[string]string,
	tty bool,
) (kubernetesExecution, error) {
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

	exec, err := remotecommand.NewSPDYExecutor(
		k.connectionConfig,
		"POST",
		req.URL(),
	)
	if err != nil {
		return nil, err
	}

	return &kubeExec{
		pod:  k,
		exec: exec,
		terminalSizeQueue: &sizeQueue{
			resizeChan: make(chan remotecommand.TerminalSize),
		},
	}, nil
}

func (k *kubePod) remove(ctx context.Context) error {
	var lastError error
loop:
	for {
		request := k.restClient.
			Delete().
			Namespace(k.pod.Namespace).
			Resource("pods").
			Name(k.pod.Name).
			Body(&meta.DeleteOptions{})
		result := request.Do(ctx)
		lastError = result.Error()
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
	return k, err
}

func (k *kubePod) isPodAvailableEvent(event watch.Event) (bool, error) {
	if event.Type == watch.Deleted {
		return false, errors.NewNotFound(schema.GroupResource{Resource: "pods"}, "")
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
