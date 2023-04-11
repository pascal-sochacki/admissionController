package validation

import (
	"context"
	"log"

	v1 "k8s.io/api/admission/v1"
	core "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type Handler struct {
	Decoder *admission.Decoder
}

func (h Handler) Handle(ctx context.Context, request admission.Request) admission.Response {
	configMap := &core.ConfigMap{}

	err := h.Decoder.Decode(request, configMap)
	if err != nil {
		return admission.Response{}
	}

	log.Print(configMap)
	return admission.Response{
		AdmissionResponse: v1.AdmissionResponse{
			Allowed: true,
		},
	}
}
