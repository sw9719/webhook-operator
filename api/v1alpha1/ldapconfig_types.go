/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type tlscfg struct {
	// name of the secret containing the server key to use
	// // +kubebuilder:validation:Optional
	KeyCert string `json:"keycert"`
	// name of the configmap containing ca bundle to use
	// +kubebuilder:validation:Optional
	Ca string `json:"ca"`
	// require secure binds or not
	// +kubebuilder:default:=false
	RequireSSlBinds bool `json:"requiresslbinds"`
}

type storage struct {
	// Class defines the type of kubernetes storage to use. This option defines the logic of how operator parses the storage options
	Class    string `json:"class"`
	HostPath string `json:"hostpath"`
}

type Cfg struct {
	// basedn specified the DN of the root entry
	// +kubebuilder:validation:Required
	Basedn string `json:"basedn"`
	// backend name to use when creating backend
	// +kubebuilder:validation:Required
	Backendname string `json:"backendname"`
	// Password to use for cn=Directory Manager
	// +kubebuilder:validation:Required
	Password string `json:"password"`
	// LDAP configuration related to tls
	// +kubebuilder:validation:Optional
	Tls tlscfg `json:"tls"`
	// Option to perfom re-indexing during startup
	// +kubebuilder:validation:Optional
	Reindex bool `json:"reindex"`
	// loglevel for the container logs
	// +kubebuilder:validation:Optional
	Loglevel string `json:"loglevel"`
	// Storage options for ldap for data persistence
	// +kubebuilder:validation:Optional
	Stg storage `json:"storage"`
	// allow anonymous binds
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=true
	Allowanonymous bool `json:"allowAnonymous"`
	// extended config allows allows adding custom schema. This is usefull for adding
	// custom objectclasses and attributes. It follows the same format as 99user.ldif file
	// Set the value of this parameter to configmap containing the file contents in the
	// same namespace. The key should be '99user.ldif'
	// +kubebuilder:validation:Optional
	ExtendedConfig string `json:"extendedConfig"`
}

// LdapconfigSpec defines the desired state of Ldapconfig
type LdapconfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// config defines the configuration of ldap service
	Config Cfg `json:"config"`
}

// LdapconfigStatus defines the observed state of Ldapconfig
type LdapconfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Condition string `json:"condition"`
	Ready     bool   `json:"ready"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Ldapconfig is the Schema for the ldapconfigs API
type Ldapconfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LdapconfigSpec   `json:"spec,omitempty"`
	Status LdapconfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LdapconfigList contains a list of Ldapconfig
type LdapconfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Ldapconfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Ldapconfig{}, &LdapconfigList{})
}
