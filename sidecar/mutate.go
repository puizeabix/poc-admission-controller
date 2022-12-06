package sidecar

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	admission "k8s.io/api/admission/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecFactory  = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecFactory.UniversalDeserializer()
)

func Mutate(ar admission.AdmissionReview) *admission.AdmissionResponse {
	log.Info("Start mutate deployment")

	raw := ar.Request.Object.Raw
	deployment := v1.Deployment{}

	if _, _, err := deserializer.Decode(raw, nil, &deployment); err != nil {

		log.Error(err)
		return &admission.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	containers := deployment.Spec.Template.Spec.Containers

	containers = append(containers, corev1.Container{
		Name:            "sidecar",
		Image:           "redis",
		ImagePullPolicy: corev1.PullIfNotPresent,
	})

	cStr, err := json.Marshal(containers)
	if err != nil {
		log.Error(err.Error())
	}

	log.WithFields(log.Fields{
		"patch": string(cStr),
	}).Info("New Containers spec")

	pt := admission.PatchTypeJSONPatch
	deploymentPatch := fmt.Sprintf(`[{"op": "replace", "path": "/spec/template/spec/containers", "value": %s }]`, string(cStr))

	log.Info("Patch deployment with additional sidecar")
	//return &admission.AdmissionResponse{Allowed: true, PatchType: &pt, Patch: []byte(deploymentPatch)}
	return &admission.AdmissionResponse{Allowed: true, PatchType: &pt, Patch: []byte(deploymentPatch)}
}
