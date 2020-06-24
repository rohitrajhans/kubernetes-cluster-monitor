package main

import (
	"fmt"
	"net/http"
	"strconv"
	"encoding/json"
	"io/ioutil"
	"strings"
	"flag"
	"context"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// struct to store logs
// similar to the one `sidecar/sidecar.go`
type loginfo struct {
	Time 		string	`json:"time"`
	Level 		string	`json:"level"`
	Msg 		string	`json:"msg"`
	Container 	string	`json:"container"`
	Pod 		string	`json:"pod"`
	Request 	string	`json:"request"`
	Status 		string	`json:"status"`
	StatusCode 	string	`json:"statuscode"`
	Msgtype 	string	`json:"type"`
	Url			string	`json:"url"`
}

// filename where logs are stored on mounted volume
const (
	filename = "log-data/logfile.log"
)

// kubernetes config path
var (
	kubeConfigPath string
)

func init() {
	flag.StringVar(&kubeConfigPath, "kubeconfig", "", "Path to KUBECONFIG for running out of cluster. (Default: null)")
}

func getClientConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

// returns kubernetes client
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

func int32Ptr(i int32) *int32 { return &i }

// function to create new router
func newRouter() *mux.Router {

	r := mux.NewRouter()
	r.HandleFunc("/get_logs", retrieveLogs).Methods("GET")
	r.HandleFunc("/", responseMain).Methods("GET")
	r.HandleFunc("/action", takeAction).Methods("POST")
	
	return r
}

// handles the `action` endpoint
// receives `podname` as form data
// takes appropriate action on the pod and other similar pods based on the policy
func takeAction(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Errorf("Parseform() err: %v", err)
	}
	log.Info(r.Form)	
	podname := r.FormValue("podname")
	// get podname from form data
	log.Infof("Target Pod: %s", podname)

	resp := make([]string, 0)
	// get kubernetes client
	clientset, err := GetKubernetesClient(kubeConfigPath)
	if err != nil {
		log.Error(err)
		return
	}
	// retrieve deployments to search for pod
	dps, err := clientset.AppsV1().Deployments("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Errorf("Error in retrieving deployments: %v", err)
	}

	for _, dp := range dps.Items {

		log.Infof("Deployment has key: %s", dp.Name)
		setPodSelector, err := metav1.LabelSelectorAsSelector(dp.Spec.Selector)
		if err != nil {
			log.Error(err)
		}
		log.Infof("Pod Selector: %s", setPodSelector)
		listPodOptions := metav1.ListOptions{LabelSelector: setPodSelector.String()}
		pods, err := clientset.CoreV1().Pods("").List(context.TODO(), listPodOptions)
		if err != nil {
			log.Errorf("Error in retrieving pods %v", err)
			continue
		}

		found := 0
		var pod *corev1.Pod
		for _, dpPod := range pods.Items {
			log.Infof("%s %s", podname, dpPod.ObjectMeta.Name)
			if dpPod.ObjectMeta.Name == podname {
				log.Infof("Pod found: %s", podname)
				pod = &dpPod
				found = 1
				break
			}
		}
		// in case pod is found, take action on pod and other similar pods
		if found == 1 {
			// get policy information from deployment metadata
			action := dp.Annotations["Action"]
			grouplabel := dp.Annotations["GroupLabel"]
			if action == "" {
				action = "quarantine"
			}
			// create a labelMap for similar pods (specified by groupLabel)
			labelMap := pod.Labels
			log.Info("Pod labels: ", labelMap)
			labelSelector := ""

			var labels []string
			if grouplabel == "" {
				for k, _ := range labelMap {
					labels = append(labels, k)
				}
			} else {
				labels = strings.Split(grouplabel, "--")
			}

			// if action is quarantine, reduce the no of running pods of the deployment to 0
			// which in a way quarantines the deployment from the cluster
			if action == "quarantine" {
				for _, val := range labels {
					_, ok := labelMap[val]
					if ok {
						labelSelector = labelSelector + val + "=" +  labelMap[val] + ","
					}
				}
				labelSelector = strings.TrimRight(labelSelector, ",")
				log.Infof("Label Selector: %s", labelSelector)
				listSelector := metav1.ListOptions{LabelSelector: labelSelector}
				deploymentsClient := clientset.AppsV1().Deployments(pod.Namespace)
				targetDps, err := deploymentsClient.List(context.TODO(), listSelector)
				if err != nil {
					log.Error(err)
				}

				for _, targetDp := range targetDps.Items {
					log.Infof("Target Deployment: %s", targetDp.Name)
					result, err := deploymentsClient.Get(context.TODO(), targetDp.Name, metav1.GetOptions{})
					if err != nil {
						log.Errorf("Cannot get deployments %v", err)
					}

					// update the number of replicas to 0
					result.Spec.Replicas = int32Ptr(0)
					_, err = deploymentsClient.Update(context.TODO(), result, metav1.UpdateOptions{})
					if err != nil {
						log.Errorf("Error in updating: %v", err)
					}

					resp = append(resp, targetDp.Name)
					log.Infof("Deplyoment: %s has been quarantined", targetDp.Name)
				}
			}

			break
		}
	}

	// send back the list of pods which have been quarantined
	log.Infof("Action has been taken")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// endpoint to get logs
func retrieveLogs(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Errorf("Error in reading file: %s %v", filename, err)
	}

	// process log string
	text := string(data)
	fmt.Println("Printing original data 1: ", text)
	text = strings.Replace(text, "\\n", "", -1)
	text = strings.Replace(text, "\\", "", -1)
	text = strings.Replace(text, ",,", ",", -1)
	//text = strings.Replace(text, ",\n,", ",", -1)
	//text = strings.Replace(text, ",\\n,", ",", -1)
	//text = strings.TrimSpace(text)
	text = strings.TrimRight(text, "\n")
	text = strings.TrimRight(text, "\\n")
	text = strings.TrimRight(text, ",")
	text = strings.TrimRight(text, "\n")
	text = strings.TrimRight(text, "\\n")
	text = strings.TrimRight(text, ",")
	text = strings.TrimLeft(text, ",")
	text = "[" + text + "]"
	fmt.Println("Printing edited data 1: ", text)

	// store log string in array of loginfo struct
	// and then send it as a JSON response
	var logdata []loginfo
	err = json.Unmarshal([]byte(text), &logdata)
	if err != nil {
		//log.Info(text)
		log.Errorf("Error in conversion: %v", err)
	}

	// tail specifies the number of logs to get
	// if no number is specified, all logs are returned
	var val int
	valStr, ok := query["tail"]
	if !ok {
		val = 0
	} else {
		val, err = strconv.Atoi(valStr[0])
		if err != nil {
			log.Errorf("Error converting key to integer: %v", err)
			val = 0
		} else {
			val = len(logdata) - val
		}
	}
	
	if val < 0 {
		val = 0
	}
	
	// send log data
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(logdata[val:])

	log.Infof("Sent %d logs", len(logdata[val:]))
}

func responseMain(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "/get_logs")
}

func main() {
	// application deployed on port: 9080
	port := ":9080"
	r := newRouter()
	log.Infof("Now serving on Port %s\n", port)
	log.Fatal(http.ListenAndServe(port, r))
}
