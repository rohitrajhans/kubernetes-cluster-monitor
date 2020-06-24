package main

// import relevant packages
import (
	"bytes"
	"context"
	"flag"
	"os"
	"time"
	//"fmt"
	//"strconv"
	// "io/ioutil"
	"bufio"
	"strings"
	"encoding/json"

	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// struct that stores log info
// similar to `sidecar/sidecar.go`
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

// `type=liveness-check` refers to pod health logs
// filename to store logs in mounted volume
const (
	logKey = "liveness-check"
	filename = "log-data/logfile.log"
)

// check sidecar injection status, it is stored as annotation in the pod
const (
	injectedStatusKey   = "sidecar-injector-webhook.rohit.in/status"
	injectedStatusValue = "injected"
)

// kubernetes configuration path
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
// function to process a line of log and add it to the map
// logs are an array of map[string] string
// individual objects of the map are key value pairs as found in the logs
// example: m["container"] = "title"
func processLine(str string) map[string]string {
	// map object to store a single log
	m := make(map[string]string)
	s := str
	//log.Info(s)
	// string processing to convert it to appropriate format
	for i:=strings.Index(s, "="); i>=0; i=strings.Index(s, "=") {
		var key, value string
		key = s[:i]
		key = strings.TrimSpace(key)
		s = s[i+1:]
		//log.Infof("KEY: %s, --> %s", key, s)
		
		if string(s[0]) == "\"" {
			s = s[1:]
			//log.Infof("DEBUG: %s ---- %d", s, len(s))
			if j := strings.Index(s, "\""); j>=0 {
				//log.Infof("Error :%d", j)
				value = s[:j]
				s = s[j+1:]	
			} else {
				//log.Infof("Error here:%d", j)
				value = s[:j-1]
			}
		} else {
			j := strings.Index(s, " ")
			if j < 0 {
				value = s
			} else {
				value = s[:j]
				s = s[j+1:]
			}
		}
		//log.Infof("VALUE: %s, --> %s", value, s)
		m[key] = value
	}
	
	// return the single log map
	//log.Info(m)
	return m
}

// adds the single map to the array of loginfo objects
func addToLog(m map[string]string, logs *[]loginfo) {
	if m["type"] != logKey {
		return
	}

	var newlog loginfo

	for k, v := range m {
		//log.Info(k, v)
		switch k {
		case "time":
			newlog.Time = v
		case "msg":
			newlog.Msg = v
		case "level":
			newlog.Level = v
		case "container":
			newlog.Container = v
		case "request":
			newlog.Request = v
		case "statusMsg":
			newlog.Status = v
		case "statuscode":
			newlog.StatusCode = v
		case "pod":
			newlog.Pod = v
		case "type":
			newlog.Msgtype = v
		case "url":
			newlog.Url = v
		}
	}

	*logs = append(*logs, newlog)
	return
}

// adds the single map to the array of maps
// this is not used
// struct is prefered to store logs as it is convenient to send as a JSON Response
func addToLogMap(m map[string]string, logs *[]map[string]string) {
	if m["type"] != logKey {
		return
	} else {
		*logs = append(*logs, m)
	}
	return
}

func main() {
	
	// logs collected after every `logDuration` seconds
	logDuration := 30
	// get kubernetes client
	clientset, err := GetKubernetesClient(kubeConfigPath)
	if err != nil {
		log.Error(err)
		return
	}
	
	// create file, only if it doesn't exist
	// otherwise append to it
	fileExistsBool := 0
	
	if _, err = os.Stat(filename); err == nil {	
		fileExistsBool = 1
	 } else if os.IsNotExist(err) {
	 	fileExistsBool = 0
	 } else {
	 	log.Info(err)
	 	return
	 }

	file, err := os.OpenFile(filename,  os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
	if err != nil {
        log.Errorf("Error creating log file: %v", err)
		return
	}
	defer file.Close()
	// log.Info(f.Readdir(15))

	// repeat loop at regular intervals
	for true {
		logs := make([]loginfo, 0)
		pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Error(err)
			return
		}

		// range through all the pods in the cluster
		for _, pod := range pods.Items {
			_, ok := pod.ObjectMeta.Annotations[injectedStatusKey]
			// if sidecar is injected in the pod, then retrieve logs from the sidecar
			if ok && (pod.ObjectMeta.Annotations[injectedStatusKey] == injectedStatusValue) {

				var logsFrom int64 = int64(logDuration)
				// logs of the last logDuration seconds are retrieved from the sidecar
				podLogOptions := corev1.PodLogOptions{Container: "monitor-sidecar", SinceSeconds: &logsFrom}
				if fileExistsBool == 0 {
					podLogOptions = corev1.PodLogOptions{Container: "monitor-sidecar"}
					fileExistsBool = 1
				}
				req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOptions)
				podLogs, err := req.Stream(context.TODO())
				if err != nil {
					log.Errorf("Error in retrieving logs: %v", err)
					continue
				}

				defer podLogs.Close()
				buf := new(bytes.Buffer)
				buf.ReadFrom(podLogs)
				logStr := buf.String()

				// logs added to struct after some string processing
				scanner := bufio.NewScanner(strings.NewReader(logStr))
				for scanner.Scan() {
					text := scanner.Text()
			    	stext := strings.Replace(text, "\\", "", -1)	
					m := processLine(stext)
					//log.Info(m)
					addToLog(m, &logs)
				}
			}
		}

		logData, err := json.MarshalIndent(logs, "", "")
		if err != nil {
			log.Infof("Cannot convert to JSON: %v", err)
		}
		//log.Info(string(logData))
		
		// store log to file
		logString := string(logData)
		// log.Info(logString)
		logString = strings.TrimRight(logString, "]")
		logString = strings.TrimLeft(logString, "[")
		logString = strings.TrimSpace(logString)
		log.Info(logString)
		// do not write to file if logString is empty
		if logString != "" {
			logString = logString + "," + string("\n")
		}
		n, err := file.WriteString(logString)
		log.Infof(": %d Bytes Written", n)
		if err != nil {
			log.Infof("Cannot write to file: %v", err)
		}
		time.Sleep(time.Duration(logDuration) * time.Second)
	}
}
