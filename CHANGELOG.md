# Changelog

## 0.9.2: Added metrics

This release adds integration with the [metrics library](https://github.com/containerssh/metrics). This changes the function signature of the `New` and `NewKubeRun` methods to include the two metrics.

## 0.9.1: Restored 0.3 compatibility

In this release we are restoring compatibility with ContainerSSH 0.3. The previous release caused an error when unmarshaling a YAML config from ContainerSSH 0.3.

## 0.9.0: Split from KubeRun

This version is the first release of the more advanced `kubernetes` backend split from the `kuberun` backend. This backend introduces the ability to run several commands using the `exec` mechanism and adds support for the [ContainerSSH Guest Agent](https://github.com/containerssh/agent).

For more details see the [kuberun deprecation notice](https://containerssh.io/deprecations/kuberun)