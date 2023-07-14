package server

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"k8s.io/klog/v2"
)

const (
	DefaultLabel string = "container != 'POD',container!=''"
)

type ClientInfo struct {
	Prometheus_server_addr string
	Prometheus_client      api.Client
	Prometheus_labels      ArrayFlags
	Kubeconfig             *string
	v1.API
	context.Context
	context.CancelFunc
	time.Duration
	PodSample []*model.Sample
}

func NewClient(prometheus_server_addr string, prometheus_labels ArrayFlags, kubeconfig *string, t time.Duration) *ClientInfo {
	//ctx, cancel := context.WithCancel(context.Background())
	prometheus_client, err := api.NewClient(api.Config{
		Address: prometheus_server_addr,
	})
	if err != nil {
		klog.Fatalf("Failed to connect to prometheus, err: %v", err)
	}

	return &ClientInfo{
		//Prometheus_server_addr: prometheus_server_addr,
		Prometheus_client: prometheus_client,
		Prometheus_labels: prometheus_labels,
		Kubeconfig:        kubeconfig,
		API:               v1.NewAPI(prometheus_client),
		//Context:                ctx,
		//CancelFunc:             cancel,
		Duration: t,
	}
}

type PodsInfo struct {
	PodInfo []PodInfo
}

type PodInfo struct {
	Namespace         string `json:"namespace"`
	Deployment        string `json:"deployment"`
	Container         string `json:"container"`
	Req_CPU           string `json:"req_cpu"`
	Req_MEM           string `json:"req_mem"`
	Lim_CPU           string `json:"lim_cpu"`
	Lim_MEM           string `json:"lim_mem"`
	Req_CPU_Suggested string `json:"req_cpu_suggested"`
	Req_MEM_Suggested string `json:"req_mem_suggested"`
	Lim_CPU_Suggested string `json:"lim_cpu_suggested"`
	Lim_MEM_Suggested string `json:"lim_mem_suggested"`
}

type Value interface {
	String() string
	Set(string) error
}

type ArrayFlags []string

func (i *ArrayFlags) String() string {
	return fmt.Sprint(*i)
}

func (i *ArrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}
