# Changelog

## 0.9.6: Conformance tests

This release makes use of the comprehensive conformance test introduced in [sshserver](https://github.com/containerssh/sshserver) and fixes a number of issues found in the process.

## 0.9.5: Regression fixes; Added `SetConfigFromKubeConfig` method to the `kuberun` configuration structure

This release fixes a regression where session mode pods would not be handled correctly.

Also, we added a `SetConfigFromKubeConfig()` method to the `kubernetes.KubeRunConfig` struct to make it easer to read the current kubeconfig. It is considered experimental and for testing purposes only.

## 0.9.4: Added `SetConfigFromKubeConfig` method to configuration structure

Added a `SetConfigFromKubeConfig()` method to the `kubernetes.Config` struct to make it easer to read the current kubeconfig. It is considered experimental and for testing purposes only.

## 0.9.3: Added Validate to KubeRun

This release adds a `Validate()` method to the KubeRun configuration.

## 0.9.2: Added metrics

This release adds integration with the [metrics library](https://github.com/containerssh/metrics). This changes the function signature of the `New` and `NewKubeRun` methods to include the two metrics.

## 0.9.1: Restored 0.3 compatibility

In this release we are restoring compatibility with ContainerSSH 0.3. The previous release caused an error when unmarshaling a YAML config from ContainerSSH 0.3.

## 0.9.0: Split from KubeRun

This version is the first release of the more advanced `kubernetes` backend split from the `kuberun` backend. This backend introduces the ability to run several commands using the `exec` mechanism and adds support for the [ContainerSSH Guest Agent](https://github.com/containerssh/agent).

For more details see the [kuberun deprecation notice](https://containerssh.io/deprecations/kuberun).
