# Changelog

## 2.0.1: Fixed bug in previous release

This release fixes the YAML marshalling introduced in the previous release.

## 2.0.0: Fixing Kubernetes data structure encoding/decoding

This release changes the encoding/decoding process to use the Kubernetes YAML library. This is done because the previously used library would not work with several Kubernetes embedded structures. This change is backwards incompatible as it requires `gopkg.in/yaml.v3` instead `gopkg.in/yaml.v2`.

## 1.0.0: First stable release

This release tags the first stable version for ContainerSSH 0.4.0.

## 0.9.9: Moving to ContainerSSH agent 

This release is moving to using the ContainerSSH agent as the init process.

## 0.9.8: Better JSON and YAML support

Explicitly disabled internal fields in JSON or YAML.

## 0.9.7: Better logging, proper shutdown handling

This release adds better log messages for administrators as well as proper shutdown handling when a network connection breaks without closing the channels first.

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
