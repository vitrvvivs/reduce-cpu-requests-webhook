package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/wI2L/jsondiff"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	http.HandleFunc("/webhook", Mutate)
	http.HandleFunc("/health", Health)

	certFile := os.Getenv("TLS_CERT_FILE") 
	if certFile == "" {
		certFile = "/etc/webhook/certs/tls.crt"
	}
	keyFile := os.Getenv("TLS_KEY_FILE")
	if keyFile == "" {
		keyFile = "/etc/webhook/certs/tls.key"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "443"
	}
	port = ":" + port
	
	fmt.Println("Starting webhook server on port", port)
	log.Fatal(http.ListenAndServeTLS(port, certFile, keyFile, nil))
}

func Mutate(w http.ResponseWriter, r *http.Request) {
	var admission admissionv1.AdmissionReview
	fmt.Println("Received request for webhook" )

	// Parse Request
	bodybuf := new(bytes.Buffer)
	bodybuf.ReadFrom(r.Body)
	body := bodybuf.Bytes()
	if len(body) == 0 {
		return
	}
	err := json.Unmarshal(body, &admission)
	if err != nil {
		fmt.Printf("could not unmarshal request: %v\n", err)
		http.Error(w, fmt.Sprintf("could not unmarshal request: %v", err), http.StatusBadRequest)
		return
	}

	// Get Pod
	pod := corev1.Pod{}
	if admission.Request == nil || admission.Request.Kind.Kind != "Pod" {
		http.Error(w, "only 'Pod's are supported", http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(admission.Request.Object.Raw, &pod)
	if err != nil {
		fmt.Printf("could not unmarshal pod: %v\n", err)
		http.Error(w, fmt.Sprintf("could not unmarshal pod: %v", err), http.StatusBadRequest)
		return
	}

	// Prepare Patch
	var patch jsondiff.Patch
	patchtype := admissionv1.PatchTypeJSONPatch
	review := &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: &admissionv1.AdmissionResponse{
			UID:     admission.Request.UID,
			Allowed: true,
		},
	}

	// Make changes to Pod and generate diff
	fmt.Println(pod.Name, "in namespace", pod.Namespace)
	newPod := pod.DeepCopy()
	for _, container := range pod.Spec.Containers {
		if container.Resources.Requests == nil {
			container.Resources.Requests = corev1.ResourceList{}
		}
		container.Resources.Requests[corev1.ResourceCPU] = resource.MustParse("10m")
	}
	for _, container := range pod.Spec.InitContainers {
		if container.Resources.Requests == nil {
			container.Resources.Requests = corev1.ResourceList{}
		}
		container.Resources.Requests[corev1.ResourceCPU] = resource.MustParse("10m")
	}
	patch, err = jsondiff.Compare(newPod, pod)
	if err != nil {
		fmt.Printf("could not create JSONPatch: %v\n", err)
		http.Error(w, fmt.Sprintf("could create JSONPatch: %v", err), http.StatusInternalServerError)
		return
	}
	if patch != nil {

		//s, _ := json.MarshalIndent(pod.Spec.Containers, "", "  ")
		//fmt.Println(string(s))
		//s, _ = json.MarshalIndent(patch, "", "  ")
		//fmt.Println("Patch to apply:")
		//fmt.Println(string(s))

		review.Response.PatchType = &patchtype
		review.Response.Patch, err = json.Marshal(patch)
		if err != nil {
			fmt.Printf("could not marshal patch: %v\n", err)
			http.Error(w, fmt.Sprintf("could not marshal patch: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		review.Response.Allowed = true
		review.Response.Result = &metav1.Status{
			Code:    http.StatusOK,
			Message: "No changes made to Pod",
		}
	}

	response, err := json.Marshal(review)
	if err != nil {
		fmt.Printf("could not marshal response: %v\n", err)
		http.Error(w, fmt.Sprintf("could not marshal response: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", response)
}

func Health(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}
