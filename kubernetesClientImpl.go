package kubernetes

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/containerssh/log"
	"github.com/containerssh/metrics"
	"github.com/containerssh/structutils"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

type kubernetesClientImpl struct {
	config                Config
	logger                log.Logger
	client                *kubernetes.Clientset
	restClient            *restclient.RESTClient
	connectionConfig      *restclient.Config
	backendRequestsMetric metrics.SimpleCounter
	backendFailuresMetric metrics.SimpleCounter
}

func (k *kubernetesClientImpl) createPod(
	ctx context.Context,
	labels map[string]string,
	annotations map[string]string,
	env map[string]string,
	tty *bool,
	cmd []string,
) (kubePod kubernetesPod, lastError error) {
	podConfig, err := k.getPodConfig(tty, cmd, labels, annotations, env)
	if err != nil {
		return nil, err
	}
	logger := k.logger

	logger.Debug(log.NewMessage(MPodCreate, "Creating pod"))
loop:
	for {
		kubePod, lastError = k.attemptPodCreate(ctx, podConfig, logger, tty)
		if lastError == nil {
			return kubePod, nil
		}
		select {
		case <-ctx.Done():
			break loop
		case <-time.After(10 * time.Second):
		}
	}
	if lastError == nil {
		lastError = fmt.Errorf("timeout")
	}
	err = log.WrapUser(
		lastError,
		EFailedPodCreate,
		UserMessageInitializeSSHSession,
		"Failed to create pod, giving up",
	)
	logger.Error(err)
	return nil, err
}

func (k *kubernetesClientImpl) attemptPodCreate(
	ctx context.Context,
	podConfig PodConfig,
	logger log.Logger,
	tty *bool,
) (kubernetesPod, error) {
	var pod *core.Pod
	var lastError error
	k.backendRequestsMetric.Increment()
	pod, lastError = k.client.CoreV1().Pods(podConfig.Metadata.Namespace).Create(
		ctx,
		&core.Pod{
			ObjectMeta: podConfig.Metadata,
			Spec:       podConfig.Spec,
		},
		meta.CreateOptions{},
	)
	if lastError == nil {
		createdPod := &kubernetesPodImpl{
			pod:                   pod,
			client:                k.client,
			restClient:            k.restClient,
			config:                k.config,
			logger:                logger.WithLabel("podName", pod.Name),
			tty:                   tty,
			connectionConfig:      k.connectionConfig,
			backendRequestsMetric: k.backendRequestsMetric,
			backendFailuresMetric: k.backendFailuresMetric,
			lock:                  &sync.Mutex{},
			wg:                    &sync.WaitGroup{},
			removeLock:            &sync.Mutex{},
		}
		return createdPod.wait(ctx)
	}
	k.backendFailuresMetric.Increment()
	logger.Debug(
		log.Wrap(
			lastError,
			EFailedPodCreate,
			"Failed to create pod, retrying in 10 seconds",
		),
	)
	return nil, lastError
}

func (k *kubernetesClientImpl) getPodConfig(
	tty *bool,
	cmd []string,
	labels map[string]string,
	annotations map[string]string,
	env map[string]string,
) (
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
		podConfig.Spec.Containers[k.config.Pod.ConsoleContainerNumber].Stdin = true
		podConfig.Spec.Containers[k.config.Pod.ConsoleContainerNumber].StdinOnce = true
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
		} else {
			podConfig.Spec.Containers[k.config.Pod.ConsoleContainerNumber].Command = cmd
		}
		if podConfig.Spec.RestartPolicy == "" {
			podConfig.Spec.RestartPolicy = core.RestartPolicyNever
		}
	} else {
		podConfig.Spec.Containers[k.config.Pod.ConsoleContainerNumber].Command = k.config.Pod.IdleCommand
	}

	k.addLabelsToPodConfig(podConfig, labels)
	k.addAnnotationsToPodConfig(podConfig, annotations)
	k.addEnvToPodConfig(env, podConfig)
	return podConfig, nil
}

func (k *kubernetesClientImpl) addLabelsToPodConfig(podConfig PodConfig, labels map[string]string) {
	if podConfig.Metadata.Labels == nil {
		podConfig.Metadata.Labels = map[string]string{}
	}
	for k, v := range labels {
		podConfig.Metadata.Labels[k] = v
	}
}

func (k *kubernetesClientImpl) addAnnotationsToPodConfig(podConfig PodConfig, annotations map[string]string) {
	if podConfig.Metadata.Annotations == nil {
		podConfig.Metadata.Annotations = map[string]string{}
	}
	for k, v := range annotations {
		podConfig.Metadata.Annotations[k] = v
	}
}

func (k *kubernetesClientImpl) addEnvToPodConfig(env map[string]string, podConfig PodConfig) {
	for key, value := range env {
		podConfig.Spec.Containers[k.config.Pod.ConsoleContainerNumber].Env = append(
			podConfig.Spec.Containers[k.config.Pod.ConsoleContainerNumber].Env,
			core.EnvVar{
				Name:  key,
				Value: value,
			},
		)
	}
}
