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

package clustercoredns

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/duyanghao/cluster-coredns-controller/cmd/controller/clustercoredns/config"
	"github.com/duyanghao/cluster-coredns-controller/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	clusterscheme "github.com/tkestack/tke/api/client/clientset/versioned/scheme"
	platformversionedclient "github.com/tkestack/tke/api/client/clientset/versioned/typed/platform/v1"
	informers "github.com/tkestack/tke/api/client/informers/externalversions/platform/v1"
	listers "github.com/tkestack/tke/api/client/listers/platform/v1"
	v1 "github.com/tkestack/tke/api/platform/v1"
)

const controllerAgentName = "cluster-coredns-controller"

const (
	// SuccessSynced is used as part of the Event 'reason' when a Cluster is synced
	SuccessSynced = "Synced"

	// MessageResourceSynced is the message used for an Event fired when a Cluster
	// is synced successfully
	MessageResourceSynced = "Cluster synced successfully"
)

// Controller is the controller implementation for Cluster resources
type Controller struct {
	cfg              *config.Config
	clusterclientset platformversionedclient.ClusterInterface
	clustersLister   listers.ClusterLister
	clustersSynced   cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewController returns a new sample controller
func NewController(
	clusterCfg *config.Config,
	kubeclientset kubernetes.Interface,
	clusterclientset platformversionedclient.ClusterInterface,
	clusterInformer informers.ClusterInformer) *Controller {
	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	var recorder record.EventRecorder
	if clusterCfg.ClusterServerCfg.EnableEvent {
		utilruntime.Must(clusterscheme.AddToScheme(scheme.Scheme))
		klog.V(4).Info("Creating event broadcaster")
		eventBroadcaster := record.NewBroadcaster()
		eventBroadcaster.StartLogging(klog.Infof)
		eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
		recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})
	}

	// replace corednsServerBlockTemplate for preparation ...
	constants.CorednsServerBlockTemplate = strings.ReplaceAll(constants.CorednsServerBlockTemplate, "INTERVAL", controller.cfg.CoreDnsCfg.Interval)
	constants.CorednsServerBlockTemplate = strings.ReplaceAll(constants.CorednsServerBlockTemplate, "JITTER", controller.cfg.CoreDnsCfg.Jitter)

	controller := &Controller{
		cfg:              clusterCfg,
		clusterclientset: clusterclientset,
		clustersLister:   clusterInformer.Lister(),
		clustersSynced:   clusterInformer.Informer().HasSynced,
		workqueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Clusters"),
		recorder:         recorder,
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when Cluster resources change
	clusterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueClusterAdd,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueClusterUpdate(new)
		},
		DeleteFunc: controller.enqueueClusterDelete,
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting Cluster controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.clustersSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch two workers to process Cluster resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Cluster resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Corefile with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	klog.V(4).Infof("Cluster (%s) namespace: (%s), name: (%s)", key, namespace, name)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the Cluster resource with this namespace/name
	exist := true
	cluster, err := c.clustersLister.Get(name)
	if err != nil {
		// The Cluster resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("cluster '%s' in work queue no longer exists", key))
			exist = false
			//return nil
		} else {
			klog.Errorf("Get Cluster from indexer failed: %s", err.Error())
			return err
		}
	}
	klog.V(5).Infof("Get Cluster: %+v from indexer successfully", cluster)

	// Finally, we update the Corefile of coredns to reflect the
	// current state of the world
	err = c.updateCorefile()
	if err != nil {
		klog.Errorf("update Corefile of coredns failed: %s", err.Error())
		return err
	}

	klog.V(4).Infof("update Corefile of coredns successfully ...")
	if exist && c.cfg.ClusterServerCfg.EnableEvent {
		c.recorder.Event(cluster, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	}
	return nil
}

func (c *Controller) updateCorefile() error {
	// get cluster list ...
	clusterList, err := c.clusterclientset.List(metav1.ListOptions{})
	if err != nil {
		klog.Errorf("List Cluster failure: %s", err.Error())
		return err
	}
	// do sync coredns Corefile ...
	// remove duplicate values from clusterList, unuseful most likely, just for double-check :)
	seen := make(map[string]struct{})
	filterClusters := make([]v1.Cluster, 0)
	for _, cluster := range clusterList.Items {
		if cluster.Name == "" || len(cluster.Status.Addresses) == 0 || cluster.Status.Addresses[0].Host == "" {
			klog.V(4).Infof("cluster name: %s address: %s(invalid), ignores it ...", cluster.Name, cluster.Status.Addresses[0].Host)
			continue
		}
		if cluster.Status.Phase != v1.ClusterRunning {
			klog.V(4).Infof("cluster name: %s is not Running, ignores it ...", cluster.Name)
			continue
		}
		if _, ok := seen[cluster.Name]; !ok {
			if c.cfg.CoreDnsCfg.WildcardDomainSuffix != "" {
				cluster.Name = cluster.Name + "." + c.cfg.CoreDnsCfg.WildcardDomainSuffix
			}
			seen[cluster.Name] = struct{}{}
			filterClusters = append(filterClusters, cluster)
		}
	}
	// update coredns corefile and zones
	targetCorefileContent := ""
	flag := true
	for _, cluster := range filterClusters {
		klog.V(4).Infof("cluster domain info: %s => %s", cluster.Name, cluster.Status.Addresses[0].Host)
		// get target zone content
		targetZoneContent := strings.ReplaceAll(constants.CorednsZoneTemplate, "xxx", cluster.Name)
		targetZoneContent = strings.ReplaceAll(targetZoneContent, "ip", cluster.Status.Addresses[0].Host)
		// check exists
		zonePath := filepath.Join(c.cfg.CoreDnsCfg.ZonesDir, cluster.Name)
		if _, err := os.Stat(zonePath); err == nil { // exists
			zoneContent, err := ioutil.ReadFile(zonePath)
			if err == nil && string(zoneContent) == targetZoneContent {
				klog.V(4).Infof("zone: %s content ok", zonePath)
				// update target server block content
				targetCorefileContent += strings.ReplaceAll(constants.CorednsServerBlockTemplate, "xxx", cluster.Name)
			} else {
				klog.Warningf("zone: %s content update, ignores it this time", zonePath)
			}
			continue
		}
		flag = false
		err := ioutil.WriteFile(zonePath, []byte(targetZoneContent), 0644)
		if err != nil {
			klog.Errorf("write zone: %s failure: %v", zonePath, err)
			continue
		}
		klog.V(4).Infof("write new content to zone: %s successfully", zonePath)
		// update target server block content
		targetCorefileContent += strings.ReplaceAll(constants.CorednsServerBlockTemplate, "xxx", cluster.Name)
	}
	if flag {
		coreFileContent, err := ioutil.ReadFile(c.cfg.CoreDnsCfg.CorefilePath)
		if err == nil && string(coreFileContent) == targetCorefileContent {
			// remove unuseful cluster domains - keep numbers
			zones, _ := ioutil.ReadDir(c.cfg.CoreDnsCfg.ZonesDir)
			for _, zone := range zones {
				if _, ok := seen[zone.Name()]; !ok {
					klog.V(4).Infof("remove unuseful zone: %s", zone.Name())
					os.RemoveAll(filepath.Join(c.cfg.CoreDnsCfg.ZonesDir, zone.Name()))
				}
			}
			klog.V(4).Infof("========cluster info stay unchanged, there is no need to update coredns========")
			return nil
		}
	}
	// return directly when targetCorefileContent is empty
	if targetCorefileContent == "" {
		return nil
	}
	err = ioutil.WriteFile(c.cfg.CoreDnsCfg.CorefilePath, []byte(targetCorefileContent), 0644)
	if err != nil {
		klog.Errorf("write coredns corefile: %s failure: %v", c.cfg.CoreDnsCfg.CorefilePath, err)
	} else {
		// remove unuseful cluster domains - keep numbers
		zones, _ := ioutil.ReadDir(c.cfg.CoreDnsCfg.ZonesDir)
		for _, zone := range zones {
			if _, ok := seen[zone.Name()]; !ok {
				klog.V(4).Infof("remove unuseful zone: %s", zone.Name())
				os.RemoveAll(filepath.Join(c.cfg.CoreDnsCfg.ZonesDir, zone.Name()))
			}
		}
		klog.V(4).Infof("========update coredns corefile: %s successfully========", c.cfg.CoreDnsCfg.CorefilePath)
	}
	return err
}

// enqueueClusterAdd takes a Cluster resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Cluster.
func (c *Controller) enqueueClusterAdd(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	klog.V(4).Infof("Watch Cluster: %s Added ...", key)
	c.workqueue.Add(key)
}

// enqueueClusterUpdate takes a Cluster resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Cluster.
func (c *Controller) enqueueClusterUpdate(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	klog.V(4).Infof("Watch Cluster: %s Updated ...", key)
	c.workqueue.Add(key)
}

// enqueueClusterDelete takes a Cluster resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Cluster.
func (c *Controller) enqueueClusterDelete(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	klog.V(4).Infof("Watch Cluster: %s Deleted ...", key)
	c.workqueue.Add(key)
}
