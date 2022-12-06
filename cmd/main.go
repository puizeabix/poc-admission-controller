package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/puizeabix/poc-admission-controller/sidecar"
	log "github.com/sirupsen/logrus"

	admission "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecFactory  = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecFactory.UniversalDeserializer()
)

// add kind AdmissionReview in scheme
func init() {
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admission.AddToScheme(runtimeScheme)
	_ = v1.AddToScheme(runtimeScheme)
}

type admitv1Func func(admission.AdmissionReview) *admission.AdmissionResponse

type admitHandler struct {
	v1 admitv1Func
}

func AdmitHandler(f admitv1Func) admitHandler {
	return admitHandler{
		v1: f,
	}
}

// serve handles the http portion of a request prior to handing to an admit
// function
func serve(w http.ResponseWriter, r *http.Request, admit admitHandler) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		log.Error("contentType=%s, expect application/json", contentType)
		return
	}

	log.Info("handling request: %s", body)
	var responseObj runtime.Object
	if obj, gvk, err := deserializer.Decode(body, nil, nil); err != nil {
		msg := fmt.Sprintf("Request could not be decoded: %v", err)
		log.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return

	} else {
		requestedAdmissionReview, ok := obj.(*admission.AdmissionReview)
		if !ok {
			log.Error("Expected v1.AdmissionReview but got: %T", obj)
			return
		}
		responseAdmissionReview := &admission.AdmissionReview{}
		responseAdmissionReview.SetGroupVersionKind(*gvk)
		responseAdmissionReview.Response = admit.v1(*requestedAdmissionReview)
		responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
		responseObj = responseAdmissionReview

		log.WithFields(log.Fields{
			"patch": string(responseAdmissionReview.Response.Patch),
		}).Info("PatchOperation")

		rw, err := json.Marshal(responseAdmissionReview)

		if err != nil {
			log.Error(err.Error())
			return
		}

		log.WithFields(log.Fields{
			"patch": string(rw),
		}).Info("responseAdmissionReview")

	}
	log.Info("sending response: %v", responseObj)

	respBytes, err := json.Marshal(responseObj)
	if err != nil {
		log.Error("Error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(respBytes); err != nil {
		log.Error("Error: %v", err)
	}
}

func serveMutate(w http.ResponseWriter, r *http.Request) {
	serve(w, r, AdmitHandler(sidecar.Mutate))
}

func main() {
	var tlsKey, tlsCert string
	flag.StringVar(&tlsKey, "tlsKey", "/etc/certs/tls.key", "Path to the TLS key")
	flag.StringVar(&tlsCert, "tlsCert", "/etc/certs/tls.crt", "Path to the TLS certificate")
	flag.Parse()
	http.HandleFunc("/mutate", serveMutate)
	log.Info("Server started ...")
	if err := http.ListenAndServeTLS(":8443", tlsCert, tlsKey, nil); err != nil {
		log.Fatal("Webhook server existed: %v", err)
	}
}
