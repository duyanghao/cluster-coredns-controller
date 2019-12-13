package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/glog"
)

// args
var (
	argConfigPath = flag.String("c", "/coredns-sync/examples/config.yml", "The configuration filepath for server.")
)

type cluster struct {
	Domain string `json:"name,omitempty"`
	Ip     string `json:"ip,omitempty"`
}

type clusterList []*cluster

type corednsSyncDaemon struct {
	cfg    *Config
	client *http.Client
}

// coredns template(server block&zone)
var (
	corednsZoneTemplate string = `
$ORIGIN xxx.
@       3600 IN SOA sns.dns.icann.org. noc.dns.icann.org. (
                                2017042745 ; serial
                                7200       ; refresh (2 hours)
                                3600       ; retry (1 hour)
                                1209600    ; expire (2 weeks)
                                3600       ; minimum (1 hour)
                                )

        3600 IN NS a.iana-servers.net.
        3600 IN NS b.iana-servers.net.

*.xxx.  IN A     ip
xxx.  IN A       ip
`

	corednsServerBlockTemplate string = `xxx:53 {
    file /etc/coredns/zones/xxx
    errors stdout  # show errors
    log stdout     # show query logs
}

`
)

func NewCorednsSyncDaemon(c *Config) (*corednsSyncDaemon, error) {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:        c.DaemonCfg.MaxIdleConns,
			MaxIdleConnsPerHost: c.DaemonCfg.MaxIdleConnsPerHost,
			IdleConnTimeout:     time.Duration(c.DaemonCfg.IdleConnTimeout) * time.Second,
		},
	}
	return &corednsSyncDaemon{
		cfg:    c,
		client: client,
	}, nil
}

func (csd *corednsSyncDaemon) getClusterDomainList() (clusterList, error) {
	// construct encoded endpoint
	Url, err := url.Parse(fmt.Sprintf("http://%s:%d", csd.cfg.ClusterServerCfg.Addr, csd.cfg.ClusterServerCfg.Port))
	if err != nil {
		return nil, err
	}
	Url.Path += csd.cfg.ClusterServerCfg.URI
	endpoint := Url.String()
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	// use httpClient to send request
	rsp, err := csd.client.Do(req)
	if err != nil {
		return nil, err
	}
	// close the connection to reuse it
	defer rsp.Body.Close()
	// check status code
	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get cluster domain list rsp error: %v", rsp)
	}
	// get cluster domain list by parsing response
	var clusterList clusterList
	err = json.NewDecoder(rsp.Body).Decode(&clusterList)
	if err != nil {
		return nil, err
	}
	return clusterList, nil
}

func (csd *corednsSyncDaemon) syncCoredns(clusters clusterList) error {
	// remove duplicate values from clusterList, unuseful most likely, just for double-check :)
	seen := make(map[string]struct{})
	filterClusters := make([]*cluster, 0)
	for _, cluster := range clusters {
		if _, ok := seen[cluster.Domain]; !ok {
			cluster.Domain = cluster.Domain + "." + csd.cfg.CoreDnsCfg.WildcardDomainSuffix
			seen[cluster.Domain] = struct{}{}
			filterClusters = append(filterClusters, cluster)
		}
	}
	// update coredns corefile and zones
	targetCorefileContent := ""
	flag := true
	for _, cluster := range filterClusters {
		glog.V(5).Infof("cluster domain info: %s => %s", cluster.Domain, cluster.Ip)
		// get target zone content
		targetZoneContent := strings.ReplaceAll(corednsZoneTemplate, "xxx", cluster.Domain)
		targetZoneContent = strings.ReplaceAll(targetZoneContent, "ip", cluster.Ip)
		// check exists
		zonePath := filepath.Join(csd.cfg.CoreDnsCfg.ZonesDir, cluster.Domain)
		if _, err := os.Stat(zonePath); err == nil { // exists
			zoneContent, err := ioutil.ReadFile(zonePath)
			if err == nil && string(zoneContent) == targetZoneContent {
				glog.V(5).Infof("zone: %s content ok", zonePath)
				// update target server block content
				targetCorefileContent += strings.ReplaceAll(corednsServerBlockTemplate, "xxx", cluster.Domain)
				continue
			}

		}
		flag = false
		err := ioutil.WriteFile(zonePath, []byte(targetZoneContent), 0644)
		if err != nil {
			glog.Errorf("write zone: %s failure: %v", zonePath, err)
			continue
		}
		glog.V(5).Infof("write new content to zone: %s successfully", zonePath)
		// update target server block content
		targetCorefileContent += strings.ReplaceAll(corednsServerBlockTemplate, "xxx", cluster.Domain)
	}
	if flag {
		coreFileContent, err := ioutil.ReadFile(csd.cfg.CoreDnsCfg.CorefilePath)
		if err == nil && string(coreFileContent) == targetCorefileContent {
			glog.V(5).Infof("========cluster info stay unchanged, there is no need to update coredns========")
			return nil
		}
	}
	err := ioutil.WriteFile(csd.cfg.CoreDnsCfg.CorefilePath, []byte(targetCorefileContent), 0644)
	if err != nil {
		glog.Errorf("write coredns corefile: %s failure: %v", csd.cfg.CoreDnsCfg.CorefilePath, err)

	} else {
		glog.V(5).Infof("========update coredns corefile: %s successfully========", csd.cfg.CoreDnsCfg.CorefilePath)
		// remove unuseful cluster domains - keep numbers
		zones, _ := ioutil.ReadDir(csd.cfg.CoreDnsCfg.ZonesDir)
		for _, zone := range zones {
			if _, ok := seen[zone.Name()]; !ok {
				glog.V(5).Infof("remove unuseful zone: %s", zone.Name())
				os.RemoveAll(filepath.Join(csd.cfg.CoreDnsCfg.ZonesDir, zone.Name()))
			}
		}

	}
	return err
}

func (csd *corednsSyncDaemon) run() {
	loop := 0
	for { // run forever
		// go to next loop whether or not succeed
		loop = (loop + 1) % 100
		time.Sleep(time.Duration(csd.cfg.DaemonCfg.SyncInterval) * time.Second)
		glog.V(5).Infof("\n\n=================== sync coredns loop: %d =====================", loop)
		// get cluster domain list
		clusterList, err := csd.getClusterDomainList()
		if err != nil {
			glog.Errorf("getClusterDomainList failure: %v", err)
			continue
		}
		glog.V(5).Infof("getClusterDomainList successfully")

		err = csd.syncCoredns(clusterList)
		if err != nil {
			glog.Errorf("syncCoredns failure: %v", err)
		} else {
			glog.V(5).Infof("syncCoredns successfully")
		}
	}
}

func main() {
	flag.Parse()
	// load configuration
	glog.V(5).Infof("Starting Loading configuration: %s.", *argConfigPath)
	cfg, err := LoadConfig(*argConfigPath)
	if err != nil {
		glog.Errorf("Loading configuration error: %v", err)
		return
	}
	glog.V(5).Infof("Loading configuration done.")
	// create corednsSync daemon
	csd, err := NewCorednsSyncDaemon(cfg)
	if err != nil {
		glog.Errorf("create corednsSync daemon failed: %v", err)
		return
	}
	// execute all workers
	glog.V(5).Infof("Starting coredns sync forever...")
	csd.run()
}
