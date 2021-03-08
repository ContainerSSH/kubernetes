# Message / error codes

| Code | Explanation |
|------|-------------|
| `KUBERNETES_CLOSE_OUTPUT_FAILED` | The ContainerSSH Kubernetes module attempted to close the output (stdout and stderr) for writing but failed to do so. |
| `KUBERNETES_CONFIG_ERROR` | The ContainerSSH Kubernetes module detected a configuration error. Please check your configuration. |
| `KUBERNETES_EXEC` | The ContainerSSH Kubernetes module is creating an execution. This may be in connection mode, or it may be the module internally using the exec mechanism to deliver a payload into the pod. |
| `KUBERNETES_EXEC_RESIZE` | The ContainerSSH Kubernetes module is resizing the terminal window. |
| `KUBERNETES_EXEC_RESIZE_FAILED` | The ContainerSSH Kubernetes module failed to resize the console. |
| `KUBERNETES_EXEC_SIGNAL` | The ContainerSSH Kubernetes module is delivering a signal. |
| `KUBERNETES_EXEC_SIGNAL_FAILED` | The ContainerSSH Kubernetes module failed to deliver a signal. |
| `KUBERNETES_EXEC_SIGNAL_FAILED_NO_AGENT` | The ContainerSSH Kubernetes module failed to deliver a signal because guest agent support is disabled. |
| `KUBERNETES_EXEC_SIGNAL_SUCCESSFUL` | The ContainerSSH Kubernetes module successfully delivered the requested signal. |
| `KUBERNETES_EXIT_CODE_FAILED` | The ContainerSSH Kubernetes module has failed to fetch the exit code of the program. |
| `KUBERNETES_GUEST_AGENT_DISABLED` | The [ContainerSSH Guest Agent](https://github.com/podssh/agent) has been disabled, which is strongly discouraged. ContainerSSH requires the guest agent to be installed in the pod image to facilitate all SSH features. Disabling the guest agent will result in breaking the expectations a user has towards an SSH server. We provide the ability to disable guest agent support only for cases where the guest agent binary cannot be installed in the image at all. |
| `KUBERNETES_PID_RECEIVED` | The ContainerSSH Kubernetes module has received a PID from the Kubernetes guest agent. |
| `KUBERNETES_POD_ATTACH` | The ContainerSSH Kubernetes module is attaching to a pod in session mode. |
| `KUBERNETES_POD_CREATE` | The ContainerSSH Kubernetes module is creating a pod. |
| `KUBERNETES_POD_CREATE_FAILED` | The ContainerSSH Kubernetes module failed to create a pod. This may be a temporary and retried or a permanent error message. Check the log message for details. |
| `KUBERNETES_POD_REMOVE` | The ContainerSSH Kubernetes module is removing a pod. |
| `KUBERNETES_POD_REMOVE_FAILED` | The ContainerSSH Kubernetes module could not remove the pod. This message may be temporary and retried or permanent. Check the log message for details. |
| `KUBERNETES_POD_REMOVE_SUCCESSFUL` | The ContainerSSH Kubernetes module has successfully removed the pod. |
| `KUBERNETES_POD_SHUTTING_DOWN` | The ContainerSSH Kubernetes module is shutting down a pod. |
| `KUBERNETES_POD_WAIT` | The ContainerSSH Kubernetes module is waiting for the pod to come up. |
| `KUBERNETES_POD_WAIT_FAILED` | The ContainerSSH Kubernetes module failed to wait for the pod to come up. Check the error message for details. |
| `KUBERNETES_PROGRAM_ALREADY_RUNNING` | The ContainerSSH Kubernetes module can't execute the request because the program is already running. This is a client error. |
| `KUBERNETES_PROGRAM_NOT_RUNNING` | This message indicates that the user requested an action that can only be performed when a program is running, but there is currently no program running. |
| `KUBERNETES_SIGNAL_FAILED_EXITED` | The ContainerSSH Kubernetes module can't deliver a signal because the program already exited. |
| `KUBERNETES_SIGNAL_FAILED_NO_PID` | The ContainerSSH Kubernetes module can't deliver a signal because no PID has been recorded. This is most likely because guest agent support is disabled. |
| `KUBERNETES_SUBSYSTEM_NOT_SUPPORTED` | The ContainerSSH Kubernetes module is not configured to run the requested subsystem. |
| `KUBERUN_DEPRECATED` | This message indicates that you are still using the deprecated KubeRun backend. This backend doesn't support all safety and functionality improvements and will be removed in the future. Please read the [deprecation notice for a migration guide](https://containerssh.io/deprecations/kuberun) |
| `KUBERUN_EXEC_DISABLED` | This message indicates that the user tried to execute a program, but program execution is disabled in the legacy KubeRun configuration. |
| `KUBERUN_INSECURE` | This message indicates that you are using Kubernetes in the "insecure" mode where certificate verification is disabled. This is a major security flaw, has been deprecated and is removed in the new Kubernetes backend. Please change your configuration to properly validates the server certificates. |

