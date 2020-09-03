module github.com/duyanghao/cluster-coredns-controller

go 1.13

replace (
	github.com/chartmuseum/storage => github.com/choujimmy/storage v0.0.0-20200507092433-6aea2df34764
	github.com/deislabs/oras => github.com/deislabs/oras v0.8.0
	github.com/dovics/domain-role-manager => github.com/amasser/domain-role-manager v0.0.0-20200325101749-a44f9c315081
	helm.sh/helm/v3 => helm.sh/helm/v3 v3.2.0
	k8s.io/client-go => k8s.io/client-go v0.18.2
)

require (
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog v1.0.0
	tkestack.io/tke v1.3.0
)
