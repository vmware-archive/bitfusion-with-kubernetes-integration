module github.com/vmware/bitfusion-device-plugin

go 1.14

require (
        github.com/docker/distribution v2.8.2-beta.1
	github.com/ghodss/yaml v1.0.0
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/stretchr/testify v1.5.1
	golang.org/x/net v0.7.0
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.19.0
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v0.20.0-alpha.2
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920 // indirect
)

replace (
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf => github.com/golang/protobuf v1.4.3
	gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.3.0
)
