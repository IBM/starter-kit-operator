package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// StarterKitSpec defines the desired state of StarterKit
// +k8s:openapi-gen=true
type StarterKitSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Options      StarterKitSpecOptions  `json:"options"`
	TemplateRepo StarterKitSpecTemplate `json:"templateRepo"`
}

type StarterKitSpecOptions struct {
	Port int32           `json:"port"`
	Env  []corev1.EnvVar `json:"env"`
}

type StarterKitSpecTemplate struct {
	TemplateOwner    string                   `json:"templateOwner"`
	TemplateRepoName string                   `json:"templateRepoName"`
	Owner            string                   `json:"owner"`
	Name             string                   `json:"name"`
	Description      string                   `json:"repoDescription"`
	SecretKeyRef     corev1.SecretKeySelector `json:"secretKeyRef"`
}

// StarterKitStatus defines the observed state of StarterKit
// +k8s:openapi-gen=true
type StarterKitStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	TargetRepo string `json:"targetRepo"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StarterKit is the Schema for the starterkits API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=starterkits,scope=Namespaced
type StarterKit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StarterKitSpec   `json:"spec,omitempty"`
	Status StarterKitStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StarterKitList contains a list of StarterKit
type StarterKitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StarterKit `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StarterKit{}, &StarterKitList{})
}
