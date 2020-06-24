package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"strconv"

	// "path/filepath"

	"time"

	// "github.com/golang/glog"
	log "github.com/sirupsen/logrus"
	
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// struct to store log
type logStruct struct {
	request       string
	statusCode    int
	status        string
	containerName string
	podName       string
	url           string
}

// struct to store info about pod, later used for logging
type podInfo struct {
	podName       string
	containerName string
	port          string
	ip            string
}

// struct to store the list of valid URLs in the pod
type urlStruct struct {
	url string
	pod *podInfo
}

// store kube config path
var (
	kubeConfigPath string
)

// annotation required to check if deployment is valid for injection
// status annotation tells if a sidecar has already been deployed
const (
	admissionWebhookAnnotationInjectKey = "sidecar-injector-webhook.rohit.in/inject"
	admissionWebhookAnnotationStatusKey = "sidecar-injector-webhook.rohit.in/status"
)

func init() {
	flag.StringVar(&kubeConfigPath, "kubeconfig", "", "Path to KUBECONFIG for running out of cluster. (Default: null)")
}

// returns kubernetes client
func getClientConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func GetKubernetesClient(kubeConfigPath string) (kubernetes.Interface, error) {

	config, err := getClientConfig(kubeConfigPath)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return clientset, nil
}

// function to add various policy details to the environment so that the sidecar can take appropriate actions
func setEnvironment(clientset kubernetes.Interface, pod *corev1.Pod) {
	metadata := &pod.ObjectMeta
	dps, err := clientset.AppsV1().Deployments(metadata.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Errorf("Error in retrieving deployments: %v", err)
	}

	// find current deployment from all deployments in namespace
	// deployment is checked for valid annotation
	for _, dp := range dps.Items {
		log.WithFields(log.Fields{"type": "info"}).Infof("Deployment: %s", dp.Name)
		_, ok := dp.Annotations[admissionWebhookAnnotationInjectKey]
		if ok {
			log.WithFields(log.Fields{"type": "info"}).Infof("Deployment has key: %s", dp.Name)
			setPodSelector, err := metav1.LabelSelectorAsSelector(dp.Spec.Selector)
			if err != nil {
				log.Error(err)
			}
			log.WithFields(log.Fields{"type": "info"}).Infof("Pod Selector: %s", setPodSelector)
			listPodOptions := metav1.ListOptions{LabelSelector: setPodSelector.String()}
			pods, err := clientset.CoreV1().Pods(metadata.Namespace).List(context.TODO(), listPodOptions)
			if err != nil {
				log.Errorf("Error in retrieving pods %v", err)
				continue
			}

			found := 0
			for _, dpPod := range pods.Items {
				// log.Infof("Current: %s, Admission: %s", dpPod.ObjectMeta.Name, p.Name)
				// log.Info(metadata)
				// log.Info(dpPod.ObjectMeta.Name)
				if dpPod.ObjectMeta.Name == metadata.Name {
					log.WithFields(log.Fields{"type": "info"}).Infof("Pod found: %s", metadata.Name)
					found = 1
					break
				}
			}

			// set the environment with different policy options
			if found == 1 {
				os.Setenv("Action", dp.Annotations["Action"])
				os.Setenv("GroupLabel", dp.Annotations["GroupLabel"])
				os.Setenv("LogFrequency", dp.Annotations["LogFrequency"])
				log.WithFields(log.Fields{"type": "info"}).Infof("Log frequency: %s %s", dp.Annotations["LogFrequency"], os.Getenv("LogFrequency"))
				break
			}
		}
	}
}

func main() {

	// for _, e := range os.Environ() {
	// 	pair := strings.SplitN(e, "=", 2)
	// 	log.Println(pair[0], ":", pair[1])
	// }

	// get kubernetes client
	clientset, err := GetKubernetesClient(kubeConfigPath)
	if err != nil {
		log.Error(err)
		return
	}
	// get hostname of the pod
	// necessary to search for the pod and then find the url for all the containers
	hostname := os.Getenv("HOSTNAME")
	log.WithFields(log.Fields{"type": "info"}).Infof("Hostname: %s", hostname)

	if hostname == "" {
		log.WithFields(log.Fields{"type": "info"}).Info("Unable to get host")
	}

	// list all pods
	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Error(err)
		return
	}

	var podIP, namespace string
	ports := make([]string, 0)
	pinfo := make([]podInfo, 0)

	for _, pod := range pods.Items {
		if pod.Name == hostname {
			namespace = pod.Namespace
			break
		}
	}

	// run loop at regular intervals
	for true {
		pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), hostname, metav1.GetOptions{})
		if err != nil {
			log.Error(err)
		}
		podCopy := pod.DeepCopy()

		// wait for pod status to be running, otherwise IP will not be assigned to it
		if podCopy.Status.Phase != "Running" {
			log.WithFields(log.Fields{"type": "info"}).Info(podCopy.Status.Phase)
			sleepDuration := 5
			time.Sleep(time.Duration(sleepDuration) * time.Second)

			pod, err = clientset.CoreV1().Pods(namespace).Get(context.TODO(), hostname, metav1.GetOptions{})
			if err != nil {
				log.Error(err)
			}
			podCopy = pod.DeepCopy()

		} else {
			// set environment
			if os.Getenv("Action") == "" {
				setEnvironment(clientset, podCopy)
			}

			// store pod IP, this is common for all containers
			podIP = podCopy.Status.PodIP
			for podIP == "" {
                log.WithFields(log.Fields{"type": "info"}).Infof("Waiting for IP: %s", podIP)
                sleepDuration := 5
                time.Sleep(time.Duration(sleepDuration) * time.Second)
                pod, err = clientset.CoreV1().Pods(namespace).Get(context.TODO(), hostname, metav1.GetOptions{})
                if err != nil {
                    log.Error(err)
                }
                podCopy = pod.DeepCopy()
                podIP = podCopy.Status.PodIP
			}
			// get port nos of containers within the pod
            // port = strconv.Itoa(int(podCopy.Spec.Containers[0].Name))
			for _, container := range podCopy.Spec.Containers {
				for _, port := range container.Ports {
					if container.Name == "monitor-sidecar" {
						continue
					}
					ports = append(ports, strconv.Itoa(int(port.ContainerPort)))

					// store podinfo in the struct
					var podinfo podInfo
					podinfo.ip = podIP
					podinfo.port = strconv.Itoa(int(port.ContainerPort))
					podinfo.podName = podCopy.ObjectMeta.Name
					podinfo.containerName = container.Name

					pinfo = append(pinfo, podinfo)

					log.WithFields(log.Fields{"type": "info"}).Infof("Pod Address: %s:%s", podIP, strconv.Itoa(int(port.ContainerPort)))
				}
			}
			// log.Info(podCopy.Status)
			// log.Infof("Pod Address: %s:%s", podIP, port)
			break
		}
	}

	log.WithFields(log.Fields{"type": "info"}).Info(pinfo)
	// urls := make([]string, len(ports))
	urls := make([]urlStruct, len(pinfo))

	// storing all URLs of the containers in the pod
	for i, data := range pinfo {
		urls[i].url = "http://" + data.ip + ":" + data.port
		urls[i].pod = &data
	}
	log.WithFields(log.Fields{"type": "info"}).Info(urls)

	// check health at regular intervals
	for true {
		for i, url := range urls {
			resp, err := http.Get(url.url)
			if err != nil {
				log.Error(err)
			} else {
				addLogData(url, pinfo[i], resp)
			}
		}
		// if interval not specified by policy, then default interval = 10 seconds
		sleepDuration, err := strconv.ParseInt(os.Getenv("LogFrequency"), 0, 64)
		if err != nil {
			sleepDuration = 10
		}
		time.Sleep(time.Duration(sleepDuration) * time.Second)
	}
}

// function that stores log data in proper format
// to differentiate pod logs from other logs:
// "type" = "liveness-check" has been set
// "type" is then used to differntiate between the logs
// it is necessary to add the "type" field whenever logging from the sidecar
func addLogData(url urlStruct, pod podInfo, resp *http.Response) {
	loginfo := logStruct{"GET", resp.StatusCode, http.StatusText(resp.StatusCode), url.pod.containerName, url.pod.podName, url.url}

	log.WithFields(log.Fields {
		"type": 		"liveness-check",
		"request": 		loginfo.request,
		"statuscode": 	loginfo.statusCode,
		"statusMsg": 	loginfo.status,
		"container": 	loginfo.containerName,
		"pod": 			loginfo.podName,
		"url": 			loginfo.url,
	}).Info("Liveness check performed")
}
