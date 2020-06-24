package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/kubernetes"
)

// docs on admission controller:
// https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/

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

// returns kubernetes client config
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

func main() {
	var parameters WhSvrParameters

	// get command line parameters
	flag.IntVar(&parameters.port, "port", 8443, "Webhook server port.")
	flag.StringVar(&parameters.certFile, "tlsCertFile", "/etc/webhook/certs/cert.pem", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&parameters.keyFile, "tlsKeyFile", "/etc/webhook/certs/key.pem", "File containing the x509 private key to --tlsCertFile.")
	flag.StringVar(&parameters.sidecarCfgFile, "sidecarCfgFile", "/etc/webhook/config/sidecarconfig.yaml", "File containing the mutation configuration.")
	flag.Parse()

	glog.Info("Loading Configuration:\n")
	// get kubernetes client
	clientset, err := GetKubernetesClient(kubeConfigPath)
	if err != nil {
		glog.Errorf("Error loading clientset: %v, err")
	}
	// load configuration for sidecar
	// this is then added to the appropriate pod template
	sidecarConfig, err := loadConfig(parameters.sidecarCfgFile)
	if err != nil {
		glog.Info("Failed to load configuration\n")
		glog.Errorf("Failed to load configuration: %v", err)
	}

	glog.Info("Successfully loaded configuration\n")

	// certs needed for TLS communication between apiserver and webhook server
	pair, err := tls.LoadX509KeyPair(parameters.certFile, parameters.keyFile)
	if err != nil {
		glog.Errorf("Failed to load key pair: %v", err)
	}

	whsvr := &WebhookServer{
		sidecarConfig: sidecarConfig,
		server: &http.Server{
			Addr:      fmt.Sprintf(":%v", parameters.port),
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		},
		clientset: clientset,
	}

	// define http server and server handler
	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", whsvr.serve)
	whsvr.server.Handler = mux

	glog.Infoln("Listening on port:", parameters.port)

	// start webhook server in new rountine
	go func() {
		if err := whsvr.server.ListenAndServeTLS("", ""); err != nil {
			glog.Errorf("Failed to listen and serve webhook server: %v", err)
		}
	}()

	// listening OS shutdown singal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	glog.Infof("Got OS shutdown signal, shutting down webhook server gracefully...")
	whsvr.server.Shutdown(context.Background())
}
