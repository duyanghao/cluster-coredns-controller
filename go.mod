module github.com/duyanghao/cluster-coredns-controller

go 1.12

replace (
	k8s.io/api => k8s.io/api v0.0.0-20191016110408-35e52d86657a
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20191016113550-5357c4baaf65
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191004115801-a2eda9f80ab8
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20191016112112-5190913f932d
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20191016114015-74ad18325ed5
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191016111102-bec269661e48
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20191016115129-c07a134afb42
	// Use component-base v1.17.0-rc1 for use github.com/prometheus/client_golang v1.0.0
	k8s.io/component-base => k8s.io/component-base v0.0.0-20191122220729-2684fb322cb9
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20191016112429-9587704a8ad4
	tkestack.io/tke v1.0.1 => github.com/t1mt/tke v1.0.1
)

require gopkg.in/yaml.v2 v2.2.7
