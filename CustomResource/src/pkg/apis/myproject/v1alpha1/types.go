package v1alpha1
import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Receiver struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec ReceiverSpec `json:"spec"`
}

type ReceiverSpec struct {
	Namespace []string			`json:"Namespace"`
	LabelSelector PodSelector 	`json:"labelSelector"`
	GroupLabel []string			`json:"groupLabel"`
	// +optional
	LogFrequency int			`json:"logFrequency"`
	Action string				`json:"action"`
	// +optional
	Destination []Dest			`json:"destination"`
}

// destination of external server
type Dest struct {
	IPAddress string	`json:"ipaddress"`
	Port string			`json:"port"`
	Endpoint string		`json:"endpoint"`
}

// similar to official implementation
// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
type PodSelector struct {
	// +optional
	Expression []*ExpressionStruct	`json:"matchExpression,omitempty"`
	// +optional
	Labels map[string]string		`json:"matchLabels,omitempty"`
}

type ExpressionStruct struct {
	Key string			`json:"key"`
	Operator string		`json:"operator"`
	Values []string		`json:"values"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ReceiverList struct {
	metav1.TypeMeta 	`json:",inline"`
	metav1.ListMeta 	`json:"metadata"`
	Items []Receiver	`json:"items"`
}
