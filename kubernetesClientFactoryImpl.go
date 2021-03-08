package kubernetes

import (
	"context"

	"github.com/containerssh/log"
	"github.com/containerssh/metrics"
	core "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
)

type kubernetesClientFactoryImpl struct {
	backendRequestsMetric metrics.SimpleCounter
	backendFailuresMetric metrics.SimpleCounter
}

func (f *kubernetesClientFactoryImpl) get(
	_ context.Context,
	config Config,
	logger log.Logger,
) (kubernetesClient, error) {
	connectionConfig := f.createConnectionConfig(config)

	cli, err := kubernetes.NewForConfig(&connectionConfig)
	if err != nil {
		err = log.WrapUser(
			err,
			EConfigError,
			UserMessageInitializeSSHSession,
			"Failed to initialize Kubernetes client.",
		)
		logger.Error(err)
		return nil, err
	}

	restClient, err := restclient.RESTClientFor(&connectionConfig)
	if err != nil {
		err = log.WrapUser(
			err,
			EConfigError,
			UserMessageInitializeSSHSession,
			"Failed to initialize Kubernetes REST client.",
		)
		logger.Error(err)
		return nil, err
	}

	return &kubernetesClientImpl{
		client:                cli,
		restClient:            restClient,
		config:                config,
		logger:                logger,
		connectionConfig:      &connectionConfig,
		backendRequestsMetric: f.backendRequestsMetric,
		backendFailuresMetric: f.backendFailuresMetric,
	}, nil
}

func (f *kubernetesClientFactoryImpl) createConnectionConfig(config Config) restclient.Config {
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
			ServerName: config.Connection.ServerName,
			CertFile:   config.Connection.CertFile,
			KeyFile:    config.Connection.KeyFile,
			CAFile:     config.Connection.CAFile,
			CertData:   []byte(config.Connection.CertData),
			KeyData:    []byte(config.Connection.KeyData),
			CAData:     []byte(config.Connection.CAData),
			Insecure:   config.Connection.insecure,
		},
		UserAgent: "ContainerSSH",
		QPS:       config.Connection.QPS,
		Burst:     config.Connection.Burst,
		Timeout:   config.Timeouts.HTTP,
	}
}
