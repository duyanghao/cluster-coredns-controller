/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/duyanghao/cluster-coredns-controller/cmd/controller/clustercoredns"
	"github.com/duyanghao/cluster-coredns-controller/cmd/controller/clustercoredns/config"
	"github.com/duyanghao/cluster-coredns-controller/pkg/signals"
	versionedclientset "github.com/tkestack/tke/api/client/clientset/versioned"
	platformversionedclient "github.com/tkestack/tke/api/client/clientset/versioned/typed/platform/v1"
	versionedinformers "github.com/tkestack/tke/api/client/informers/externalversions"
)

var (
	argConfigPath string
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// load configuration
	clusterCorednsCfg, err := config.LoadConfig(argConfigPath)
	if err != nil {
		klog.Fatalf("Error loading clusterCorednsConfig: %s", err.Error())
	}
	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(clusterCorednsCfg.ClusterServerCfg.MasterURL, clusterCorednsCfg.ClusterServerCfg.KubeConfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	versionedClient, err := versionedclientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building versiond clientset: %s", err.Error())
	}

	clusterInformerFactory := versionedinformers.NewSharedInformerFactory(versionedClient, time.Second*30)

	clusterClient, err := platformversionedclient.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building cluster clientset: %s", err.Error())
	}

	controller := clustercoredns.NewController(clusterCorednsCfg, kubeClient, clusterClient.Clusters(), clusterInformerFactory.Platform().V1().Clusters())

	// notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(stopCh)
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	clusterInformerFactory.Start(stopCh)

	if err = controller.Run(1, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}

func init() {
	flag.StringVar(&argConfigPath, "c", "/cluster-coredns-controller/examples/config.yml", "The configuration filepath for cluster-coredns-controller.")
}
