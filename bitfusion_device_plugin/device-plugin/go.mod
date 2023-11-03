module github.com/vmware/bitfusion-device-plugin

go 1.14

require (
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/stretchr/testify v1.4.0
	golang.org/x/net v0.17.0
	google.golang.org/grpc v1.56.3
	k8s.io/kubelet v0.19.6
        k8s.io/client-go v0.19.6
)

replace (
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf => github.com/golang/protobuf v1.4.3
)
