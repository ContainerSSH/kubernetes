package kubernetes_test

import (
	"io/ioutil"
	"net"
	"os"
	"testing"

	"github.com/containerssh/geoip"
	"github.com/containerssh/log"
	"github.com/containerssh/metrics"
	"github.com/containerssh/sshserver"
	"github.com/containerssh/structutils"
	"gopkg.in/yaml.v2"

	"github.com/containerssh/kubernetes"
)

func TestConformance(t *testing.T) {
	var factories = map[string]func() (sshserver.NetworkConnectionHandler, error) {
		"session": func() (sshserver.NetworkConnectionHandler, error) {
			config, err := getKubernetesConfig()
			if err != nil {
				return nil, err
			}
			config.Pod.Mode = kubernetes.ExecutionModeSession
			return getKubernetes(config)
		},
		"connection": func() (sshserver.NetworkConnectionHandler, error) {
			config, err := getKubernetesConfig()
			if err != nil {
				return nil, err
			}
			config.Pod.Mode = kubernetes.ExecutionModeConnection
			return getKubernetes(config)
		},
		"kuberun": func() (sshserver.NetworkConnectionHandler, error) {
			config, err := getKubeRunConfig()
			if err != nil {
				return nil, err
			}
			return getKubeRun(config)
		},
	}

	sshserver.RunConformanceTests(t, factories)
}

func getKubernetes(config kubernetes.Config) (sshserver.NetworkConnectionHandler, error) {
	connectionID := sshserver.GenerateConnectionID()
	geoipProvider, err := geoip.New(geoip.Config{
		Provider: geoip.DummyProvider,
	})
	if err != nil {
		return nil, err
	}
	collector := metrics.New(geoipProvider)
	logger, err := log.New(
		log.Config{
			Level:  log.LevelDebug,
			Format: log.FormatText,
		},
		"kubernetes",
		os.Stdout,
	)
	if err != nil {
		return nil, err
	}
	return kubernetes.New(
		net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 2222,
			Zone: "",
		}, connectionID, config, logger,
		collector.MustCreateCounter("backend_requests", "", ""),
		collector.MustCreateCounter("backend_failures", "", ""),
	)
}

func getKubernetesConfig() (kubernetes.Config, error) {
	config := kubernetes.Config{}
	structutils.Defaults(&config)
	err := config.SetConfigFromKubeConfig()
	return config, err
}

//goland:noinspection GoDeprecation
func getKubeRunConfig() (kubernetes.KubeRunConfig, error) {
	fh, err := os.Open("testdata/config-0.3.yaml")
	if err != nil {
		return kubernetes.KubeRunConfig{}, err
	}
	defer func() {
		_ = fh.Close()
	}()

	data, err := ioutil.ReadAll(fh)
	if err != nil {
		return kubernetes.KubeRunConfig{}, err
	}
	//goland:noinspection GoDeprecation
	config := kubernetes.KubeRunConfig{}
	structutils.Defaults(&config)
	fileConfig := kubernetes.KubeRunConfig{}
	if err := yaml.Unmarshal(data, &fileConfig); err != nil {
		return config, err
	}
	if err := structutils.Merge(&config, &fileConfig); err != nil {
		return config, err
	}
	if err := config.SetConfigFromKubeConfig(); err != nil {
		return config, err
	}
	return config, nil
}

//goland:noinspection GoDeprecation
func getKubeRun(config kubernetes.KubeRunConfig) (sshserver.NetworkConnectionHandler, error) {
	geoipProvider, err := geoip.New(geoip.Config{
		Provider: geoip.DummyProvider,
	})
	if err != nil {
		return nil, err
	}
	collector := metrics.New(geoipProvider)
	logger, err := log.New(
		log.Config{
			Level:  log.LevelDebug,
			Format: log.FormatText,
		},
		"kuberun",
		os.Stdout,
	)
	if err != nil {
		return nil, err
	}
	//goland:noinspection GoDeprecation
	return kubernetes.NewKubeRun(
		net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 2222,
			Zone: "",
		},
		sshserver.GenerateConnectionID(),
		config,
		logger,
		collector.MustCreateCounter("backend_requests", "", ""),
		collector.MustCreateCounter("backend_failures", "", ""),
	)
}
