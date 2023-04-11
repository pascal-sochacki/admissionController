package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/pascal-sochacki/admissionController/pkg/mutation"
	"github.com/pascal-sochacki/admissionController/pkg/validation"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/open-policy-agent/cert-controller/pkg/rotator"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	ownScheme = runtime.NewScheme()

	secretName      = "admission-controller"
	certDir         = flag.String("cert-dir", "./certs", "The directory where certs are stored, defaults to /certs")
	certServiceName = flag.String("cert-service-name", "admission-controller-webhook-service", "The service name used to generate the TLS cert's hostname. Defaults to admission-controller-webhook-service")
	port            = flag.Int("port", 443, "port for the server. defaulted to 443 if unspecified ")
	host            = flag.String("host", "", "the host address the webhook server listens on. defaults to all addresses.")
	VwhName         = flag.String("validating-webhook-configuration-name", "admission-controller-validating-webhook-configuration", "name of the ValidatingWebhookConfiguration")
	MwhName         = flag.String("mutation-webhook-configuration-name", "admission-controller-mutation-webhook-configuration", "name of the MutatingWebhookConfiguration")

	caName         = "admission-controller-ca"
	caOrganization = "admission-controller"
)

func main() {
	ns, found := os.LookupEnv("POD_NAMESPACE")
	if !found {
		log.Fatalln("Dont know my namespace, please set POD_NAMESPACE env")
	}

	err := scheme.AddToScheme(ownScheme)
	if err != nil {
		log.Fatalln(err.Error())
	}

	decoder, err := admission.NewDecoder(ownScheme)
	if err != nil {
		log.Fatalln(err.Error())
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		LeaderElection: false,
		Port:           *port,
		Scheme:         ownScheme,
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
	webhooks = append(webhooks, rotator.WebhookInfo{
		Name: *MwhName,
		Type: rotator.Mutating,
	})

	setupFinished := make(chan struct{})
	if err := rotator.AddRotator(mgr, &rotator.CertRotator{
		SecretKey: types.NamespacedName{
			Namespace: ns,
			Name:      secretName,
		},
		CertDir:        *certDir,
		CAName:         caName,
		Webhooks:       webhooks,
		CAOrganization: caOrganization,
		DNSName:        fmt.Sprintf("%s.%s.svc", *certServiceName, ns),
		IsReady:        setupFinished,
	}); err != nil {
		log.Fatalln(err, "unable to set up cert rotation")
	}
	mgr.GetWebhookServer().Register("/validate", &admission.Webhook{
		Handler: validation.Handler{
			Decoder: decoder,
		},
	})
	mgr.GetWebhookServer().Register("/mutate", &admission.Webhook{
		Handler: mutation.Handler{
			Decoder: decoder,
		},
	})

	err = mgr.Start(ctrl.SetupSignalHandler())
	if err != nil {
		log.Fatalln(err.Error())
	}

}
