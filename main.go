package main

import (
	"flag"
	"path/filepath"
	"time"

	"podsmetric/server"

	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
)

var Client *server.ClientInfo

func init() {
	klog.InitFlags(nil)

	prometheus_server_addr := flag.String("prometheus_server_addr", "http://localhost:9090/", "prometheus server address")
	prometheus_timeout := flag.Duration("timeout", 5*time.Second, "prometheus query timeout")

	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	var prometheus_labels server.ArrayFlags
	flag.Var(&prometheus_labels, "prometheus_labels", "")

	flag.Parse()

	Client = server.NewClient(*prometheus_server_addr, prometheus_labels, kubeconfig, *prometheus_timeout)

	klog.V(2).Info(Client)
}

func main() {
	var pi server.PodsInfo
	pi.ListSamples(Client)
	pi.ConvertCSV()
}
