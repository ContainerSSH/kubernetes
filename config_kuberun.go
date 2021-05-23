package kubernetes

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	k8sYaml "sigs.k8s.io/yaml"
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

	// Insecure means that the server should be accessed without TLS verification. This is NOT recommended.
	Insecure bool `json:"insecure" yaml:"insecure" comment:"Server should be accessed without verifying the TLS certificate." default:"false"`

	// Timeout indicates the timeout for client calls.
	Timeout time.Duration `json:"timeout" yaml:"timeout" comment:"Timeout"`
}

// KubeRunPodConfig is the legacy pod configuration structure for the "kuberun" backend.
// Deprecated: Use PodConfig instead.
//goland:noinspection GoDeprecation
type KubeRunPodConfig struct {
	// Namespace is the namespace to run the pod in.
	Namespace string `json:"namespace,omitempty" yaml:"namespace" comment:"Namespace to run the pod in" default:"default"`
	// ConsoleContainerNumber specifies the container to attach the running process to. Defaults to 0.
	ConsoleContainerNumber int `json:"consoleContainerNumber,omitempty" yaml:"consoleContainerNumber" comment:"Which container to attach the SSH connection to" default:"0"`
	// Spec contains the pod specification to launch.
	Spec v1.PodSpec `json:"podSpec,omitempty" yaml:"podSpec" comment:"Pod specification to launch" default:"{\"containers\":[{\"name\":\"shell\",\"image\":\"containerssh/containerssh-guest-image\"}]}"`
	// Subsystems contains a map of subsystem names and the executable to launch.
	Subsystems map[string]string `json:"subsystems,omitempty" yaml:"subsystems" comment:"Subsystem names and binaries map." default:"{\"sftp\":\"/usr/lib/openssh/sftp-server\"}"`

	// AgentPath contains the path to the ContainerSSH Guest Agent.
	AgentPath string `json:"agentPath,omitempty" yaml:"agentPath" default:"/usr/bin/containerssh-agent"`
	// EnableAgent enables using the ContainerSSH Guest Agent.
	EnableAgent bool `json:"disableAgent,omitempty" yaml:"disableAgent"`
	// ShellCommand is the command used for launching shells. This is required when using the ContainerSSH agent.
	ShellCommand []string `json:"shellCommand,omitempty" yaml:"shellCommand" comment:"Run this command as a default shell."`

	// DisableCommand is a configuration option to support legacy command disabling from the kuberun config.
	// See https://containerssh.io/deprecations/kuberun for details.
	DisableCommand bool `json:"disableCommand,omitempty" yaml:"disableCommand" comment:"DisableCommand is a configuration option to support legacy command disabling from the kuberun config."`
}

// MarshalYAML uses the Kubernetes YAML library to encode the PodConfig instead of the default configuration.
//goland:noinspection GoDeprecation
func (c *KubeRunPodConfig) MarshalYAML() (interface{}, error) {
	data, err := k8sYaml.Marshal(c)
	if err != nil {
		return nil, err
	}
	node := map[string]yaml.Node{}
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, err
	}
	return node, nil
}

// UnmarshalYAML uses the Kubernetes YAML library to encode the PodConfig instead of the default configuration.
//goland:noinspection GoDeprecation
func (c *KubeRunPodConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	node := map[string]yaml.Node{}
	if err := unmarshal(&node); err != nil {
		return err
	}
	data, err := yaml.Marshal(node)
	if err != nil {
		return err
	}
	if err := k8sYaml.Unmarshal(data, c); err != nil {
		return err
	}
	return nil
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
func (c KubeRunPodConfig) Validate() error {
	if c.Namespace == "" {
		return fmt.Errorf("no namespace provided")
	}
	if len(c.Spec.Containers) == 0 {
		return fmt.Errorf("invalid pod spec: no containers provided")
	}
	for container, spec := range c.Spec.Containers {
		if spec.Image == "" {
			return fmt.Errorf("invalid pod spec: empty image name provided for container %d", container)
		}
	}
	if len(c.Spec.Containers) < c.ConsoleContainerNumber+1 {
		return fmt.Errorf("invalid console container number %d", c.ConsoleContainerNumber)
	}
	return nil
}
