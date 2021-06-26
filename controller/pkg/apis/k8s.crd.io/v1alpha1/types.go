package v1alpha1

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PolicyDefinition struct {
	metav1.TypeMeta             `json:",inline"`
	metav1.ObjectMeta           `json:"metadata"`
	Spec Spec                   `json:"spec"`
}

type Spec struct {
	Namespace []string			`json:"Namespace"`
	Action string				`json:"action"`
	LabelSelector LabelSelector `json:"labelSelector"`
	GroupLabel []string			`json:"groupLabel"`
	// +optional
	LogSpec LogSpec             `json:"logSpec"`
}

type LogSpec struct {
    LogFrequency int            `json:"logFrequency"`
    Destination []Destination   `json:"destination"`
}

// destination of external server
type Destination struct {
	Ipaddress string	`json:"ipaddress"`
	Port string			`json:"port"`
	Endpoint string		`json:"endpoint"`
}

// similar to official implementation
// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
type LabelSelector struct {
	// +optional
	Expression []*Expression    `json:"matchExpression,omitempty"`
    // +optional
    Labels map[string]string    `json:"matchLabels,omitempty"`
}

type Expression struct {
	Key string			`json:"key"`
	Operator string		`json:"operator"`
	Values []string		`json:"values"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PolicyDefinitionList struct {
	metav1.TypeMeta	            `json:",inline"`
	metav1.ListMeta	            `json:"metadata"`
	Items []PolicyDefinition    `json:"items"`
}
