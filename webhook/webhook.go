package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/kubernetes/pkg/apis/core/v1"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()

	// (https://github.com/kubernetes/kubernetes/issues/57982)
	defaulter = runtime.ObjectDefaulter(runtimeScheme)
)

// list of namespaces for which mutation would be rejected
var ignoredNamespaces = []string{
	metav1.NamespaceSystem,
	metav1.NamespacePublic,
}

// annotations that are checked before mutating pod template
// only mutate request if `admissionWebhookAnnotationInjectKey=yes`
// also update status of mutation
const (
	admissionWebhookAnnotationInjectKey = "sidecar-injector-webhook.rohit.in/inject"
	admissionWebhookAnnotationStatusKey = "sidecar-injector-webhook.rohit.in/status"
)

type WebhookServer struct {
	sidecarConfig *Config
	server        *http.Server
	clientset     kubernetes.Interface
}

// Webhook Server parameters
type WhSvrParameters struct {
	port           int    // webhook server port
	certFile       string // path to the x509 certificate for https
	keyFile        string // path to the x509 private key matching `CertFile`
	sidecarCfgFile string // path to sidecar injector configuration file
}

// config struct for sidecar container
type Config struct {
	Containers []corev1.Container `yaml:"containers"`
	Volumes    []corev1.Volume    `yaml:"volumes"`
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func init() {
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admissionregistrationv1beta1.AddToScheme(runtimeScheme)
	// defaulting with webhooks:
	_ = v1.AddToScheme(runtimeScheme)
}

func applyDefaultsWorkaround(containers []corev1.Container, volumes []corev1.Volume) {
	defaulter.Default(&corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: containers,
			Volumes:    volumes,
		},
	})
}

func loadConfig(configFile string) (*Config, error) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	glog.Infof("New configuration: sha256sum %x", sha256.Sum256(data))

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Check whether the target resoure needs to be mutated
func mutationRequired(ignoredList []string, pod *corev1.Pod, clientset kubernetes.Interface) bool {

	metadata := &pod.ObjectMeta
	// skip special kubernetes system namespaces
	for _, namespace := range ignoredList {
		if metadata.Namespace == namespace {
			glog.Infof("Skip mutation for %v for it's in special namespace:%v", metadata.Name, metadata.Namespace)
			return false
		}
	}
	// retrieve annotations of pod
	annotations := metadata.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	status := annotations[admissionWebhookAnnotationStatusKey]

	// determine whether to perform mutation based on annotation for the target resource
	// if status is injected, then no need to mutate again
	var required bool
	if strings.ToLower(status) == "injected" {
		required = false
		return required
	} else {
		switch strings.ToLower(annotations[admissionWebhookAnnotationInjectKey]) {
		default:
			required = false
		case "y", "yes", "true", "on":
			required = true
		}
	}

	// if pod doesn't havve annotation, check the corresponding deployment for the annotation
	setSelector := labels.Set(pod.Labels)
	glog.Infof("Pod selector: %s", setSelector.AsSelector().String())
	// retrieve the deployment of the pod
	listOptions := metav1.ListOptions{LabelSelector: setSelector.AsSelector().String()}
	pods, err := clientset.CoreV1().Pods(pod.Namespace).List(context.TODO(), listOptions)
	if err != nil {
		glog.Error(err)
	}

	for _, p := range pods.Items {
		dps, err := clientset.AppsV1().Deployments(metadata.Namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			glog.Errorf("Error in retrieving deployments: %v", err)
			return required
		}
		for _, dp := range dps.Items {
			glog.Infof("Deployment: %s", dp.Name)
			_, ok := dp.Annotations[admissionWebhookAnnotationInjectKey]
			if ok {
				glog.Infof("Deployment has key: %s", dp.Name)
				setPodSelector, err := metav1.LabelSelectorAsSelector(dp.Spec.Selector)
				if err != nil {
					glog.Error(err)
				}
				glog.Infof("Pod Selector: %s", setPodSelector)
				listPodOptions := metav1.ListOptions{LabelSelector: setPodSelector.String()}
				pods, err := clientset.CoreV1().Pods(metadata.Namespace).List(context.TODO(), listPodOptions)
				if err != nil {
					glog.Errorf("Error in retrieving pods %v", err)
					continue
				}

				for _, dpPod := range pods.Items {
					glog.Infof("Current: %s, Admission: %s", dpPod.ObjectMeta.Name, p.Name)
					// glog.Info(metadata)
					// glog.Info(dpPod.ObjectMeta.Name)
					if dpPod.ObjectMeta.Name == p.Name {
						glog.Infof("Pod qualifies for mutation: %s", metadata.Name)
						required = true
						break
					}
				}
				if required == true {
					break
				}
			}
		}
		if required == true {
			break
		}
	}

	// setSelector := labels.Set(pod.Labels)
	// ignoredLabels := [1]string{"pod-template-hash"}
	// for _, label := range ignoredLabels {
	// 	_, ok := setSelector[label]
	// 	if ok {
	// 		delete(setSelector, label)
	// 	}
	// }

	// glog.Infof("Deployment selector: %s", setSelector.AsSelector().String())

	// listOptions := metav1.ListOptions{LabelSelector: setSelector.AsSelector().String()}
	// dps, err := clientset.AppsV1().Deployments(pod.Namespace).List(context.TODO(), listOptions)
	// if err != nil {
	// 	glog.Errorf("Error in retrieving deployments: %v", err)
	// 	return required
	// }

	// for _, dp := range dps.Items {
	// 	dpCopy := dp.DeepCopy()
	// 	glog.Infof("Deployment: %s", dpCopy.Name)
	// 	_, ok := dpCopy.Annotations[admissionWebhookAnnotationInjectKey]
	// 	if ok {
	// 		glog.Info(dpCopy.Annotations[admissionWebhookAnnotationInjectKey])
	// 		switch strings.ToLower(dpCopy.Annotations[admissionWebhookAnnotationInjectKey]) {
	// 		default:
	// 			required = false
	// 		case "y", "yes", "true", "on":
	// 			required = true
	// 		}
	// 	}

	// 	if required == true {
	// 		break
	// 	}
	// }

	glog.Infof("Mutation policy for %v/%v: status: %q required:%v", metadata.Namespace, metadata.Name, status, required)
	return required
}

// adds the container and returns the patch
func addContainer(target, added []corev1.Container, basePath string) (patch []patchOperation) {
	first := len(target) == 0
	var value interface{}
	for _, add := range added {
		value = add
		path := basePath
		if first {
			first = false
			value = []corev1.Container{add}
		} else {
			path = path + "/-"
		}
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}

// adds the volume for the sidecar and returns the patch
func addVolume(target, added []corev1.Volume, basePath string) (patch []patchOperation) {
	first := len(target) == 0
	var value interface{}
	for _, add := range added {
		value = add
		path := basePath
		if first {
			first = false
			value = []corev1.Volume{add}
		} else {
			path = path + "/-"
		}
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}

func updateAnnotation(target map[string]string, added map[string]string) (patch []patchOperation) {
	for key, value := range added {
		if target == nil || target[key] == "" {
			target = map[string]string{}
			patch = append(patch, patchOperation{
				Op:   "add",
				Path: "/metadata/annotations",
				Value: map[string]string{
					key: value,
				},
			})
		} else {
			patch = append(patch, patchOperation{
				Op:    "replace",
				Path:  "/metadata/annotations/" + key,
				Value: value,
			})
		}
	}
	return patch
}

// create mutation patch for resoures
func createPatch(pod *corev1.Pod, sidecarConfig *Config, annotations map[string]string) ([]byte, error) {
	var patch []patchOperation

	patch = append(patch, addContainer(pod.Spec.Containers, sidecarConfig.Containers, "/spec/containers")...)
	patch = append(patch, addVolume(pod.Spec.Volumes, sidecarConfig.Volumes, "/spec/volumes")...)
	patch = append(patch, updateAnnotation(pod.Annotations, annotations)...)

	return json.Marshal(patch)
}

// main mutation process
func (whsvr *WebhookServer) mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		glog.Errorf("Could not unmarshal raw object: %v", err)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	glog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, pod.Name, req.UID, req.Operation, req.UserInfo)

	// determine whether to perform mutation
	if !mutationRequired(ignoredNamespaces, &pod, whsvr.clientset) {
		glog.Infof("Skipping mutation for %s/%s due to policy check", pod.Namespace, pod.Name)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	applyDefaultsWorkaround(whsvr.sidecarConfig.Containers, whsvr.sidecarConfig.Volumes)
	annotations := map[string]string{admissionWebhookAnnotationStatusKey: "injected"}
	patchBytes, err := createPatch(&pod, whsvr.sidecarConfig, annotations)
	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	glog.Infof("AdmissionResponse: patch=%v\n", string(patchBytes))
	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

// Serve method for webhook server
func (whsvr *WebhookServer) serve(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		glog.Error("empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		glog.Errorf("Content-Type=%s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		glog.Errorf("Can't decode body: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		admissionResponse = whsvr.mutate(&ar)
	}

	admissionReview := v1beta1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	resp, err := json.Marshal(admissionReview)
	if err != nil {
		glog.Errorf("Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	glog.Infof("Ready to write reponse ...")
	if _, err := w.Write(resp); err != nil {
		glog.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}
