package controller

// import required packages
import (
    "os"
    "os/signal"
    "syscall"
    "flag"

    log "github.com/sirupsen/logrus"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
    "k8s.io/client-go/util/workqueue"
	"k8s.io/client-go/rest"

    api "github.com/kubernetes-cluster-monitor/controller/pkg/apis/k8s.crd.io/v1alpha1"
    clientSet "github.com/kubernetes-cluster-monitor/controller/pkg/client/clientset/versioned"
	informers "github.com/kubernetes-cluster-monitor/controller/pkg/client/informers/externalversions/k8s.crd.io/v1alpha1"
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

// returns kubernetes client and custom resource client
func getKubernetesClient(kubeConfigPath string) (kubernetes.Interface, clientSet.Interface) {
    // kubeConfigPath := "/home/devilblade/.kube/config"

    config, err := getClientConfig(kubeConfigPath)
    if err != nil {
        log.Fatalf("getClusterConfig: %v, path: %s", err, kubeConfigPath)
    }

    client, err := kubernetes.NewForConfig(config)
    if err != nil {
        log.Fatalf("getClusterConfig: %v", err)
    }

    myprojectClient, err := clientSet.NewForConfig(config)
    if err != nil {
        log.Fatalf("getClusterConfig: %v", err)
    }

    log.Info("Successfully constructed k8s client")
    return client, myprojectClient
}

func ControllerMain() {

    flag.Parse()
    // get clients
    client, myprojectClient := getKubernetesClient(kubeConfigPath)
    // retrieve custom resource informer which was generated from
    // the code generator and pass it the custom resource client
    // used for listing and watching custom resource across all namespaces
    informer := informers.NewPolicyDefinitionInformer(
		myprojectClient,
		meta_v1.NamespaceAll,
		0,
		cache.Indexers{},
	)

    logger := log.NewEntry(log.New())
    // create new workqueue to process requests
    queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
    // controller object
    controller := Controller {
        logger: logger,
        clientset: client,
        informer: informer,
        queue: queue,
        handler: &TestHandler{},
    }
    // event handlers for CRUD
    informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc: func(obj interface{}) {
            key, err := cache.MetaNamespaceKeyFunc(obj)
            log.Infof("Add PolicyDefinition : %s", key)
            if err == nil {
                queue.Add(key)
            }
        },
        UpdateFunc: func(oldObj, newObj interface{}) {
            // update function only adds policy details in case a new target has been identified
            // deleting previous targets isn't included
            // delete on update behavior isn't observed on default kubernetes configurations
            newDepl := newObj.(*api.PolicyDefinition)
            oldDepl := oldObj.(*api.PolicyDefinition)

            // periodic resync updates are sent, if both versions are same
            // then no need for update
            if newDepl.ResourceVersion == oldDepl.ResourceVersion {
                return
            }

            key, err := cache.MetaNamespaceKeyFunc(newObj)
            log.Infof("Update PolicyDefinition: %s", key)

            // oldKey, err := cache.MetaNamespaceKeyFunc(oldObj)
            // log.Infof("Old pod: %s", oldKey)
            if err == nil {
                queue.Add(key)
                // queue.Add(oldKey)
            }
        },
        DeleteFunc: func(obj interface{}) {
            // delete policy details before adding to the queue
            // once it has been added to queue, resource details are lost
            // hence deletion has to be taken care of before adding to queue
            key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
            log.Infof("Delete PolicyDefinition: %s", key)
            controller.handler.ObjectDeleted(obj, &controller)
            if err == nil {
                queue.Add(key)
            }
        },
    })


    stopCh := make(chan struct{})
    defer close(stopCh)
    // starting the controller
    go controller.Run(stopCh)

    sigTerm := make(chan os.Signal, 1)
    signal.Notify(sigTerm, syscall.SIGTERM)
    signal.Notify(sigTerm, syscall.SIGINT)
    <-sigTerm
}
