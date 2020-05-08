package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type State struct {
	metav1.TypeMeta `json:",inline"`

	BZStats *BZStats `json:"bzStats"`
}

type BZStats struct {
}
