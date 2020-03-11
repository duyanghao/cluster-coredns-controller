module github.com/duyanghao/cluster-coredns-controller

go 1.13

replace (
	// wait https://github.com/chartmuseum/storage/pull/34 to be merged
	github.com/chartmuseum/storage => github.com/choujimmy/storage v0.5.1-0.20191225102245-210f7683d0a6
	github.com/deislabs/oras => github.com/deislabs/oras v0.8.0
	// wait https://github.com/dexidp/dex/pull/1607 to be merged
	github.com/dexidp/dex => github.com/choujimmy/dex v0.0.0-20191225100859-b1cb4b898bb7
	k8s.io/client-go => k8s.io/client-go v0.17.0
	tkestack.io/tke => github.com/tkestack/tke v1.1.0
)

require (
	gopkg.in/yaml.v2 v2.2.4
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog v1.0.0
	// k8s.io/client-go v12.0.0+incompatible
	tkestack.io/tke v1.1.0
)
