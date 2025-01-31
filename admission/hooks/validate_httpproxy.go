package hooks

import (
	"context"
	"fmt"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/validate-projectcontour-io-httpproxy,mutating=false,failurePolicy=fail,sideEffects=None,groups=projectcontour.io,resources=httpproxies,verbs=create;update,versions=v1,name=vhttpproxy.kb.io,admissionReviewVersions={v1,v1beta1}

type contourHTTPProxyValidator struct {
	client  client.Client
	decoder *admission.Decoder
}

// NewContourHTTPProxyValidator creates a webhook handler for Contour HTTPProxy.
func NewContourHTTPProxyValidator(c client.Client, dec *admission.Decoder) http.Handler {
	return &webhook.Admission{Handler: &contourHTTPProxyValidator{c, dec}}
}

func (v *contourHTTPProxyValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	hp := &unstructured.Unstructured{}
	if err := v.decoder.Decode(req, hp); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	newAnn := hp.GetAnnotations()
	newIngressClassNameField, ok, err := getHTTPProxyIngressClassNameField(hp)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	if !ok {
		newIngressClassNameField = ""
	}

	switch req.Operation {
	case admissionv1.Create:
		if newAnn[annotationKubernetesIngressClass] == "" && newAnn[annotationContourIngressClass] == "" && newIngressClassNameField == "" {
			return admission.Denied(fmt.Sprintf("either %s or %s annotation or .spec.ingressClassName field should be set", annotationKubernetesIngressClass, annotationContourIngressClass))
		}

	case admissionv1.Update:
		old := &unstructured.Unstructured{}
		if err := v.decoder.DecodeRaw(req.OldObject, old); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		oldAnn := old.GetAnnotations()
		oldIngressClassNameField, ok, err := getHTTPProxyIngressClassNameField(old)
		if err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		if !ok {
			oldIngressClassNameField = ""
		}

		if newAnn[annotationKubernetesIngressClass] != oldAnn[annotationKubernetesIngressClass] {
			return admission.Denied("changing annotation " + annotationKubernetesIngressClass + " is not allowed")
		}
		if newAnn[annotationContourIngressClass] != oldAnn[annotationContourIngressClass] {
			return admission.Denied("changing annotation " + annotationContourIngressClass + " is not allowed")
		}
		if newIngressClassNameField != oldIngressClassNameField {
			return admission.Denied("changing field .spec.ingressClassName is not allowed")
		}
	}

	return admission.Allowed("ok")
}
