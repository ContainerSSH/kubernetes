package kubernetes

import (
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Config is the base configuration structure for kuberun
type Config struct {
	// Connection configures the connection to the Kubernetes cluster.
	Connection ConnectionConfig `json:"connection" yaml:"connection" comment:"Kubernetes configuration options"`
	// Pod contains the spec and specific settings for creating the pod.
	Pod PodConfig `json:"pod" yaml:"pod" comment:"Container configuration"`
	// Timeout specifies how long to wait for the Pod to come up.
	Timeouts TimeoutConfig `json:"timeouts" yaml:"timeouts" comment:"Timeout for pod creation" default:"60s"`
}

// ConnectionConfig configures the connection to the Kubernetes cluster.
type ConnectionConfig struct {
	// Host is a host string, a host:port pair, or a URL to the Kubernetes apiserver. Defaults to kubernetes.default.svc.
	Host string `json:"host" yaml:"host" comment:"a host string, a host:port pair, or a URL to the base of the apiserver." default:"kubernetes.default.svc"`
	// APIPath is a sub-path that points to the API root. Defaults to /api
	APIPath string `json:"path" yaml:"path" comment:"APIPath is a sub-path that points to an API root." default:"/api"`

	// Username is the username for basic authentication.
	Username string `json:"username" yaml:"username" comment:"Username for basic authentication"`
	// Password is the password for basic authentication.
	Password string `json:"password" yaml:"password" comment:"Password for basic authentication"`

	// Insecure means that the server should be accessed without TLS verification. This is NOT recommended.
	Insecure bool `json:"insecure" yaml:"insecure" comment:"Server should be accessed without verifying the TLS certificate." default:"false"`
	// ServerName sets the server name to be set in the SNI and used by the client for TLS verification.
	ServerName string `json:"serverName" yaml:"serverName" comment:"ServerName is passed to the server for SNI and is used in the client to check server certificates against."`

	// CertFile points to a file that contains the client certificate used for authentication.
	CertFile string `json:"certFile" yaml:"certFile" comment:"File containing client certificate for TLS client certificate authentication."`
	// KeyFile points to a file that contains the client key used for authentication.
	KeyFile string `json:"keyFile" yaml:"keyFile" comment:"File containing client key for TLS client certificate authentication"`
	// CAFile points to a file that contains the CA certificate for authentication.
	CAFile string `json:"cacertFile" yaml:"cacertFile" comment:"File containing trusted root certificates for the server"`

	// CertData contains a PEM-encoded certificate for TLS client certificate authentication.
	CertData string `json:"cert" yaml:"cert" comment:"PEM-encoded certificate for TLS client certificate authentication"`
	// KeyData contains a PEM-encoded client key for TLS client certificate authentication.
	KeyData string `json:"key" yaml:"key" comment:"PEM-encoded client key for TLS client certificate authentication"`
	// CAData contains a PEM-encoded trusted root certificates for the server.
	CAData string `json:"cacert" yaml:"cacert" comment:"PEM-encoded trusted root certificates for the server"`

	// BearerToken contains a bearer (service) token for authentication.
	BearerToken string `json:"bearerToken" yaml:"bearerToken" comment:"Bearer (service token) authentication"`
	// BearerTokenFile points to a file containing a bearer (service) token for authentication.
	// Set to /var/run/secrets/kubernetes.io/serviceaccount/token to use service token in a Kubernetes kubeConfigCluster.
	BearerTokenFile string `json:"bearerTokenFile" yaml:"bearerTokenFile" comment:"Path to a file containing a BearerToken. Set to /var/run/secrets/kubernetes.io/serviceaccount/token to use service token in a Kubernetes kubeConfigCluster."`

	// QPS indicates the maximum QPS to the master from this client. Defaults to 5.
	QPS float32 `json:"qps" yaml:"qps" comment:"QPS indicates the maximum QPS to the master from this client." default:"5"`
	// Burst indicates the maximum burst for throttle.
	Burst int `json:"burst" yaml:"burst" comment:"Maximum burst for throttle." default:"10"`
	// Timeout indicates the timeout for client calls.
	Timeout time.Duration `json:"timeout" yaml:"timeout" comment:"Timeout"`
}

// PodConfig describes the pod to launch.
type PodConfig struct {
	Metadata metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty" default:"{\"namespace\":\"default\",\"generator\":\"containerssh-\"}"`
	// Spec contains the pod specification to launch.
	Spec v1.PodSpec `json:"podSpec" yaml:"podSpec" comment:"Pod specification to launch" default:"{\"containers\":[{\"name\":\"shell\",\"image\":\"containerssh/containerssh-guest-image\"}]}"`

	// ConsoleContainerNumber specifies the container to attach the running process to. Defaults to 0.
	ConsoleContainerNumber int `json:"consoleContainerNumber" yaml:"consoleContainerNumber" comment:"Which container to attach the SSH connection to" default:"0"`
	// Subsystems contains a map of subsystem names and the executable to launch.
	Subsystems map[string]string `json:"subsystems" yaml:"subsystems" comment:"Subsystem names and binaries map." default:"{\"sftp\":\"/usr/lib/openssh/sftp-server\"}"`

	// IdleCommand contains the command to run as the first process in the container. Other commands are executed using the "exec" method.
	IdleCommand []string `json:"idleCommand" yaml:"idleCommand" comment:"Run this command to wait for container exit" default:"[\"/bin/sh\", \"-c\", \"sleep infinity & PID=$!; trap \\\"kill $PID\\\" INT TERM; wait\"]"`
	// ShellCommand is the command used for launching shells when the container is in ExecutionModeConnection. Ignored in ExecutionModeSession.
	ShellCommand []string `json:"shellCommand" yaml:"shellCommand" comment:"Run this command as a default shell." default:"[\"/bin/bash\"]"`

	// Mode influences how commands are executed.
	//
	// - If ExecutionModeConnection is chosen (default) a new pod is launched per connection. In this mode
	//   sessions are executed using the "docker exec" functionality and the main container console runs a script that
	//   waits for a termination signal.
	// - If ExecutionModeSession is chosen a new pod is launched per session, leading to potentially multiple
	//   pods per connection. In this mode the program is launched directly as the main process of the container.
	//   When configuring this mode you should explicitly configure the "cmd" option to an empty list if you want the
	//   default command in the container to launch.
	Mode ExecutionMode `json:"mode" yaml:"mode" default:"connection"`

	// disableCommand is a configuration option to support legacy command disabling from the kuberun config.
	// See https://containerssh.io/deprecations/kuberun for details.
	disableCommand bool
}

type TimeoutConfig struct {
	// HTTP configures the timeout for HTTP calls
	HTTP time.Duration `json:"http" yaml:"http" default:"15s"`
	// PodStart is the timeout for creating and starting the pod.
	PodStart time.Duration `json:"podStart" yaml:"podStart" default:"60s"`
	// PodStop is the timeout for stopping and removing the pod.
	PodStop time.Duration `json:"podStop" yaml:"podStop" default:"60s"`
	// CommandStart sets the maximum time starting a command may take.
	CommandStart time.Duration `json:"commandStart" yaml:"commandStart" default:"60s"`
}

// ExecutionMode determines when a container is launched.
// ExecutionModeConnection launches one container per SSH connection (default), while ExecutionModeSession launches
// one container per SSH session.
type ExecutionMode string

const (
	// ExecutionModeConnection launches one container per SSH connection.
	ExecutionModeConnection ExecutionMode = "connection"
	// ExecutionModeSession launches one container per SSH session (multiple containers per connection).
	ExecutionModeSession ExecutionMode = "session"
)

// Validate validates the execution config.
func (e ExecutionMode) Validate() error {
	switch e {
	case ExecutionModeConnection:
		fallthrough
	case ExecutionModeSession:
		return nil
	default:
		return fmt.Errorf("invalid execution mode: %s", e)
	}
}
