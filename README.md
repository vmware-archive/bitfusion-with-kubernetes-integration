# Bitfusion with Kubernetes Integration

This project is a collection of tools for Bitfusion to be used with Kubernetes and Docker.
It contains:<br/>
1.[Bitfusion device plugin for Kubernetes](./bitfusion_device_plugin) (For Kubernetes users).<br/>
2.[Dockerfile templates for Bitfusion client](./Dockerfiles) (For Docker users).<br/>
3.[Shell scripts to accelerate the deployment](./nvidia-env.sh) (For Docker users).<br/>

## Features
* Bitfusion device plugin provides the interface for applications to request
Bitfusion GPU resources via Kubernetes native mechanism. Also, it could
help to bake Bitfusion client image automatically.
* Dockerfile templates show Bitfusion client docker image file samples, which
could help to build customized Bitfusion application docker images.
* Other. Scripts are provided to build Bitfusion execution environment
automatically.

## Usage
* For device plugin, refer to [this page](./bitfusion_device_plugin/Readme.md)
for more usage information
* Dockerfile templaes are located in the directory ./Dockerfiles
* nvidia-env.sh is used to setup Bitfusion client

## Feedback
Feel free to send us feedback by [filing an issue](https://github.com/vmware/bitfusion-with-kubernetes-integration/issues/new). Feature requests are always
welcome. If you wish to contribute, please take a quick look at the next section.

## How to Contribute
This project team welcomes contributions from the community. If you wish to
contribute code and you have not signed our contributor license agreement (CLA),
our bot will update the issue when you open a Pull Request. For any questions
about the CLA process, please refer to our [FAQ](https://cla.vmware.com/faq).

* Clone this repository and create a new branch
* Make changes and test
* Submit Pull Request with comprehensive description of changes

## License
The project is licensed under the terms of the Apache 2.0 license.
