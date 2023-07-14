package server

import (
	"context"
	"encoding/csv"

	"fmt"
	"os"

	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/klog/v2"
)

func (c *ClientInfo) Generate(query string) {

	result, warnings, err := c.Query(c.Context, query, time.Now(), v1.WithTimeout(c.Duration))
	defer c.CancelFunc()
	if err != nil {
		klog.Fatalf("Querying prometheus error: %v", err)
	}
	if len(warnings) > 0 {
		klog.Warningf("Warnings: %v", warnings)
	}
	klog.V(2).Infof("Result: %v", result)

	switch v := result.(type) {
	case model.Vector:
		// 处理向量类型值
		c.PodSample = v
		/*for _, sample := range v {
			fmt.Printf("Namespace: %v, Pod: %v, Container: %v, Value: %f, Timestamp: %s\n",
				//sample.Metric.String(),
				sample.Metric["namespace"],
				sample.Metric["pod"],
				sample.Metric["container"],
				sample.Value,
				sample.Timestamp.String(),
			)
		}*/
	case model.Matrix:
		// 处理矩阵类型值
		for _, series := range v {
			for _, sample := range series.Values {
				fmt.Printf("Namespace: %v, Pod: %v, Container: %v, Value: %f, Timestamp: %s\n",
					//series.Metric.String(),
					series.Metric["namespace"],
					series.Metric["pod"],
					series.Metric["container"],
					sample.Value,
					sample.Timestamp.String(),
				)
			}
		}
	default:
		fmt.Println("Unknown value type")
	}
}

func (p *PodsInfo) ConvertCSV() {
	file, err := os.Create("output.csv")
	if err != nil {
		klog.Fatal(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"Namespace", "Deployment", "Container", "CPU Requests", "MEM Requests"}
	err = writer.Write(header)
	if err != nil {
		klog.Fatal(err)
	}

	for _, v := range p.PodInfo {
		row := []string{string(v.Namespace), string(v.Deployment), string(v.Container), v.Req_CPU + " → " + v.Req_CPU_Suggested, v.Req_MEM + " → " + v.Req_MEM_Suggested}
		err = writer.Write(row)
		if err != nil {
			klog.Fatal(err)
		}
	}

}

func (c *ClientInfo) Execute(query string) string {
	ctx, cancel := context.WithCancel(context.Background())
	c.Context = ctx
	c.CancelFunc = cancel

	result, warnings, err := c.Query(c.Context, query, time.Now(), v1.WithTimeout(c.Duration))
	defer c.CancelFunc()

	if err != nil {
		klog.Fatalf("Querying prometheus error: %v", err)
	}
	if len(warnings) > 0 {
		klog.Warningf("Warnings: %v", warnings)
	}
	klog.V(2).Infof("Result: %v", result)

	switch v := result.(type) {
	case model.Vector:
		// 处理向量类型值
		for _, sample := range v {
			return sample.Value.String()
		}
	case model.Matrix:
		// 处理矩阵类型值
		for _, series := range v {
			for _, sample := range series.Values {
				return sample.Value.String()
			}
		}
	default:
		fmt.Println("Unknown value type")
	}
	return ""
}

func (p *PodsInfo) ListSamples(c *ClientInfo) {
	config, err := clientcmd.BuildConfigFromFlags("", *c.Kubeconfig)
	if err != nil {
		klog.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
	}

	deploymentsClient := clientset.AppsV1().Deployments(apiv1.NamespaceAll)

	list, err := deploymentsClient.List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		klog.Fatal(err)
	}

	f := func(arg string) string {
		if arg == "0" {
			return "None"
		} else {
			return arg
		}
	}

	for _, d := range list.Items {
		if *d.Spec.Replicas == 0 {
			continue
		}

		req_cpu_suggested := c.Execute(NewreqCPUsuggestedquery(d.Spec.Template.Spec.Containers[0].Name, c.Prometheus_labels))
		if len(req_cpu_suggested) <= 0 {
			req_cpu_suggested = "None"
		} else {
			req_cpu_suggested += "m"
		}

		req_mem_suggested := c.Execute(NewreqMEMsuggestedquery(d.Spec.Template.Spec.Containers[0].Name, c.Prometheus_labels))
		if len(req_mem_suggested) <= 0 {
			req_mem_suggested = "None"
		} else {
			req_mem_suggested += "Mi"
		}

		pd := PodInfo{
			Namespace:         d.Namespace,
			Deployment:        d.Name,
			Container:         d.Spec.Template.Spec.Containers[0].Name,
			Req_CPU:           f(d.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String()),
			Req_MEM:           f(d.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String()),
			Lim_CPU:           d.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().String(),
			Lim_MEM:           d.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String(),
			Req_CPU_Suggested: req_cpu_suggested,
			Req_MEM_Suggested: req_mem_suggested,
		}
		p.PodInfo = append(p.PodInfo, pd)
	}
}

func NewreqCPUsuggestedquery(container string, labels ArrayFlags) string {
	var l string
	for i := 0; i < len(labels); i++ {
		l += labels[i] + ","
	}
	return fmt.Sprintf("floor(avg by (container) ((quantile_over_time(0.90,(irate(container_cpu_usage_seconds_total{%s,container=~'%s',%s}[1m])[7d:1m]))) * 1000))", DefaultLabel, container, l)
}

func NewreqMEMsuggestedquery(container string, labels ArrayFlags) string {
	var l string
	for i := 0; i < len(labels); i++ {
		l += labels[i] + ","
	}
	return fmt.Sprintf("floor(avg by (container) (quantile_over_time(0.90,container_memory_working_set_bytes{image!='',container=~'%s',%s}[7d:1m]) / 1024/1024))", container, l)
}
