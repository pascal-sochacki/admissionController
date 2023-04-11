package mutation

import (
	"context"

	"gomodules.xyz/jsonpatch/v2"
	v1 "k8s.io/api/admission/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type Handler struct {
	Decoder *admission.Decoder
}

func (m Handler) Handle(ctx context.Context, request admission.Request) admission.Response {
	return admission.Response{
		AdmissionResponse: v1.AdmissionResponse{
			Allowed: true,
		},
		Patches: []jsonpatch.JsonPatchOperation{
			{
				Operation: "add",
				Path:      "/metadata/annotations",
				Value:     map[string]string{"bob": "5"},
			},
		},
	}
}
