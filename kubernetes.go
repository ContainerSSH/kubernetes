package kubernetes

import (
	"context"
	"io"

	"github.com/containerssh/log"
)

// kubernetesClientFactory creates a kubernetesClient based on a configuration
type kubernetesClientFactory interface {
	// get takes a configuration and returns a kubernetes client if the configuration was populated.
	// Returns an error if the configuration is invalid. Returns errkubernetesClientNotConfigured if the specific client is
	// not configured
	get(ctx context.Context, config Config, logger log.Logger) (kubernetesClient, error)
}

// kubernetesClient is a simplified representation of a kubernetes client.
type kubernetesClient interface {
	// createPod creates and starts the configured Pod. May return a Pod even if an error happened.
	// This pod will need to be removed. Passing tty also means that the main console will be prepared for
	// attaching.
	createPod(
		ctx context.Context,
		labels map[string]string,
		env map[string]string,
		tty *bool,
		cmd []string,
	) (kubernetesPod, error)
}

// kubernetesPod is the representation of a created Pod.
type kubernetesPod interface {
	// attach attaches to the Pod on the main console.
	attach(ctx context.Context) (kubernetesExecution, error)

	// createExec creates an execution process for the given program with the given parameters. The passed context is
	// the start context.
	createExec(ctx context.Context, program []string, env map[string]string, tty bool) (kubernetesExecution, error)

	// remove removes the Pod within the given context.
	remove(ctx context.Context) error
}

// kubernetesExecution is an execution process on either an "exec" process or attached to the main console of a Pod.
type kubernetesExecution interface {
	// resize resizes the current terminal to the given dimensions.
	resize(ctx context.Context, height uint, width uint) error
	// signal sends the given signal to the currently running process. Returns an error if the process is not running,
	// the signal is not known or permitted, or the process ID is not known.
	signal(ctx context.Context, sig string) error
	// run runs the process in question.
	run(
		stdout io.Writer,
		stderr io.Writer,
		stdin io.Reader,
		onExit func(exitStatus int),
	)
}
