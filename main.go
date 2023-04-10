package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	v1 "k8s.io/api/admission/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/open-policy-agent/cert-controller/pkg/rotator"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type Handle struct {
	decoder *admission.Decoder
}

func (h Handle) Handle(ctx context.Context, request admission.Request) admission.Response {
	configMap := &core.ConfigMap{}

	err := h.decoder.Decode(request, configMap)
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

func GetNamespace() string {
	ns, found := os.LookupEnv("POD_NAMESPACE")
	if !found {
		return "default"
	}
	return ns
}

var (
	scheme = runtime.NewScheme()

	secretName      = "admission-controller"
	certDir         = flag.String("cert-dir", "./certs", "The directory where certs are stored, defaults to /certs")
	certServiceName = flag.String("cert-service-name", "admission-controller-webhook-service", "The service name used to generate the TLS cert's hostname. Defaults to admission-controller-webhook-service")
	port            = flag.Int("port", 443, "port for the server. defaulted to 443 if unspecified ")
	host            = flag.String("host", "", "the host address the webhook server listens on. defaults to all addresses.")
	VwhName         = flag.String("validating-webhook-configuration-name", "admission-controller-validating-webhook-configuration", "name of the ValidatingWebhookConfiguration")

	caName         = "admission-controller-ca"
	caOrganization = "admission-controller"
)

func main() {

	clientgoscheme.AddToScheme(scheme)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		LeaderElection: false,
		Port:           *port,
		Scheme:         scheme,
		Host:           *host,
		CertDir:        *certDir,
		MapperProvider: func(c *rest.Config) (meta.RESTMapper, error) {
			return apiutil.NewDynamicRESTMapper(c)
		},
	})
	if err != nil {
		log.Fatalln(err.Error())
	}

	var webhooks []rotator.WebhookInfo
	webhooks = append(webhooks, rotator.WebhookInfo{
		Name: *VwhName,
		Type: rotator.Validating,
	})

	setupFinished := make(chan struct{})
	if err := rotator.AddRotator(mgr, &rotator.CertRotator{
		SecretKey: types.NamespacedName{
			Namespace: GetNamespace(),
			Name:      secretName,
		},
		CertDir:        *certDir,
		CAName:         caName,
		Webhooks:       webhooks,
		CAOrganization: caOrganization,
		DNSName:        fmt.Sprintf("%s.%s.svc", *certServiceName, GetNamespace()),
		IsReady:        setupFinished,
	}); err != nil {
		log.Fatalln(err, "unable to set up cert rotation")
	}

	decoder, err := admission.NewDecoder(scheme)
	if err != nil {
		log.Fatalln(err.Error())
	}

	wh := &admission.Webhook{
		Handler: Handle{
			decoder: decoder,
		},
	}

	mgr.GetWebhookServer().Register("/validate", wh)

	err = mgr.Start(ctrl.SetupSignalHandler())
	if err != nil {
		log.Fatalln(err.Error())
	}

}
