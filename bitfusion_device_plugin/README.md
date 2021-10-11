# Bitfusion on Kubernetes ##


Current solutions of GPU virtualization may have some shortcomings:

1.Underutilized GPU compute cycle   
2.Limited and preset granularity    
3.Resource bound to local machine    
4.Hard for application scheduling   

Bitfusion helps address the problems by providing remote GPU pool. Bitfusion makes GPUs a first class citizen that can be abstracted, partitioned, automated and shared like compute resource. On the other hand, Kubernetes has become the de facto platform to deploy and manage machine learning workloads.

However, out of the box Kubernetes does not offer a way to consume Bitfusion's network-attached GPUs. This limitation becomes a key challenge to enable jobs on Kubernetes to use Bitfusion’s GPU. Kubernetes needs a friendly way to consume Bitfusion GPU resources for the following reasons:  
- Resource management  
- GPU pool management

To address these problems, this project allows Kubernetes to work with Bitfusion.



## 1. Architecture

![img](diagrams/architecture.png)

Bitfusion on Kubernetes consists of the following two components.  
- 1.bitfusion-device-plugin  
- 2.bitfusion-webhook

Component 1 and 2 are built into separated docker images.    
bitfusion-device-plugin runs as a DaemonSet  on each worker node where kubelet is running.  
bitfusion-webhook runs as a Deployment on the Kubernetes master node.  

## 2. Prerequisites 
-  Ubuntu Linux as the operating system of the installation machine 
-  OpenSSL needs to be installed on Ubuntu
-  Kubernetes 1.17+
-  Bitfusion 3.5 and 4.0.1 
-  kubectl and docker command are ready to use.  
-  need to specify the command field in the POD



### 2.1. Quota configuration

In order to enable quota, follow below steps to update the configuration before starting the deployment.

- Update file bitfusion-injector.yaml

```
vim bitfusion-with-kubernetes-integration/bitfusion_device_plugin/webhook/deployment/bitfusion-injector.yaml
```

- Set the value of TOTAL_GPU_MEMORY(the unit is MB by default) which means GPU memory size that Bitfusion Server managed.

```
apiVersion: apps/v1
...
          env:
            - name: TOTAL_GPU_MEMORY
              value: 16000
...

```

### 2.2. Get Baremetal Token for authorization
In order to enable Bitfusion, users must generate a **Baremetal Token** for authorization and download the related tar file to the installation machine.  
Follow these steps to get the token from the vCenter:  
Step 1. Login to vCenter  
Step 2. Click on **Bitfusion** in Plugins section  
![img](diagrams/click-bitfusion-plugin.png)  
Step 3. Select the **Tokens** tab and then  select the proper token to download   
![img](diagrams/click-tokens-tag.png)   
Step 4. Click **DOWNLOAD**  button, make sure the token is **Enabled**.  
![img](diagrams/click-download-tag.png)   
If no tokens are available in the list, click on **NEW TOKEN** to create a Token.  
For more details, please refer to:   
<https://docs.vmware.com/en/VMware-vSphere-Bitfusion/2.5/Install-Guide/GUID-361A9C59-BB22-4BF0-9B05-8E80DE70BE5B.html>


### 2.3. Create a Kubernetes Secret  using the Baremetal Token

Upload the Baremetal Tokens files to the installation machine. Use the following command to unzip the files(**Noote:** the filename of the tar file may be different from the `./2BgkZdN.tar`, please change to your filename):

```shell
$ mkdir tokens    
$ tar -xvf ./2BgkZdN.tar -C tokens
```
Now we have three files in the tokens/  directory: ca.crt, client.yaml and services.conf :

```   
tokens  
├── ca.crt  
├── client.yaml  
└── servers.conf  

```

If we want to use Bitfusion client version 3.5, please update the `servers.conf` file as follows(**Noote:** the IP address in `servers.conf` may be different from yours, which means to indicate the IP address of the bitfusion server, please change to your own bitfusion server IP address):

```
# Source file content
servers:
- addresses:
  - 10.202.122.248:56001
```
Change the file above to

```
# Modified file contents
servers:
- reachable:10.202.122.248:56001
  addresses:
  - 10.202.122.248:56001
```

Then use the following command to create a secret in Kubernetes in the namespace of kube-system:  

```shell
$ kubectl create secret generic bitfusion-secret --from-file=tokens -n kube-system
```
For more details about kubectl:  <https://kubernetes.io/docs/reference/kubectl/overview/>


## 3. Quick Start

There are two deployment options:  
- Using pre-built images   
- Building images from scratch  


### 3.1. Option 1: Using pre-built images (recommended) 

Use the following command to clone the source code:  

```shell
$ git clone https://github.com/vmware/bitfusion-with-kubernetes-integration.git
```
Use the following commands to deploy the **Bitfusion device plugin** and other related components, make sure the Kubernetes cluster has Internet connection.  

```
$ cd bitfusion-with-kubernetes-integration-main/bitfusion_device_plugin
$ make deploy
```


### 3.2. Option 2: Building images from scratch 

Instead of using the pre-buit images, users can choose to build the images from source. Optionally, after the images are built, they can be pushed to a registry service (either Docker Hub or an internal registry server). 

Use the following command to clone the source code:

```shell
$ git clone https://github.com/vmware/bitfusion-with-kubernetes-integration.git
```
Modify the values of these variables in the **Makefile** before starting the build process:

```shell
$ cd bitfusion-with-kubernetes-integration-main/bitfusion_device_plugin
$ vim Makefile
```
The values of most of the variables do not need to be changed. If images are to be pushed to a registry service, make sure the vairable **IMAGE_REPO** points to the right registry service of your choice (it defaults to `docker.io/bitfusiondeviceplugin` ): 

```shell
# Variables below are the configuration of Docker images and repo for this project.
# Update these variable values with your own configuration if necessary.

IMAGE_REPO ?= docker.io/bitfusiondeviceplugin
DEVICE_IMAGE_NAME ?= bitfusion-device-plugin
WEBHOOK_IMAGE_NAME ?= bitfusion-webhook
PKG_IMAGE_NAME ?= bitfusion-client
IMAGE_TAG  ?= 0.3
```

Now start building images using the command below:

```shell
$ make build-image
```
If everything works well, use the following command to check images:

```shell
$ docker images
REPOSITORY                                                                         TAG
docker.io/bitfusiondeviceplugin/bitfusion-device-plugin                            0.3                 
docker.io/bitfusiondeviceplugin/bitfusion-webhook                                  0.3                 
docker.io/bitfusiondeviceplugin/bitfusion-client                                   0.3         

```

(Optional, but recommended) If the images need to be pushed to a registry service, use the following command to push them to the registry service.  
Use “docker login” command to log in to the registry service if necessary.([How to use docker login?](https://docs.docker.com/engine/reference/commandline/login/))

```shell
$ make push-image
```
**NOTE:** If there is no registry service avaialble, images can be exported to a file and then imported to each worker node of the Kubernetes cluster. Use docker command to save the docker images as tar files and distribute them to Kubernetes nodes manually. Then load the images from the tar files on each node. Refer to [document of docker command](https://docs.docker.com/engine/reference/commandline/save/) for more details. 

The next step is to use the following command to deploy the **Bitfusion device plugin** and other related components:

```shell
$ make deploy
```

### 3.3. Verifying the deployment  
After the installation is completed either via Option 1 or Option 2, use the following command to see if all components have been started properly in the namespace `bwki`:  

Check to see if the Device Plugin is running:

```shell
$ kubectl get pods -n kube-system

NAME                            READY   STATUS    RESTARTS   AGE
bitfusion-device-plugin-cfr87   1/1     Running   0          6m13s 
```
Check to see if the Webhook  is running:

```shell
$ kubectl  get pod -n bwki 

NAME                                            READY   STATUS    RESTARTS   AGE
bitfusion-webhook-deployment-6dbc6df664-td6t7   1/1     Running   0          7m49s 
```

Check other deployment components

```shell
$ kubectl get configmap -n bwki

NAME                                DATA   AGE
bwki-webhook-configmap              1      71m
```

```shell
$ kubectl get serviceaccount  -n bwki

NAME                           SECRETS   AGE
bitfusion-webhook-deployment   1         72m
```

```shell
$ kubectl get ValidatingWebhookConfiguration  -n bwki

NAME                          CREATED AT
validation.bitfusion.io-cfg   2021-03-25T05:29:17Z
```

```shell
$ kubectl get MutatingWebhookConfiguration   -n bwki

NAME                          CREATED AT
bwki-webhook-cfg              2021-03-25T05:29:17Z
```

```shell
$ kubectl get svc   -n bwki

NAME                          TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)   AGE
bwki-webhook-svc              ClusterIP   10.101.39.4   <none>        443/TCP   76m
```

### 3.4. Uninstall

If you decide uninstall the program and clean up all cache files, you should use the following command:
```bash
$ make uninstall
```

## 4. Using Bitfusion GPU in Kubernetes workload 

After completing the installation, users can write a YAML file of Kubernetes to consume the Bitfusion resources. There are three parameters related to Bitfusion resource in a YAML file: 


| Key        | Value    |  Describe  |
| :--------   | :-----   | :---- |
| auto-management/bitfusion | all / none / injection                  |[all] injecting a Bitfusion dependency and BareMetal Token and adding the Bitfusion prefix to the content of the Container's command, [injection] only Bitfusion dependencies and BareMetal tokens are injected and [none] do nothing to POD        |
| bitfusion.io/gpu-amount   | positive integer                        |The amount of GPU the workload request from the Bitfusion cluster|
| bitfusion.io/gpu-percent  | positive integer                        |Percentage of the memory of each GPU|
| bitfusion.io/gpu-memory   | positive integer                        |Memory size of each GPU,The default unit is bit.It can be used with the K8s native memory application unit (Mi,M,G,Gi)|
| bitfusion-client/os       | ubuntu18 / ubuntu20 / centos7 / centos8 |The OS of the containers that use the Bitfusion client|
| bitfusion-client/version  | 350/401                             |The version of Bitfusion client to be used in this container is 3.5 or 4.0.1|


Below is a sample YAML of Pod which runs a benchmark of Tensorflow. The variable `hostPath` is the directory where the Tensorflow Benchmarks code resides on the host and it will be mounted into the pod.

There are two options to specify the Bitfusion resource which the workload request.

### 4.1. Option 1: Submit the workload with "gpu-percent" parameter ###

Use bitfusion.io/gpu-amount and bitfusion.io/gpu-percent parameters in YAML file to specify the resource the pod request.

```yaml
apiVersion: v1
kind: Pod
metadata:
  annotations:
    auto-management/bitfusion: "all"
    bitfusion-client/os: "ubuntu18"
    bitfusion-client/version: "401"
  name: bf-pkgs
  # You can specify any namespace
  namespace: tensorflow-benchmark
spec:
  containers:
    - image: nvcr.io/nvidia/tensorflow:19.07-py3
      imagePullPolicy: IfNotPresent
      name: bf-pkgs
      command: ["python /benchmark/scripts/tf_cnn_benchmarks/tf_cnn_benchmarks.py --local_parameter_device=gpu --batch_size=32 --model=inception3"]
      resources:
        limits:
          # Request one GPU for this Pod from the Bitfusion cluster
          bitfusion.io/gpu-amount: 1
          # 50 percent of each GPU to be consumed
          bitfusion.io/gpu-percent: 50
      volumeMounts:
        - name: code
          mountPath: /benchmark
    volumes:
        - name: code
          # The Benchmarks used for the test came from: https://github.com/tensorflow/benchmarks/tree/tf_benchmark_stage 
          # Please make sure you have the corresponding content in /home/benchmarks directory on your node
          hostPath:
            path: /home/benchmarks
```

Then apply the yaml with the following command to deploy:

```bash
$ kubectl create namespace tensorflow-benchmark
$ kubectl create -f example/pod.yaml
```



### 4.2. Option 2: Submit the workload with "gpu-memory" parameter ###

Use bitfusion.io/gpu-amount and bitfusion.io/gpu-memory parameters in YAML file to specify the resource the pod request.


```yaml
apiVersion: v1
kind: Pod
metadata:
  annotations:
    auto-management/bitfusion: "all"
    bitfusion-client/os: "ubuntu18"
    bitfusion-client/version: "401"
  name: bf-pkgs
  # You can specify any namespace
  namespace: tensorflow-benchmark
spec:
  containers:
    - image: nvcr.io/nvidia/tensorflow:19.07-py3
      imagePullPolicy: IfNotPresent
      name: bf-pkgs
      command: ["python /benchmark/scripts/tf_cnn_benchmarks/tf_cnn_benchmarks.py --local_parameter_device=gpu --batch_size=32 --model=inception3"]
      resources:
        limits:
          bitfusion.io/gpu-amount: 1
          bitfusion.io/gpu-memory: 8000M
      volumeMounts:
        - name: code
          mountPath: /benchmark
    volumes:
        - name: code
          # The Benchmarks used for the test came from: https://github.com/tensorflow/benchmarks/tree/tf_benchmark_stage 
          # Please make sure you have the corresponding content in /home/benchmarks directory on your node
          hostPath:
            path: /home/benchmarks
```

Apply the yaml with the following command to deploy:

```shell
$ kubectl create namespace tensorflow-benchmark
$ kubectl create -f example/pod-memory.yaml
```

**If the Pod runs successfully, the output looks like below:**

```text
[INFO] 2021-03-27T04:26:40Z Query server 192.168.1.100:56001 gpu availability
[INFO] 2021-03-27T04:26:41Z Choosing GPUs from server list [192.168.1.100:56001]
[INFO] 2021-03-27T04:26:41Z Requesting GPUs [0] with 8080 MiB of memory from server 0, with version 2.5.0-fd3e4839...
Requested resources:
Server List: 192.168.1.100:56001
Client idle timeout: 0 min
[INFO] 2021-03-27T04:26:42Z Locked 1 GPUs with partial memory 0.5, configuration saved to '/tmp/bitfusion125236687'
[INFO] 2021-03-27T04:26:42Z Running client command 'python /benchmark/scripts/tf_cnn_benchmarks/tf_cnn_benchmarks.py --local_parameter_device=gpu --batch_size=32 --model=inception3' on 1 GPUs, with the following servers:
[INFO] 2021-03-27T04:26:42Z 192.168.1.100 55001 ab4a56d5-8df4-4c93-891d-1c5814cf83f6 56001 2.5.0-fd3e4839

2021-03-27 04:26:43.511803: I tensorflow/stream_executor/platform/default/dso_loader.cc:42] Successfully opened dynamic library libcudart.so.10.1

......

Instructions for updating:
non-resource variables are not supported in the long term
2021-03-27 04:26:48.173243: I tensorflow/core/platform/profile_utils/cpu_utils.cc:94] CPU Frequency: 2394455000 Hz
2021-03-27 04:26:48.174378: I tensorflow/compiler/xla/service/service.cc:168] XLA service 0x4c8ad60 executing computations on platform Host. Devices:
2021-03-27 04:26:48.174426: I tensorflow/compiler/xla/service/service.cc:175]   StreamExecutor device (0): <undefined>, <undefined>
2021-03-27 04:26:48.184024: I tensorflow/stream_executor/platform/default/dso_loader.cc:42] Successfully opened dynamic library libcuda.so.1
2021-03-27 04:26:54.831820: I tensorflow/stream_executor/cuda/cuda_gpu_executor.cc:1005] successful NUMA node read from SysFS had negative value (-1), but there must be at least one NUMA node, so returning NUMA node zero
2021-03-27 04:26:55.195722: I tensorflow/compiler/xla/service/service.cc:168] XLA service 0x4c927b0 executing computations on platform CUDA. Devices:
2021-03-27 04:26:55.195825: I tensorflow/compiler/xla/service/service.cc:175]   StreamExecutor device (0): Tesla V100-PCIE-16GB, Compute Capability 7.0
2021-03-27 04:26:56.476786: I tensorflow/stream_executor/cuda/cuda_gpu_executor.cc:1005] successful NUMA node read from SysFS had negative value (-1), but there must be at least one NUMA node, so returning NUMA node zero
2021-03-27 04:26:56.846965: I tensorflow/core/common_runtime/gpu/gpu_device.cc:1640] Found device 0 with properties:
name: Tesla V100-PCIE-16GB major: 7 minor: 0 memoryClockRate(GHz): 1.38
pciBusID: 0000:00:00.0
2021-03-27 04:26:56.847095: I tensorflow/stream_executor/platform/default/dso_loader.cc:42] Successfully opened dynamic library libcudart.so.10.1
2021-03-27 04:26:56.858148: I tensorflow/stream_executor/platform/default/dso_loader.cc:42] Successfully opened dynamic library libcublas.so.10
2021-03-27 04:26:56.870662: I tensorflow/stream_executor/platform/default/dso_loader.cc:42] Successfully opened dynamic library libcufft.so.10
2021-03-27 04:26:56.872082: I tensorflow/stream_executor/platform/default/dso_loader.cc:42] Successfully opened dynamic library libcurand.so.10
2021-03-27 04:26:56.884804: I tensorflow/stream_executor/platform/default/dso_loader.cc:42] Successfully opened dynamic library libcusolver.so.10
2021-03-27 04:26:56.891062: I tensorflow/stream_executor/platform/default/dso_loader.cc:42] Successfully opened dynamic library libcusparse.so.10
2021-03-27 04:26:56.916430: I tensorflow/stream_executor/platform/default/dso_loader.cc:42] Successfully opened dynamic library libcudnn.so.7
2021-03-27 04:26:57.108177: I tensorflow/stream_executor/cuda/cuda_gpu_executor.cc:1005] successful NUMA node read from SysFS had negative value (-1), but there must be at least one NUMA node, so returning NUMA node zero
2021-03-27 04:26:57.699172: I tensorflow/stream_executor/cuda/cuda_gpu_executor.cc:1005] successful NUMA node read from SysFS had negative value (-1), but there must be at least one NUMA node, so returning NUMA node zero
2021-03-27 04:26:58.487127: I tensorflow/core/common_runtime/gpu/gpu_device.cc:1763] Adding visible gpu devices: 0
2021-03-27 04:26:58.487327: I tensorflow/stream_executor/platform/default/dso_loader.cc:42] Successfully opened dynamic library libcudart.so.10.1
2021-03-27 04:53:53.568256: I tensorflow/core/common_runtime/gpu/gpu_device.cc:1181] Device interconnect StreamExecutor with strength 1 edge matrix:
2021-03-27 04:53:53.568703: I tensorflow/core/common_runtime/gpu/gpu_device.cc:1187]      0
2021-03-27 04:53:53.569011: I tensorflow/core/common_runtime/gpu/gpu_device.cc:1200] 0:   N
2021-03-27 04:53:53.939681: I tensorflow/stream_executor/cuda/cuda_gpu_executor.cc:1005] successful NUMA node read from SysFS had negative value (-1), but there must be at least one NUMA node, so returning NUMA node zero
2021-03-27 04:53:54.482940: I tensorflow/stream_executor/cuda/cuda_gpu_executor.cc:1005] successful NUMA node read from SysFS had negative value (-1), but there must be at least one NUMA node, so returning NUMA node zero
2021-03-27 04:53:54.846537: I tensorflow/core/common_runtime/gpu/gpu_device.cc:1326] Created TensorFlow device (/job:localhost/replica:0/task:0/device:GPU:0 with 7010 MB memory) -> physical GPU (device: 0, name: Tesla V100-PCIE-16GB, pci bus id: 0000:00:00.0, compute capability: 7.0)

......

TensorFlow:  1.14
Model:       inception3
Dataset:     imagenet (synthetic)
Mode:        training
SingleSess:  False
Batch size:  32 global
             32 per device
Num batches: 100
Num epochs:  0.00
Devices:     ['/gpu:0']
NUMA bind:   False
Data format: NCHW
Optimizer:   sgd
Variables:   parameter_server
==========
Generating training model
Initializing graph
Running warm up
Done warm up
Step    Img/sec total_loss
1       images/sec: 199.4 +/- 0.0 (jitter = 0.0)        7.312
10      images/sec: 196.6 +/- 2.1 (jitter = 5.7)        7.290
20      images/sec: 198.3 +/- 1.3 (jitter = 4.5)        7.351
30      images/sec: 198.4 +/- 0.9 (jitter = 3.8)        7.300
40      images/sec: 199.4 +/- 0.8 (jitter = 4.1)        7.250
50      images/sec: 199.8 +/- 0.7 (jitter = 4.6)        7.283
60      images/sec: 200.1 +/- 0.6 (jitter = 4.2)        7.301
70      images/sec: 199.8 +/- 0.6 (jitter = 4.2)        7.266
80      images/sec: 200.1 +/- 0.6 (jitter = 4.4)        7.286
90      images/sec: 199.9 +/- 0.5 (jitter = 4.4)        7.334
100     images/sec: 199.9 +/- 0.5 (jitter = 4.0)        7.380
----------------------------------------------------------------
total images/sec: 199.65
----------------------------------------------------------------

......
```



### 4.3. The configuration of "auto-management/bitfusion parameter"

If the value of auto-management/bitfusion parameter is set to "all", use the following command to to check the status of the pod when the workload has been summited. 

```bash
$ kubectl edit pod -n tensorflow-benchmark bf-pkgs
```

```yaml
apiVersion: v1
kind: Pod
metadata:
  annotations:
    # update annotations
    auto-management/status: injected
  name: bf-pkgs
  namespace: tensorflow-benchmark
spec:
  containers:
  # update command
  - command:
    - /bin/bash
    - -c
    - /bitfusion/bitfusion-client-ubuntu1804_2.5.1-13/usr/bin/bitfusion run -n 1 -p
      0.500000 python /benchmark/scripts/tf_cnn_benchmarks/tf_cnn_benchmarks.py --local_parameter_device=gpu
      --batch_size=32 --model=inception3
    env:
    # add LD_LIBRARY_PATH
    - name: LD_LIBRARY_PATH
      value: /bitfusion/bitfusion-client-ubuntu1804_2.5.1-13/opt/bitfusion/lib/x86_64-linux-gnu/lib/:/usr/local/nvidia/lib:/usr/local/nvidia/lib64
    image: nvcr.io/nvidia/tensorflow:19.07-py3
    imagePullPolicy: IfNotPresent
    name: bf-pkgs
    resources:
      limits:
        # update resource name
        bitfusion.io/gpu: "50"
      requests:
        bitfusion.io/gpu: "50"
    volumeMounts:
    - mountPath: /benchmark
      name: code
    # add some volume
    - mountPath: /etc/bitfusion
    .......
  # add initContainer
  initContainers:
  - command:
    - /bin/bash
    - -c
    - ' cp -ra /bitfusion/* /bitfusion-distro/ && cp /root/.bitfusion/client.yaml
      /client && cp -r /bitfusion/bitfusion-client-centos7-2.5.0-10/opt/bitfusion/*
      /workload-container-opt '
    image: docker.io/bitfusiondeviceplugin/bitfusion-client:0.1.1
    imagePullPolicy: IfNotPresent
    name: populate
    ......
```

It shows the command of the workload has been mutated with the Bitfusion parameters.

Besides that, if the auto-management/bitfusion is set to "injection",  use the following command to to check the status of the pod when the workload has been summited.

```bash
$ kubectl edit pod -n tensorflow-benchmark bf-pkgs
```

```yaml
......
spec:
  containers:
  # Not to change the command
  - command:
    - /bin/bash
    - -c
    - python /benchmark/scripts/tf_cnn_benchmarks/tf_cnn_benchmarks.py --local_parameter_device=gpu
      --batch_size=32 --model=inception3
    
    ......
    
```

The command of the workload has not been mutated, but the Bitfusion Baremetal token, Bitfusion distros and other configuration are injected into the container. Users can manually write the command of the container and use the Bitfusion command as they would like to.

Finally, use the following command to remove POD: 

```
$ kubectl delete -f example/pod.yaml
```

### 4.4. The configuration of "bitfusion-client/filter parameter"

Bitfusion added the filter parameter in version 4.0.The filter parameter has:
- server.addr (for example: server.addr=10.117.32.177)
- server.hostname (for example: server.hostname=bf-server)
- server.has-rdma (for example: server.has-rdma=true)
- server.cuda-version (for example: server.cuda-version=11.2)
- server.dirver-version (for example: server.dirver-version=460.73.01)
- device.id (for example: device.id=0)
- device.name (for example: device.name=Tesla)
- device.phy-memory (for example: device.phy-memory=16160)

Using feilter requires the addition of annotations for  **bitfusion-client/filter**

```yaml

apiVersion: v1
kind: Pod
metadata:
  annotations:
    auto-management/bitfusion: "all"
    bitfusion-client/os: "ubuntu18"
    bitfusion-client/version: "401"
    # Use a single filter condition
    bitfusion-client/filter: "server.hostname=bf-server"
  name: bf-pkgs
  namespace: tensorflow-benchmark
spec:
  containers:
    - image: nvcr.io/nvidia/tensorflow:19.07-py3
      imagePullPolicy: IfNotPresent
      name: bf-pkgs
      command: ["python /benchmark/scripts/tf_cnn_benchmarks/tf_cnn_benchmarks.py --local_parameter_device=gpu --batch_size=32 --model=inception3"]
      resources:
        limits:
          bitfusion.io/gpu-amount: 1
          bitfusion.io/gpu-percent: 50
      volumeMounts:
        - name: code
          mountPath: /benchmark
  volumes:
    - name: code
      hostPath:
        path: /home/benchmarks

```

```yaml

apiVersion: v1
kind: Pod
metadata:
  annotations:
    auto-management/bitfusion: "all"
    bitfusion-client/os: "ubuntu18"
    bitfusion-client/version: "401"
    # Use multiple filter criteria
    bitfusion-client/filter: "server.hostname=bf-server server.addr=192.168.1.1 server.hostname=bf-server2"
  name: bf-pkgs
  namespace: tensorflow-benchmark
spec:
  containers:
    - image: nvcr.io/nvidia/tensorflow:19.07-py3
      imagePullPolicy: IfNotPresent
      name: bf-pkgs
      command: ["python /benchmark/scripts/tf_cnn_benchmarks/tf_cnn_benchmarks.py --local_parameter_device=gpu --batch_size=32 --model=inception3"]
      resources:
        limits:
          bitfusion.io/gpu-amount: 1
          bitfusion.io/gpu-percent: 50
      volumeMounts:
        - name: code
          mountPath: /benchmark
  volumes:
    - name: code
      hostPath:
        path: /home/benchmarks

```

## 5.  Resource Quota (optional)
### 5.1. Enforce Quota


The resource name of the device plugin is **bitfusion.io/gpu**. Use the following command to enforce quota.  The value of "requests.bitfusion.io/gpu" is set to 100 means it is able to use of 100% of one GPU at most which is equivalent to the usage of two half GPUs. In addition to that, quota limits can only be set with "requests.bitfusion.io/gpu" and "limits.bitfusion.io/gpu" is not valid for quota.


Use the following command to create the quota

```
cat <<EOF | kubectl create -f -
apiVersion: v1
kind: List
items:
- apiVersion: v1
  kind: ResourceQuota
  metadata:
    name: bitfusion-quota
    namespace: tensorflow-benchmark
  spec:
    hard:
        requests.bitfusion.io/gpu: 100
EOF 
```


### 5.2. Validate the quota using the following two methods

#### 5.2.1. Using parameter "bitfusion.io/gpu-memory"

Use the following command to create the pod.  

```
cat <<EOF | kubectl create -f -
apiVersion: v1
kind: Pod
metadata:
  annotations:
    auto-management/bitfusion: "all"
    bitfusion-client/os: "ubuntu18"
    bitfusion-client/version: "401"
  name: bf-pkgs
  namespace: tensorflow-benchmark
spec:
  containers:
    - image: nvcr.io/nvidia/tensorflow:19.07-py3
      imagePullPolicy: IfNotPresent
      name: bf-pkgs
      command: ["python /benchmark/scripts/tf_cnn_benchmarks/tf_cnn_benchmarks.py --local_parameter_device=gpu --batch_size=32 --model=inception3"]
      resources:
        limits:
          bitfusion.io/gpu-amount: 1
          bitfusion.io/gpu-memory: 8000M
      volumeMounts:
        - name: code
          mountPath: /benchmark
  volumes:
    - name: code
      hostPath:
        path: /home/benchmarks
EOF
```

Since the value of TOTAL_GPU_MEMORY forementioned is set to 16000M, for the pod which request 8000M gpu memory will consume 50% of bitfusion.io/gpu quota

Calculation method of the quota is:  **[bitfusion.io/gpu]** = gpu-memory / TOTAL_GPU_MEMORY * gpu-amount * 100; The result is rounded up to integer.

Use the following command to check the quota consumption:  

```
$ kubectl describe quota -n tensorflow-benchmark bitfusion-quota 
```

![img](diagrams/quota.png) 

#### 5.2.2. Using parameter "bitfusion.io/gpu-percent"

Use the following command to create the pod

```
cat <<EOF | kubectl create -f -
apiVersion: v1
kind: Pod
metadata:
  annotations:
    auto-management/bitfusion: "all"
    bitfusion-client/os: "ubuntu18"
    bitfusion-client/version: "401"
  name: bf-pkgs
  namespace: tensorflow-benchmark
spec:
  containers:
    - image: nvcr.io/nvidia/tensorflow:19.07-py3
      imagePullPolicy: IfNotPresent
      name: bf-pkgs
      command: ["python /benchmark/scripts/tf_cnn_benchmarks/tf_cnn_benchmarks.py --local_parameter_device=gpu --batch_size=32 --model=inception3"]
      resources:
        limits:
          bitfusion.io/gpu-amount: 1
          bitfusion.io/gpu-percent: 50
      volumeMounts:
        - name: code
          mountPath: /benchmark
  volumes:
    - name: code
      hostPath:
        path: /home/benchmarks
EOF
```

Calculation method of the quota is:  **[bitfusion.io/gpu]** = gpu-percent * gpu-amount 


Use the following command to check the quota consumption:  

```
$ kubectl describe quota -n tensorflow-benchmark bitfusion-quota 
```

![img](diagrams/quota.png) 



## 6. Troubleshooting

- If the workload did not run successfully, use the command below to check the log of the workload for details.  

```shell
$ kubectl logs -n tensorflow-benchmark   bf-pkgs
```
"tensorflow-benchmark" is the namespace of the pod.
"bf-pkgs" is the pod name.

The logs below indicate some errors of contacting Bitfusion server.
```
root@nodel:~/final-test/bitfusion-with-kubernetes-integration-main/bitfusion_device_plugin# kubectl logs -n tensorflow-benchmark bf-deployment-54c675cbff-sb4bq 
[INFO] 2021-03-29T01:05:36Z Query server 10.117.32.156:56001 to claim host id: 012b965d-93b8-465f-97eb-eaba51bc3599
[WARN] 2021-03-29T01:05:37Z Error contacting server 10.117.32.156:56001 to verify client id: Get https://10.117.32.156:56001/claim_id: context deadline exceeded Error: none of the servers responded correctly to client id claim requests.
root@nodel:~/final-test/bitfusion-with-kubernetes-integration-main/bitfusion_device_plugin# 
```


Check the validity of the **Baremetal token** from vCenter Bitfusion Plugin. 
Re-download a new valid token and use the following commands to update the secret in Kubernetes:  (Make sure to delete all the stale bitfusion-secret in each namespace of Kubernetes)


```
$ kubectl delete secret -n kube-system bitfusion-secret  
$ kubectl delete secret -n tensorflow-benchmark  bitfusion-secret  
$ kubectl create secret generic bitfusion-secret --from-file=tokens -n kube-system
```

- If the following error occurs when running POD, modify the Serve.conf file in the tokens directory
```
root@nodel:~/final-test/bitfusion-with-kubernetes-integration-main/bitfusion_device_plugin# kubectl logs -n tensorflow-benchmark bf-pkgs
[INFO] 2021-03-29T01:05:36Z Query server health status
[WARN] 2021-03-29T01:05:37Z Failed to parse /root/.bitfusion/servers.conf: open /root/.bitfusion/servers.conf: read-only file system
Error: none of the servers responded correctly to client id claim requests.
```
Change the servers.conf file to the following format.The ACTUAL IP address prevails
```
servers:
- reachable:10.202.122.248:56001
  addresses:
  - 10.202.122.248:56001
```

Then re-run the following command to create secret 
```
$ kubectl delete secret -n kube-system bitfusion-secret  
$ kubectl delete secret -n tensorflow-benchmark  bitfusion-secret  
$ kubectl create secret generic bitfusion-secret --from-file=tokens -n kube-system
```

- If the following error occurs when running POD, please check the k8s environment 's network connection
```
[rootbf—pkgs bin]# ./bitfusion list_gpus
[INFO] 2021—09—22T11:24:04Z Query server 10.117.32.156:56001 to claim host id: 1a722830—1d27—4ee9—ac0b-f77c19fb189c
[WARN] 2021—09—22T11:2405Z Error contacting server 10.117.32.156:56001 to verify client id: Get https://10.117.32.156:56001/claim id: dial tcp 10.117.32.156:56001: i/o timeout
Error: none of the servers responded correctly to client id claim requests.
```

## 7. Note

### 7.1. The environment variable of LD_LIBRARY_PATH

If the variable LD_LIBRARY_PATH is set in Dockerfile, it is needed to reset the value of this variable in the YAML file of the workload which will use it, otherwise it won't go into effect.

The value in Dockfile will not take any effects.

```Dockerfile
ENV LD_LIBRARY_PATH /usr/local/lib
```



The YAML file of the workload which specify the value of the LD_LIBRARY_PATH variable:

```yaml
apiVersion: v1
kind: Pod
......
  containers:
    - image: nvcr.io/nvidia/tensorflow:19.07-py3
      ......
      command: ["python /benchmark/scripts/tf_cnn_benchmarks/tf_cnn_benchmarks.py --local_parameter_device=gpu --batch_size=32 --model=inception3"]
      env:
        - name: LD_LIBRARY_PATH
          value: /usr/local/lib
      resources:
        limits:
          bitfusion.io/gpu-amount: 1
          bitfusion.io/gpu-percent: 50
......

```

### 7.2. Deploy the Bitfusion Device Plugin on Tanzu

In order to deploy the Bitfusion device plugin on TKGi(Tanzu Kubernetes Grid Integrated), CFSSL is needed to be installed manually on the installation machine. Then update the value of  K8S_PLATFORM in the **bitfusion-with-kubernetes-integration-main/bitfusion_device_plugin/Makefile** to "tkgi".


```shell
K8S_PLATFORM ?= tkgi
```

After that run the "make deploy" command to start the deployment.

### 7.3 Alternative docker image registry

Below is another image regsitry for users to get the docker images.
```shell
ccr.ccs.tencentyun.com/bitfusion/bitfusion-device-plugin:0.3

ccr.ccs.tencentyun.com/bitfusion/bitfusion-webhook:0.3

ccr.ccs.tencentyun.com/bitfusion/bitfusion-client:0.3
```

