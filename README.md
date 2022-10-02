[![ContainerSSH - Launch Containers on Demand](https://containerssh.github.io/images/logo-for-embedding.svg)](https://containerssh.github.io/)

<!--suppress HtmlDeprecatedAttribute -->
<h1 align="center">ContainerSSH Kubernetes Library</h1>

<p align="center"><strong>⚠⚠⚠ Deprecated: ⚠⚠⚠</strong><br />This repository is deprecated in favor of <a href="https://github.com/ContainerSSH/libcontainerssh">libcontainerssh</a> for ContainerSSH 0.5.</p>


This library runs Kubernetes pods in integration with the [sshserver library](https://github.com/containerssh/sshserver).

## How this library works

When a client successfully performs an SSH handshake this library creates a Pod in the specified Kubernetes cluster. This pod will run the command specified in `IdleCommand`. When the user opens a session channel this library runs an `exec` command against this container, allowing multiple parallel session channels to work on the same Pod.

## Using this library

As this library is designed to be used exclusively with the [sshserver library](https://github.com/containerssh/sshserver) the API to use it is also very closely aligned. This backend doesn't implement a full SSH backend, instead it implements a network connection handler. This handler can be instantiated using the `kuberun.New()` method:

```go
handler, err := kuberun.New(
    client,
    connectionID,
    config,
    logger,
    backendRequestsCounter,
    backendFailuresCounter,
)
```

The parameters are as follows:

- `config` is a struct of the [`kuberun.Config` type](config.go).
- `connectionID` is an opaque ID for the connection.
- `client` is the `net.TCPAddr` of the client that connected.
- `logger` is the logger from the [log library](https://github.com/containerssh/log)
- `backendRequestsCounter` and `backendFailuresCounter` are counters from the [metrics library](https://github.com/containerssh/metrics)

Once the handler is created it will wait for a successful handshake:

```go
sshConnection, err := handler.OnHandshakeSuccess("username-here")
```

This will launch a pod. Conversely, the `handler.OnDisconnect()` will destroy the pod.

The `sshConnection` can be used to create session channels and launch programs as described in the [sshserver library](https://github.com/containerssh/sshserver).

**Note:** This library does not perform authentication. Instead, it will always `sshserver.AuthResponseUnavailable`.
