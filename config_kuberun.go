package kubernetes

import (
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
)

// KubeRunConfig is the legacy configuration structure for the "kuberun" backend.
// Deprecated: use Config instead
//goland:noinspection GoDeprecation
type KubeRunConfig struct {
	// Connection configures the connection to the Kubernetes cluster.
	Connection KubeRunConnectionConfig `json:"connection" yaml:"connection" comment:"Kubernetes configuration options"`
	// Pod contains the spec and specific settings for creating the pod.
	Pod KubeRunPodConfig `json:"pod" yaml:"pod" comment:"Container configuration"`
	// Timeout specifies how long to wait for the Pod to come up.
	Timeout time.Duration `json:"timeout" yaml:"timeout" comment:"Timeout for pod creation" default:"60s"`
}

// KubeRunConnectionConfig is the legacy connection configuration structure for the "kuberun" backend.
// Deprecated: use ConnectionConfig insteead.
//goland:noinspection GoDeprecation
type KubeRunConnectionConfig struct {
	// ConnectionConfig is the new configuration structure
	ConnectionConfig `json:",inline" yaml:",inline"`

	// Timeout indicates the timeout for client calls.
	Timeout time.Duration `json:"timeout" yaml:"timeout" comment:"Timeout"`
}

// KubeRunPodConfig is the legacy pod configuration structure for the "kuberun" backend.
// Deprecated: Use PodConfig instead.
//goland:noinspection GoDeprecation
type KubeRunPodConfig struct {
	// Namespace is the namespace to run the pod in.
	Namespace string `json:"namespace" yaml:"namespace" comment:"Namespace to run the pod in" default:"default"`
	// ConsoleContainerNumber specifies the container to attach the running process to. Defaults to 0.
	ConsoleContainerNumber int `json:"consoleContainerNumber" yaml:"consoleContainerNumber" comment:"Which container to attach the SSH connection to" default:"0"`
	// Spec contains the pod specification to launch.
	Spec v1.PodSpec `json:"podSpec" yaml:"podSpec" comment:"Pod specification to launch" default:"{\"containers\":[{\"name\":\"shell\",\"image\":\"containerssh/containerssh-guest-image\"}]}"`
	// Subsystems contains a map of subsystem names and the executable to launch.
	Subsystems map[string]string `json:"subsystems" yaml:"subsystems" comment:"Subsystem names and binaries map." default:"{\"sftp\":\"/usr/lib/openssh/sftp-server\"}"`

	// DisableCommand is a configuration option to support legacy command disabling from the kuberun config.
	// See https://containerssh.io/deprecations/kuberun for details.
	DisableCommand bool `json:"disableCommand" yaml:"disableCommand" comment:"DisableCommand is a configuration option to support legacy command disabling from the kuberun config."`
}

// Validate validates the KubeRunConfig
//goland:noinspection GoDeprecation
func (config KubeRunConfig) Validate() error {
	if err := config.Connection.Validate(); err != nil {
		return fmt.Errorf("invalid connection configuration (%w)", err)
	}
	if err := config.Pod.Validate(); err != nil {
		return fmt.Errorf("invalid pod configuration (%w)", err)
	}
	return nil
}

// Validate validates the KubeRunPodConfig
//goland:noinspection GoDeprecation
func (config KubeRunPodConfig) Validate() error {
	if config.Namespace == "" {
		return fmt.Errorf("no namespace provided")
	}
	if len(config.Spec.Containers) == 0 {
		return fmt.Errorf("invalid pod spec: no containers provided")
	}
	for container, spec := range config.Spec.Containers {
		if spec.Image == "" {
			return fmt.Errorf("invalid pod spec: empty image name provided for container %d", container)
		}
	}
	if len(config.Spec.Containers) <= config.ConsoleContainerNumber+1 {
		return fmt.Errorf("invalid console container number %d", config.ConsoleContainerNumber)
	}
	return nil
}
