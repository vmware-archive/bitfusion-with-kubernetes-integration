# Bitfusion with Kubernetes Integration

- bitfusion-with-kubernetes-integration is not required to run Bitfusion with Kubernetes.
- [Kubernetes integration is included with Bitfusion](https://docs.vmware.com/en/VMware-vSphere-Bitfusion/4.0/Install-Guide/GUID-3E0A4340-8EC0-4DE0-B467-8714725DF901.html#GUID-E65BA25B-3C25-41F3-B901-28B4CE30812D__GUID-8053F952-74C2-4F11-99FE-B78F0EE6B262).
- bitfusion-with-kubernetes-integration is community supported and is not supported by VMware Tech Support. If you have a problem [open an issue on Github](https://github.com/vmware/bitfusion-with-kubernetes-integration/issues).

This project is a collection of tools for Bitfusion to be used with Kubernetes and Docker. For more information about Bitfusion, visit [vSphere Bitfusion document page](https://docs.vmware.com/en/VMware-vSphere-Bitfusion/index.html). This project contains:

- [Bitfusion device plugin for Kubernetes](./bitfusion_device_plugin)
- [Dockerfile templates for Bitfusion client](./Dockerfiles)
- [Shell scripts to accelerate the deployment](./nvidia-env.sh)

## Features

- [Bitfusion device plugin](./bitfusion_device_plugin) provides the interface for applications to request Bitfusion GPU resources via Kubernetes native mechanism. Also, it could help to bake Bitfusion client image automatically.
- [Dockerfile templates](./Dockerfiles) show Bitfusion client Docker image file samples, which could help to build customized Bitfusion application Docker images.
- [Shell script](./nvidia-env.sh) is provided to build Bitfusion execution environment automatically.

## Usage

- For device plugin, refer to [this page](./bitfusion_device_plugin/README.md) for more usage information.
- Dockerfile templates are located in the directory [./Dockerfiles](./Dockerfiles) to set up a Bitfusion client based on different OS automatically. After deployment, you could run `bitfusion smi` command to check remote GPU resource of Bitfusion server connected to this Bitfusion client.
- [nvidia-env.sh](./nvidia-env.sh) is a shell script to install needed various dependencies of Bitfusion client.

## Feedback

Feel free to send us feedback by [filing an issue](https://github.com/vmware/bitfusion-with-kubernetes-integration/issues/new). Feature requests are always welcome. If you wish to contribute, please take a quick look at the next section.

## How to Contribute

This project team welcomes contributions from the community. If you wish to contribute code and you have not signed our contributor license agreement (CLA), our bot will update the issue when you open a Pull Request. For any questions about the CLA process, please refer to our [FAQ](https://cla.vmware.com/faq).

- Clone this repository and create a new branch.
- Make changes and test.
- Submit Pull Request with comprehensive description of changes.

