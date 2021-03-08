# Message / error codes

| Code | Explanation |
|------|-------------|
| `KUBERNETES_AGENT_READ_FAILED` | The ContainerSSH Kubernetes module failed to read from the ContainerSSH agent. This is most likely because the ContainerSSH guest agent is not present in the guest image, but agent support is enabled. |
| `KUBERNETES_CLOSE_INPUT_FAILED` | The ContainerSSH Kubernetes module attempted to close the input (stdin) for reading but failed to do so. |
| `KUBERNETES_CLOSE_OUTPUT_FAILED` | The ContainerSSH Kubernetes module attempted to close the output (stdout and stderr) for writing but failed to do so. |
| `KUBERNETES_CONFIG_ERROR` | The ContainerSSH Kubernetes module detected a configuration error. Please check your configuration. |
| `KUBERNETES_EXEC` | The ContainerSSH Kubernetes module is creating an execution. This may be in connection mode, or it may be the module internally using the exec mechanism to deliver a payload into the pod. |
| `KUBERNETES_EXEC_ATTACH` | The ContainerSSH Kubernetes module is attaching to the previously-created execution. |
| `KUBERNETES_EXEC_ATTACH_FAILED` | The ContainerSSH Kubernetes module could not attach to the previously-created execution. |
| `KUBERNETES_EXEC_CREATE` | The ContainerSSH Kubernetes module is creating an execution. |
| `KUBERNETES_EXEC_CREATE_FAILED` | The ContainerSSH Kubernetes module has failed to create an execution. This can be temporary and retried or permanent. See the error message for details. |
| `KUBERNETES_EXEC_PID_READ_FAILED` | The ContainerSSH Kubernetes module has failed to read the process ID from the ContainerSSH guest agent. This is most likely because the guest image does not contain the guest agent, but guest agent support has been enabled. |
| `KUBERNETES_EXEC_RESIZE` | The ContainerSSH Kubernetes module is resizing the console. |
| `KUBERNETES_EXEC_RESIZE_FAILED` | The ContainerSSH Kubernetes module failed to resize the console. |
| `KUBERNETES_EXEC_SIGNAL` | The ContainerSSH Kubernetes module is delivering a signal in pod mode. |
| `KUBERNETES_EXEC_SIGNAL_FAILED` | The ContainerSSH Kubernetes module failed to deliver a signal. |
| `KUBERNETES_EXEC_SIGNAL_FAILED_NO_AGENT` | The ContainerSSH Kubernetes module failed to deliver a signal because guest agent support is disabled. |
| `KUBERNETES_EXEC_SIGNAL_SUCCESSFUL` | The ContainerSSH Kubernetes module successfully delivered the requested signal. |
| `KUBERNETES_EXIT_CODE` | The ContainerSSH Kubernetes module is fetching the exit code from the program. |
| `KUBERNETES_EXIT_CODE_FAILED` | The ContainerSSH Kubernetes module has failed to fetch the exit code of the program. |
| `KUBERNETES_EXIT_CODE_NEGATIVE` | The ContainerSSH Kubernetes module has received a negative exit code from Kubernetes. This should never happen and is most likely a bug. |
| `KUBERNETES_EXIT_CODE_POD_RESTARTING` | The ContainerSSH Kubernetes module could not fetch the exit code from the program because the pod is restarting. This is typically a misconfiguration as ContainerSSH pods should not automatically restart. |
| `KUBERNETES_EXIT_CODE_STILL_RUNNING` | The ContainerSSH Kubernetes module could not fetch the program exit code because the program is still running. This error may be temporary and retried or permanent. |
| `KUBERNETES_GUEST_AGENT_DISABLED` | The [ContainerSSH Guest Agent](https://github.com/podssh/agent) has been disabled, which is strongly discouraged. ContainerSSH requires the guest agent to be installed in the pod image to facilitate all SSH features. Disabling the guest agent will result in breaking the expectations a user has towards an SSH server. We provide the ability to disable guest agent support only for cases where the guest agent binary cannot be installed in the image at all. |
| `KUBERNETES_PID_RECEIVED` | The ContainerSSH Kubernetes module has received a PID from the Kubernetes guest agent. |
| `KUBERNETES_POD_ATTACH` | The ContainerSSH Kubernetes module is attaching to a pod in session mode. |
| `KUBERNETES_POD_CREATE` | The ContainerSSH Kubernetes module is creating a pod. |
| `KUBERNETES_POD_CREATE_FAILED` | The ContainerSSH Kubernetes module failed to create a pod. This may be a temporary and retried or a permanent error message. Check the log message for details. |
| `KUBERNETES_POD_REMOVE` | The ContainerSSH Kubernetes module is removing a pod. |
| `KUBERNETES_POD_REMOVE_FAILED` | The ContainerSSH Kubernetes module could not remove the pod. This message may be temporary and retried or permanent. Check the log message for details. |
| `KUBERNETES_POD_REMOVE_SUCCESSFUL` | The ContainerSSH Kubernetes module has successfully removed the pod. |
| `KUBERNETES_POD_SHUTTING_DOWN` | The ContainerSSH Kubernetes module is shutting down a pod. |
| `KUBERNETES_POD_SIGNAL` | The ContainerSSH Kubernetes module is sending a signal to the pod. |
| `KUBERNETES_POD_SIGNAL_FAILED` | The ContainerSSH Kubernetes module has failed to send a signal to the pod. |
| `KUBERNETES_POD_WAIT` | The ContainerSSH Kubernetes module is waiting for the pod to come up. |
| `KUBERNETES_POD_WAIT_FAILED` | The ContainerSSH Kubernetes module failed to wait for the pod to come up. Check the error message for details. |
| `KUBERNETES_PROGRAM_ALREADY_RUNNING` | The ContainerSSH Kubernetes module can't execute the request because the program is already running. This is a client error. |
| `KUBERNETES_SIGNAL_FAILED_NO_PID` | The ContainerSSH Kubernetes module can't deliver a signal because no PID has been recorded. This is most likely because guest agent support is disabled. |
| `KUBERNETES_STREAM_INPUT_FAILED` | The ContainerSSH Kubernetes module failed to stream stdin to the Kubernetes engine. |
| `KUBERNETES_STREAM_OUTPUT_FAILED` | The ContainerSSH Kubernetes module failed to stream stdout and stderr from the Kubernetes engine. |
| `KUBERNETES_SUBSYSTEM_NOT_SUPPORTED` | The ContainerSSH Kubernetes module is not configured to run the requested subsystem. |

