package types

import (
	"github.com/intel/multus-cni/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NetworkAttachmentDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is the list of Namespace objects in the list.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/
	Items []types.NetworkAttachmentDefinition `json:"items" protobuf:"bytes,2,rep,name=items"`
}
