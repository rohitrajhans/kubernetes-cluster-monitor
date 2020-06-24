package main

// required packages
import (
	// "fmt"
	// "bytes"
	"net/http"
	"net/url"
	// "strconv"
	// "encoding/json"
	"io/ioutil"
	// "strings"
	"flag"
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	// corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

    // v1alpha1 "github.com/cisco/CustomResource/src/pkg/apis/myproject/v1alpha1"
    myprojectclientset "github.com/cisco/CustomResource/src/pkg/client/clientset/versioned"
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

// returns kubernetes client, custom resource client
func getKubernetesClient(kubeConfigPath string) (kubernetes.Interface, myprojectclientset.Interface) {
    // kubeConfigPath := "/home/devilblade/.kube/config"

    config, err := getClientConfig(kubeConfigPath)
    if err != nil {
        log.Fatalf("getClusterConfig: %v, path: %s", err, kubeConfigPath)
    }

    client, err := kubernetes.NewForConfig(config)
    if err != nil {
        log.Fatalf("getClusterConfig: %v", err)
    }

    myprojectClient, err := myprojectclientset.NewForConfig(config)
    if err != nil {
        log.Fatalf("getClusterConfig: %v", err)
    }

    log.Info("Successfully constructed k8s client")
    return client, myprojectClient
}

// retrieve logs from the API endpoint and push information to external analysis tool
// details about external analysis tool are part of the policy configuration __
// __ which is stored in the Custom Resource of type Receiver 
func main() {

	flag.Parse()
	// get kubernetes client and custom resource client
	_, myprojectClient := getKubernetesClient(kubeConfigPath)
	// duration after which to retrieve logs
	sleepDuration := 120
	// 9080 is the port number for logs API
	port := "9080"
	// api endpoint for retrieving logs
	log_url := "http://localhost:" + port + "/get_logs"

	// retrieve log at regular intervals
	for true {
		// list all the custom resources
		// and retrieve details about the external analysis tools
		list, err := myprojectClient.SampleprojectV1alpha1().Receivers("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Errorf("Unable to get custom resource: %v", err)
			time.Sleep(time.Duration(sleepDuration) * time.Second)
			continue
		}

		// get the logs from the API endpoint
		resp, err := http.Get(log_url)
		if err != nil {
			log.Errorf("Error in getting logs: %v", err)
			continue
		}
		defer resp.Body.Close()

		var jsonstr string
		log.Info(resp.StatusCode)
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		jsonstr = string(body)
		log.Infof("Received %d bytes", len(jsonstr))
		
		for _, item := range list.Items {
			// retrieve external destination details
			addrs := item.Spec.Destination
			for _, addr := range addrs {
				ip := addr.IPAddress
				port := addr.Port
				endpoint := addr.Endpoint
				// push data to external server
				posturl := "http://" + ip + ":" + port + endpoint
				log.Infof("POST Url: %s", posturl)

				response, err := http.PostForm(posturl, url.Values{
					"data": {jsonstr}})
				// in case the external server is not working, do not send data
				if err != nil {
					log.Errorf("Post form error: %v", err)
					continue
				}

				defer response.Body.Close()
				respbody, err := ioutil.ReadAll(response.Body)
				if err != nil {
					log.Errorf("Error in reading body: %v", err)
				}
				log.Infof("Post response Body: %s", string(respbody))

				// req, err := http.NewRequest("POST", posturl, bytes.NewBuffer([]byte(jsonstr)))
				// req.Header.Set("Content-Type", "application/json")
				// client := &http.Client{}
				// client.Timeout = time.Second * 15
				// postresp, err := client.Do(req)
				// if err != nil {
				// 	log.Errorf("Connection Timed out: %v", err)
				// 	continue
				// }
				// defer postresp.Body.Close()

				// log.Info("Post Response Status:", postresp.Status)
				// respbody, _ := ioutil.ReadAll(postresp.Body)
				// log.Infof("Post Response Body: %s", string(respbody))
			}
		}
		
		time.Sleep(time.Duration(sleepDuration) * time.Second)
	}
}
