module github.com/vmware/bitfusion-device-plugin

go 1.14

require (
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/stretchr/testify v1.4.0
	golang.org/x/net v0.0.0-20201021035429-f5854403a974
	google.golang.org/grpc v1.27.0
	k8s.io/kubelet v0.19.4
)

replace github.com/golang/protobuf => github.com/golang/protobuf v1.4.3
