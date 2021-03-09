# Bitfusion on Kubernetes ##


Current problems of virtual GPU (Nvidia etc.) :

Underutilized GPU compute cycle
Limited and preset granularity
Resource bound to local machine 
Hard for application scheduling
Bitfusion helps address the problems by providing remote GPU pool. Bitfusion makes GPUs a first class citizen that can be abstracted, partitioned, automated and shared like traditional compute resource. On the other hand, Kubernetes has become the de facto platform to deploy and manage the machine learning workload.

However, out of the box Kubernetes does not offer a way to consume Bitfusion's network-attached GPUs. This limitation becomes a key challenge is to enable jobs on Kubernetes to use Bitfusion’s GPU. Bitfusion it is not friendly with Kubernetes in below aspects:

1. Resource management:

  - No resource management capability from Kubernetes

  - Security problems

  - Not compatible with Kubernetes ecosystem

2.GPU pool management

3.Network bottleneck between applications and GPU pool

To address these problems, we create this project to allow Bitfusion to work with Kubernetes.




## Architecture

![img](./architecture.png)
Bitfusion on Kubernetes consists of the following two components.<br/>
 1.bitfusion-device-plugin<br/>
 2.kubernetes mutating webhook

Items 1 and 2 are built into separated docker containers. <br/>
bitfusion-device-plugin will run on each agent node where kubelet is running.<br/>
kubernetes mutating webhook will run as a deployment on the kubernetes master node.

## Installation ##

**Build docker image**

Docker image is can be build from Dockerfile we provided:

```shell
docker build -f device-plugin/build/Dockerfile -t <repository>/bitfusion-device-plugin .
```

```shell
docker build -f webhook/build/Dockerfile -t <repository>/bitfusion-webhook .
```

**Deploy to Kubernetes**

Upload the Baremetal Tokens files that are available to you and unzip them into your directory  
Then use the following command to create Secret
```shell
kubectl create secret generic bitfusion-secret --from-file=your/path -n kube-system
```

Configure device_plugin.yml to change the container image:

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: bitfusion-cli-device-plugin
  namespace: kube-system
spec:
  selector:
    matchLabels:
      tier: node
  template:
    metadata:
      labels:
        tier: node
    spec:
      hostNetwork: true
      containers:
        - name: device-plugin-ctr
          image: <your-image-name>:latest
          securityContext:
            privileged: true
          env:
            - name: SOCKET_NAME
              value: "bitfusion.io"
            - name: INTERVAL
              value: "10"
            - name: RESOURCE_NAME
              value: "bitfusion.io/gpu"
            - name: RESOURCE_NUMS
              value: "1000"
          volumeMounts:
            - mountPath: "/var/lib/kubelet"
              name: kubelet-socket
            - mountPath: "/etc/kubernetes/pki"
              name: pki
      volumes:
        - hostPath:
            path: "/var/lib/kubelet"
          name: kubelet-socket
        - hostPath:
            path: "/etc/kubernetes/pki"
          name: pki
```

<br/>

Configure webhook.yml to change the container image:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sidecar-injector-webhook-deployment
  namespace: sidecar-injector
  labels:
    app: sidecar-injector
spec:
  replicas: 1
  selector:
    matchLabels:
      app: sidecar-injector
  template:
    metadata:
      labels:
        app: sidecar-injector
    spec:
      containers:
        - name: sidecar-injector
          image: <your-image-name>:latest
          imagePullPolicy: IfNotPresent
          args:
          - -sidecarCfgFile=/etc/webhook/config/sidecarconfig.yaml
          - -tlsCertFile=/etc/webhook/certs/cert.pem
          - -tlsKeyFile=/etc/webhook/certs/key.pem
          - -alsologtostderr
          - -v=4
          - 2>&1
          volumeMounts:
          - name: webhook-certs
            mountPath: /etc/webhook/certs
            readOnly: true
          - name: webhook-config
            mountPath: /etc/webhook/config
      volumes:
      - name: webhook-certs
        secret:
          secretName: sidecar-injector-webhook-certs
      - name: webhook-config
        configMap:
          name: sidecar-injector-webhook-configmap

```

Then apply the yaml with the following command to deploy:

```shell
kubectl apply -f device-plugin/deployment/*.yml
```

```shell
cd webhook 
./deploy.sh
```


## usage ##


Configure pod.yml to change the hostPath:

```yaml
apiVersion: v1
kind: Pod
metadata:
  annotations:
    auto-management/bitfusion: "yes"
  name: bf-pkgs
  namespace: injection
spec:
  containers:
    - image: nvcr.io/nvidia/tensorflow:19.07-py3
      imagePullPolicy: IfNotPresent
      name: bf-pkgs
      command: ["python /benchmark/scripts/tf_cnn_benchmarks/tf_cnn_benchmarks.py --local_parameter_device=gpu --batch_size=32 --model=inception3"]
      resources:
        limits:
          bitfusion.io/gpu-num: 1
          bitfusion.io/gpu-percent: 100
      volumeMounts:
        - name: code
          mountPath: /benchmark
  volumes:
    - name: code
      hostPath:
        path: <your code path>

```

Benchmarks  https://github.com/tensorflow/benchmarks/tree/tf_benchmark_stage  

Then apply the yaml with the following command to deploy:

```
kubeclt apply -f example/pod.yaml
```

**If the Pod runs successfully, view the Pod log content as**
![img](./pod-success.png)
