#!/bin/bash
#
# Copyright 2020 VMware, Inc.
#
# SPDX-License-Identifier: Apache-2.0

if [ `whoami` != "root" ]
then
	echo "Run this script with root!"
	exit
fi

if [ -e "/etc/os-release" ]
then
	ID=$(grep -oP '(?<=^ID=).+' /etc/os-release | tr -d '"')
	VERSION=$(grep -oP '(?<=^VERSION_ID=).+' /etc/os-release | tr -d '"')

	if [ $ID:$VERSION != "rhel:7.4" ] && [ $ID:$VERSION != "ubuntu:18.04" ] && [ $ID:$VERSION != "ubuntu:16.04" ]
	then
		echo "Unsupported OS"
		exit
	fi
else
	echo "Unknown OS"
	exit
fi

echo -e "\033[32m ====> Install Nvidia driver dependency \033[0m"
if [ $ID:$VERSION == "rhel:7.4" ]
then
	yum install -y -q open-vm-tools \
	build-essential \
	kmod \
	rdma-core \
	libibverbs \
	libibverbs-utils \
	infiniband-diags
fi

if [ $ID:$VERSION == "ubuntu:18.04" ] && [ $ID:$VERSION == "ubuntu:16.04" ]
then
	apt-get update && apt-get install -y --no-install-recommends \
		open-vm-tools \
		apt-utils \
		build-essential \
		ca-certificates \
		curl \
		kmod
fi

BASE_URL=https://us.download.nvidia.com/tesla
DRIVER_VERSION=418.87.01
SHORT_DRIVER_VERSION=418.87

# Install the userspace components and copy the kernel module sources.
echo -e "\033[32m ====> Download and Install Nvidia driver userspace components \033[0m"
cd /tmp && \
curl -fSlL -O $BASE_URL/$SHORT_DRIVER_VERSION/NVIDIA-Linux-x86_64-$DRIVER_VERSION.run && \
sh NVIDIA-Linux-x86_64-$DRIVER_VERSION.run -x && \
cd NVIDIA-Linux-x86_64-$DRIVER_VERSION && \
./nvidia-installer --silent \
                   --no-kernel-module \
                   --install-compat32-libs \
                   --no-nouveau-check \
                   --no-nvidia-modprobe \
                   --no-rpms \
                   --no-backup \
                   --no-check-for-alternate-installs \
                   --no-libglx-indirect \
                   --no-install-libglvnd \
                   --x-prefix=/tmp/null \
                   --x-module-path=/tmp/null \
                   --x-library-path=/tmp/null \
                   --x-sysconfig-path=/tmp/null \
                   --no-glvnd-egl-client \
                   --no-glvnd-glx-client && \
cd ~ && rm -rf /tmp/*

# Install CUDA base
echo -e "\033[32m ====> Download and Install CUDA \033[0m"

if [ $ID:$VERSION == "ubuntu:18.04" ]
then
	apt-get install -y --no-install-recommends \
	    gnupg2 curl ca-certificates && \
	curl -fsSL https://developer.download.nvidia.com/compute/cuda/repos/ubuntu1804/x86_64/7fa2af80.pub | apt-key add - && \
	echo "deb https://developer.download.nvidia.com/compute/cuda/repos/ubuntu1804/x86_64 /" > /etc/apt/sources.list.d/cuda.list && \
	echo "deb https://developer.download.nvidia.com/compute/machine-learning/repos/ubuntu1804/x86_64 /" > /etc/apt/sources.list.d/nvidia-ml.list && \
	rm -rf /var/lib/apt/lists/*
fi

if [ $ID:$VERSION == "ubuntu:16.04" ]
then
	apt-get install -y --no-install-recommends \
	    apt-transport-https gnupg-curl && \
	NVIDIA_GPGKEY_SUM=d1be581509378368edeec8c1eb2958702feedf3bc3d17011adbf24efacce4ab5 && \
	NVIDIA_GPGKEY_FPR=ae09fe4bbd223a84b2ccfce3f60f4b3d7fa2af80 && \
	apt-key adv --fetch-keys https://developer.download.nvidia.com/compute/cuda/repos/ubuntu1604/x86_64/7fa2af80.pub && \
	apt-key adv --export --no-emit-version -a $NVIDIA_GPGKEY_FPR | tail -n +5 > cudasign.pub && \
	echo "$NVIDIA_GPGKEY_SUM  cudasign.pub" | sha256sum -c --strict - && rm cudasign.pub && \
	echo "deb https://developer.download.nvidia.com/compute/cuda/repos/ubuntu1604/x86_64 /" > /etc/apt/sources.list.d/cuda.list && \
	echo "deb https://developer.download.nvidia.com/compute/machine-learning/repos/ubuntu1604/x86_64 /" > /etc/apt/sources.list.d/nvidia-ml.list && \
	rm -rf /var/lib/apt/lists/*
fi

CUDA_VERSION=10.0.130
if [ $ID:$VERSION == "rhel:7.4" ]
then
	CUDA_PKG_VERSION=10-0-$CUDA_VERSION
	# For libraries in the cuda-compat-* package: https://docs.nvidia.com/cuda/eula/index.html#attachment-a
	sudo yum-config-manager --add-repo http://developer.download.nvidia.com/compute/cuda/repos/rhel7/x86_64/cuda-rhel7.repo
	yum install -y -q cuda-cudart-$CUDA_PKG_VERSION \
	cuda-compat-10-0 && \
	ln -s cuda-10.0 /usr/local/cuda

	PATH=/usr/local/nvidia/bin:/usr/local/cuda/bin:${PATH}
	LD_LIBRARY_PATH=/usr/local/nvidia/lib:/usr/local/nvidia/lib64

	# Install CUDA runtime
	echo -e "\033[32m ====> Download and Install CUDA runtimes \033[0m"
	NCCL_VERSION=2.5.6
	yum install -y -q cuda-libraries-$CUDA_PKG_VERSION \
	cuda-nvtx-$CUDA_PKG_VERSION 
	yum install -y -q https://developer.download.nvidia.com/compute/machine-learning/repos/rhel7/x86_64/nvidia-machine-learning-repo-rhel7-1.0.0-1.x86_64.rpm
	# yum clean all
	yum update -q
	yum install -y -q libnccl-$NCCL_VERSION-1+cuda10.0
	echo exclude=libnccl >> /etc/yum.conf

	# Install CUDA cudnn
	echo -e "\033[32m ====> Download and Install CUDA cudnn \033[0m"
	CUDNN_VERSION=7.6.5.32

	yum install -y -q libcudnn7-$CUDNN_VERSION-1.cuda10.0
	echo exclude=libcudnn7 >> /etc/yum.conf
	
else
	CUDA_PKG_VERSION=10-0=$CUDA_VERSION-1

	# For libraries in the cuda-compat-* package: https://docs.nvidia.com/cuda/eula/index.html#attachment-a
	apt-get update && apt-get install -y --no-install-recommends \
		cuda-cudart-$CUDA_PKG_VERSION \
		cuda-compat-10-0 && \
	ln -s cuda-10.0 /usr/local/cuda

	PATH=/usr/local/nvidia/bin:/usr/local/cuda/bin:${PATH}
	LD_LIBRARY_PATH=/usr/local/nvidia/lib:/usr/local/nvidia/lib64

	# Install CUDA runtime
	echo -e "\033[32m ====> Download and Install CUDA runtimes \033[0m"
	NCCL_VERSION=2.5.6

	apt-get install -y --no-install-recommends \
		cuda-libraries-$CUDA_PKG_VERSION \
		cuda-nvtx-$CUDA_PKG_VERSION \
		libnccl2=$NCCL_VERSION-1+cuda10.0 && \
	apt-mark hold libnccl2

	# Install CUDA cudnn
	echo -e "\033[32m ====> Download and Install CUDA cudnn \033[0m"
	CUDNN_VERSION=7.6.5.32

	apt-get install -y --no-install-recommends \
		libcudnn7=$CUDNN_VERSION-1+cuda10.0 && \
	apt-mark hold libcudnn7
fi


# Install Bitfusion dependency
if [ $ID:$VERSION == "ubuntu:18.04" ]
then
	apt-get install -y --no-install-recommends \
	    wget libjsoncpp1 librdmacm1 libssl-dev libibverbs1 libnuma1 libcapstone3 libnl-3-200 libnl-route-3-200 uuid
fi

if [ $ID:$VERSION == "ubuntu:16.04" ]
then
	apt-get install -y --no-install-recommends \
	    wget libjsoncpp1 librdmacm1 libssl-dev libibverbs1 libnuma1 libcapstone3 libnl-3-200 libnl-route-3-200 uuid \
	    zlib1g-dev libossp-uuid16
fi

if [ $ID:$VERSION == "rhel:7.4" ]
then 
	yum install -y -q wget json-c-devel librdmacm libibverbs libuuid proc-ng-devel
fi
