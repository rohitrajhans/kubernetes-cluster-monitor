package controller

import (
metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"context"
	"encoding/json"
	"strings"
	"strconv"
	"github.com/mattbaird/jsonpatch"
	"k8s.io/apimachinery/pkg/types"

	// core_v1 "k8s.io/api/core/v1"
    api "github.com/kubernetes-cluster-monitor/controller/pkg/apis/k8s.crd.io/v1alpha1"
	log "github.com/sirupsen/logrus"
)

// metadata to be added to deployments for sidecar injection
// status is used to check if a sidecar has already been injected
const (
	annotationKey       = "sidecar-injector-webhook/inject"
	annotationValue     = "true"
	injectedStatusKey   = "sidecar-injector-webhook/status"
	injectedStatusValue = "injected"
)

// handler interface to handle CRUD
type Handler interface {
	Init() error
	ObjectCreated(obj interface{}, c *Controller)
	ObjectDeleted(obj interface{}, c *Controller)
	ObjectUpdated(objOld, objNew interface{}, c *Controller)
}

type TestHandler struct{}

func (t *TestHandler) Init() error {
	log.Info("TestHandler.Init")
	return nil

}

// function converts the labelSelector string to appropriate format
// for retrieving the labeled resources
func RetrieveExpression(obj interface{}) string {

	polDef := obj.(*api.PolicyDefinition).DeepCopy()

	labelMap := polDef.Spec.LabelSelector.Labels
	expression := ""

	if len(labelMap) != 0 {
		for k, v := range labelMap {
			expression = expression + k + "=" + v + ","
		}
		expression = strings.TrimRight(expression, ",")
	}

	mExpr := polDef.Spec.LabelSelector.Expression

	if len(mExpr) != 0 {
		var mStr string

		if expression != "" {
			mStr = ","
		}

		for _, v := range mExpr {
			mStr = mStr + v.Key + " " + v.Operator + " ("
			for _, val := range v.Values {
				mStr = mStr + val + ","
			}
			mStr = strings.TrimRight(mStr, ",")
			mStr = mStr + "),"
		}
		mStr = strings.TrimRight(mStr, ",")

		expression = expression + mStr
	}

	return expression
}

// handles object creation
func (t *TestHandler) ObjectCreated(obj interface{}, c *Controller) {

	log.Info("TestHandler.ObjectCreated")

	polDef := obj.(*api.PolicyDefinition).DeepCopy()
	// retrieve label selector from the object
	expression := RetrieveExpression(obj)
	c.logger.Infof("Selector expression: `%s`", expression)

	// nspaces stores the list of namespaces to define the scope of custom resource
	// in case no namespace has been specified, default namespace is used
	var nspaces = make([]string, len(polDef.Spec.Namespace)+1)
	if len(polDef.Spec.Namespace) != 0 {
		for i, val := range polDef.Spec.Namespace {
			c.logger.Infof("Namespace: %s", val)
			nspaces[i] = val
		}
	} else {
		c.logger.Infof("No namespace mentioned in configuration. Default namespace used")
		nspaces[0] = "default"
	}

	// converts group labels to label selector
	var groupLabels = make([]string, len(polDef.Spec.GroupLabel))
	groupLabelString := ""
	for i, val := range polDef.Spec.GroupLabel {
		groupLabels[i] = val
		groupLabelString = groupLabelString + val + "--"
	}
	groupLabelString = strings.TrimRight(groupLabelString, "-")
	var logFreq string
	if polDef.Spec.LogSpec.LogFrequency == 0 {
		logFreq = "10"
	} else {
		logFreq = strconv.Itoa(polDef.Spec.LogSpec.LogFrequency)
	}
	// store action to be taken
	action := polDef.Spec.Action

	// iterate over the specified namepsaces
	for _, ns := range nspaces {

		if ns == "" {
			break
		}
		listOptions := metav1.ListOptions{LabelSelector: expression}
		// retrieve deployments that match the selector expression
		// podClient := c.clientset.CoreV1().Pods(ns)
		deployClient := c.clientset.AppsV1().Deployments(ns)

		dps, err := deployClient.List(context.TODO(), listOptions)
		if err != nil {
			c.logger.Errorf("Error in retrieving deployments: %v", err)
			return
		}

		// pods, err := podClient.List(context.TODO(), listOptions)
		// if err != nil {
		// 	c.logger.Errorf("Error in listing pods: %v", err)
		// 	return
		// }
		// loop over through the deployments to modify metadata for sidecar injection
		for _, dp := range dps.Items {
			// metadata is added through a deployment patch
			// it is assumed that applications are deployed as Kind: Deployment
			// most Kubernetes applications are deployed as Kind: Deployment
			dpCopy := dp.DeepCopy()
			c.logger.Infof("Deployment Name: %s", dpCopy.Name)

			oJson, err := json.Marshal(dpCopy)
			if err != nil {
				c.logger.Errorf("Error in creating json: %v", err)
				return
			}

			if dpCopy.Annotations == nil {
				dpCopy.Annotations = map[string]string{}
			} else if dpCopy.Annotations[annotationKey] == annotationValue {
				c.logger.Infof("Deployment already annotated")
				continue
			}
			// adding required metadata
			dpCopy.Annotations[annotationKey] = annotationValue
			dpCopy.Annotations["GroupLabel"] = groupLabelString
			dpCopy.Annotations["Action"] = action
			dpCopy.Annotations["LogFrequency"] = logFreq

			mJson, err := json.Marshal(dpCopy)
			if err != nil {
				c.logger.Errorf("Error in creating json: %v", err)
				return
			}
			// create the patch
			patch, err := jsonpatch.CreatePatch(oJson, mJson)
			if err != nil {
				c.logger.Errorf("Error in creating patch: %v", err)
				return
			}

			pb, err := json.MarshalIndent(patch, "", " ")
			if err != nil {
				c.logger.Errorf("%v", err)
				return
			}

			// c.logger.Infof("Patch: %s", string(pb))
			// apply patch to the deployment
			final, err := c.clientset.AppsV1().Deployments(ns).Patch(context.TODO(), dpCopy.Name, types.JSONPatchType, pb, metav1.PatchOptions{})
			if err != nil {
				c.logger.Errorf("Error in applying patch: %v", err)
				return
			}

			_, err = json.MarshalIndent(final, "", " ")
			if err != nil {
				c.logger.Errorf("%v", err)
				return
			}

			c.logger.Infof("Successfully applied patch")
			// now for the sidecar injection to take place, restart the pods in the deployment
			// admission controller only works on creation/ update requests
			// so it is necessary to restart pod for changes to take effect
			podSelector, err := metav1.LabelSelectorAsSelector(dpCopy.Spec.Selector)
			if err != nil {
				c.logger.Error(err)
			}

			listPodOptions := metav1.ListOptions{LabelSelector: podSelector.String()}
			podClient := c.clientset.CoreV1().Pods(ns)
			pods, err := podClient.List(context.TODO(), listPodOptions)
			if err != nil {
				c.logger.Errorf("Error in retrieving pods: %v", err)
			}

			for _, pod := range pods.Items {
				podCopy := pod.DeepCopy()
				// do not restart pod in case the sidecar is already injected
				if podCopy.ObjectMeta.Annotations[injectedStatusKey] == injectedStatusValue {
					c.logger.Infof("Sidecar is already injected in pod: %s", podCopy.Name)
					continue
				}

				err = podClient.Delete(context.TODO(), podCopy.Name, metav1.DeleteOptions{})
				if err != nil {
					c.logger.Errorf("Error in restarting pod %s: %v", podCopy.Name, err)
					return
				}

				c.logger.Infof("Restarting Pod: %s", podCopy.Name)
			}
		}
	}
}

// function to handle custom resource deletion
func (t *TestHandler) ObjectDeleted(obj interface{}, c *Controller) {
	log.Info("TestHandler.ObjectDeleted")

    polDef := obj.(*api.PolicyDefinition).DeepCopy()
	// retrieve the match selector expression
	expression := RetrieveExpression(obj)
	c.logger.Infof("Selector expression: `%s`", expression)
	// nspaces stores the list of namespaces to define the scope of custom resource
	// in case no namespace has been specified, default namespace is used
	var nspaces = make([]string, len(polDef.Spec.Namespace)+1)
	if len(polDef.Spec.Namespace) != 0 {
		for i, val := range polDef.Spec.Namespace {
			c.logger.Infof("Namespace: %s", val)
			nspaces[i] = val
		}
	} else {
		c.logger.Infof("No namespace mentioned in configuration. Default namespace used")
		nspaces[0] = "default"
	}
	// loop over the specified namespaces
	for _, ns := range nspaces {

		if ns == "" {
			break
		}

		listOptions := metav1.ListOptions{LabelSelector: expression}

		// podClient := c.clientset.CoreV1().Pods(ns)
		deployClient := c.clientset.AppsV1().Deployments(ns)
		// retrieve deployments that match the expression
		dps, err := deployClient.List(context.TODO(), listOptions)
		if err != nil {
			c.logger.Errorf("Error in retrieving deployments: %v", err)
			return
		}

		// loop over the listed deployments
		for _, dp := range dps.Items {
			dpCopy := dp.DeepCopy()
			c.logger.Infof("Deployment Name: %s", dpCopy.Name)
			// metadata is deleted from the deployment
			// a patch is created to update the metadata
			oJson, err := json.Marshal(dpCopy)
			if err != nil {
				c.logger.Errorf("Error in creating json: %v", err)
				return
			}

			_, ok := dpCopy.Annotations[annotationKey]
			if ok {
				delete(dpCopy.Annotations, annotationKey)
			} else {
				c.logger.Infof("Annotation doesn't exist, not applying patch")
				return
			}
			// delete the added metadata
			_, ok = dpCopy.Annotations[injectedStatusKey]
			if ok {
				delete(dpCopy.Annotations, injectedStatusKey)
			}
			_, ok = dpCopy.Annotations["Action"]
			if ok {
				delete(dpCopy.Annotations, "Action")
			}
			_, ok = dpCopy.Annotations["LogFrequency"]
			if ok {
				delete(dpCopy.Annotations, "LogFrequency")
			}
			_, ok = dpCopy.Annotations["GroupLabel"]
			if ok {
				delete(dpCopy.Annotations, "GroupLabel")
			}

			mJson, err := json.Marshal(dpCopy)
			if err != nil {
				c.logger.Errorf("Error in creating json: %v", err)
				return
			}
			// create the patch
			patch, err := jsonpatch.CreatePatch(oJson, mJson)
			if err != nil {
				c.logger.Errorf("Error in creating patch: %v", err)
				return
			}

			pb, err := json.MarshalIndent(patch, "", " ")
			if err != nil {
				c.logger.Errorf("%v", err)
				return
			}

			// c.logger.Infof("Patch: %s", string(pb))
			// apply the patch
			final, err := c.clientset.AppsV1().Deployments(ns).Patch(context.TODO(), dpCopy.Name, types.JSONPatchType, pb, metav1.PatchOptions{})
			if err != nil {
				c.logger.Errorf("Error in applying patch: %v", err)
				return
			}

			_, err = json.MarshalIndent(final, "", " ")
			if err != nil {
				c.logger.Errorf("%v", err)
				return
			}

			c.logger.Infof("Successfully applied patch")
			// retrieve pods in the deployment
			// the pods in the deployment need to be restarted
			// for the sidecar to be deleted
			podSelector, err := metav1.LabelSelectorAsSelector(dpCopy.Spec.Selector)
			if err != nil {
				c.logger.Error(err)
			}

			listPodOptions := metav1.ListOptions{LabelSelector: podSelector.String()}
			podClient := c.clientset.CoreV1().Pods(ns)
			pods, err := podClient.List(context.TODO(), listPodOptions)
			if err != nil {
				c.logger.Errorf("Error in retrieving pods: %v", err)
			}

			for _, pod := range pods.Items {
				podCopy := pod.DeepCopy()
				// restart pod for deleting the sidecar
				if podCopy.ObjectMeta.Annotations[injectedStatusKey] == injectedStatusValue {
					c.logger.Infof("Sidecar is injected in pod, restarting: %s", podCopy.Name)

					err = podClient.Delete(context.TODO(), podCopy.Name, metav1.DeleteOptions{})
					if err != nil {
						c.logger.Errorf("Error in restarting pod %s: %v", podCopy.Name, err)
						return
					}
				}
			}
		}
	}
}

// handles customr receiver update
// when custom receiver is updated and it is processed in the queue
// ObjectAdded method is called to deal with it
func (t *TestHandler) ObjectUpdated(objOld, objNew interface{}, c *Controller) {
	log.Info("TestHandler.ObjectUpdated")
}
