package integrationtests

// This file includes utilities to start an environment with API Server, and a configured reconciler.

import (
	"context"
	"fmt"
	"os"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	DefaultTestAPIURL    = "https://ENVIRONMENTID.live.dynatrace.com/api"
	DefaultTestNamespace = "dynatrace"
)

var schema = apiextensionsv1.JSONSchemaProps{
	Description: "DynaKube is the Schema for the DynaKube API",
	Type:        "object",
	Properties: map[string]apiextensionsv1.JSONSchemaProps{
		"apiUrl": {
			Description: "Location of the Dynatrace API to connect to, including your specific environment ID",
			Type:        "string",
		},
		"": {},

		//APIURL:      DefaultTestAPIURL,
		//Tokens:      "token-test",
		//EnableIstio: true,
		//ClassicFullStack: dynatracev1alpha1.FullStackSpec{
		//	Enabled: true,
		//},
	},
}
var validation = apiextensionsv1.CustomResourceValidation{OpenAPIV3Schema: &schema}
var testEnvironmentCRDs = []client.Object{
	&apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dynakubes.dynatrace.com",
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "dynatrace.com",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Kind:     "DynaKube",
				ListKind: "DynaKubeList",
				Plural:   "dynakubes",
				Singular: "dynakube",
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1alpha1",
					Storage: true,
					Served:  true,
					Subresources: &apiextensionsv1.CustomResourceSubresources{
						Status: &apiextensionsv1.CustomResourceSubresourceStatus{},
					},
					Schema: &validation,
				},
			},
		},
	},
	&apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "virtualservices.networking.istio.io",
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "networking.istio.io",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Kind:     "VirtualService",
				ListKind: "VirtualServiceList",
				Plural:   "virtualservices",
				Singular: "virtualservice",
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1alpha3",
					Storage: true,
					Subresources: &apiextensionsv1.CustomResourceSubresources{
						Status: &apiextensionsv1.CustomResourceSubresourceStatus{},
					},
					Schema: &validation,
				},
			},
		},
	},
	&apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "serviceentries.networking.istio.io",
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "networking.istio.io",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Kind:     "ServiceEntry",
				ListKind: "ServiceEntryList",
				Plural:   "serviceentries",
				Singular: "serviceentry",
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1alpha3",
					Storage: true,
					Subresources: &apiextensionsv1.CustomResourceSubresources{
						Status: &apiextensionsv1.CustomResourceSubresourceStatus{},
					},
					Schema: &validation,
				},
			},
		},
	},
}

func init() {
	os.Setenv("POD_NAMESPACE", DefaultTestNamespace)
}

type ControllerTestEnvironment struct {
	CommunicationHosts []string
	Client             client.Client
	Reconciler         *dynakube.ReconcileDynaKube

	server *envtest.Environment
}

func newTestEnvironment() (*ControllerTestEnvironment, error) {
	kubernetesAPIServer := &envtest.Environment{
		KubeAPIServerFlags: append(envtest.DefaultKubeAPIServerFlags, "--allow-privileged"),
		CRDs:               testEnvironmentCRDs,
	}

	cfg, err := kubernetesAPIServer.Start()
	if err != nil {
		return nil, err
	}

	kubernetesClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		errStop := kubernetesAPIServer.Stop()
		if errStop != nil {
			return nil, fmt.Errorf("%s\n%s", err.Error(), errStop.Error())
		}
		return nil, err
	}

	if err = kubernetesClient.Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: DefaultTestNamespace,
		}}); err != nil {
		errStop := kubernetesAPIServer.Stop()
		if errStop != nil {
			return nil, fmt.Errorf("%s\n%s", err.Error(), errStop.Error())
		}
		return nil, err
	}

	if err = kubernetesClient.Create(context.TODO(), buildDynatraceClientSecret()); err != nil {
		errStop := kubernetesAPIServer.Stop()
		if errStop != nil {
			return nil, fmt.Errorf("%s\n%s", err.Error(), errStop.Error())
		}
		return nil, err
	}

	communicationHosts := []string{
		"https://endpoint1.test.com/communication",
		"https://endpoint2.test.com/communication",
	}
	environment := &ControllerTestEnvironment{
		server:             kubernetesAPIServer,
		Client:             kubernetesClient,
		CommunicationHosts: communicationHosts,
	}
	environment.Reconciler = dynakube.NewDynaKubeReconciler(kubernetesClient, kubernetesClient, scheme.Scheme, mockDynatraceClientFunc(&environment.CommunicationHosts), zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout)), cfg)

	return environment, nil
}

func (e *ControllerTestEnvironment) Stop() error {
	return e.server.Stop()
}

func (e *ControllerTestEnvironment) AddOneAgent(n string, s *dynatracev1alpha1.DynaKubeSpec) error {
	return e.Client.Create(context.TODO(), &dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      n,
			Namespace: DefaultTestNamespace,
		},
		Spec: *s,
	})
}

func newReconciliationRequest(oaName string) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      oaName,
			Namespace: DefaultTestNamespace,
		},
	}
}

func mockDynatraceClientFunc(communicationHosts *[]string) dynakube.DynatraceClientFunc {
	return func(client client.Client, oa *dynatracev1alpha1.DynaKube, _ *corev1.Secret) (dtclient.Client, error) {
		commHosts := make([]dtclient.CommunicationHost, len(*communicationHosts))
		for i, c := range *communicationHosts {
			commHosts[i] = dtclient.CommunicationHost{Protocol: "https", Host: c, Port: 443}
		}

		connInfo := dtclient.ConnectionInfo{
			TenantUUID:         "asdf",
			CommunicationHosts: commHosts,
		}

		dtc := new(dtclient.MockDynatraceClient)
		dtc.On("GetLatestAgentVersion", "unix", "default").Return("17", nil)
		dtc.On("GetConnectionInfo").Return(connInfo, nil)
		dtc.On("GetCommunicationHostForClient").Return(dtclient.CommunicationHost{
			Protocol: "https",
			Host:     DefaultTestAPIURL,
			Port:     443,
		}, nil)
		dtc.On("GetTokenScopes", "42").Return(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, nil)
		dtc.On("GetTokenScopes", "43").Return(dtclient.TokenScopes{dtclient.TokenScopeDataExport}, nil)

		return dtc, nil
	}
}

func buildDynatraceClientSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "token-test",
			Namespace: DefaultTestNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"paasToken": []byte("42"),
			"apiToken":  []byte("43"),
		},
	}
}
