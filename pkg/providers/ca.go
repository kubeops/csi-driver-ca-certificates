/*
Copyright AppsCode Inc. and Contributors

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

package providers

import (
	"context"
	"crypto/x509"
	"fmt"
	"time"

	"kubeops.dev/csi-driver-cacerts/pkg/providers/lib"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"gomodules.xyz/cert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"kmodules.xyz/client-go/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type IssuerProvider struct {
	Reader client.Reader
}

var _ lib.CAProvider = &IssuerProvider{}

func (c *IssuerProvider) GetCAs(obj client.Object, _ string) ([]*x509.Certificate, error) {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	issuerKey, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return nil, err
	}

	issuer, ok := obj.(cmapi.GenericIssuer)
	if !ok {
		return nil, fmt.Errorf("%s %s is not a GenericIssuer", kind, issuerKey)
	}

	if issuer.GetSpec().CA == nil {
		return nil, fmt.Errorf("%s %s does not have a CA", kind, issuerKey)
	}

	var secret corev1.Secret
	secretRef := client.ObjectKey{
		Namespace: func() string {
			if kind == "ClusterIssuer" {
				// cert-manager requires the ClusterIssuer ca secret to be in the same namespace where it is deployed.
				// So, csi-driver must be in the same namespace where cert-manager is installed.
				// ns will be defaulted to cert-manager namespace in standard deployments.
				return meta.PodNamespace()
			}
			return issuer.GetNamespace()
		}(),
		Name: issuer.GetSpec().CA.SecretName,
	}
	err = c.Reader.Get(context.TODO(), secretRef, &secret)
	if err != nil {
		return nil, err
	}
	key := corev1.TLSCertKey
	data, ok := secret.Data[key]
	if !ok {
		return nil, fmt.Errorf("missing key %s in secret %s", key, secretRef)
	}
	caCerts, _, err := cert.ParseRootCAs(data)
	if err != nil {
		return nil, err
	}
	if len(caCerts) == 0 {
		return nil, fmt.Errorf("%s %s signing certificate is not a CA", kind, issuerKey)
	}

	now := time.Now()
	for _, caCert := range caCerts {
		if now.Before(caCert.NotBefore) {
			return nil, fmt.Errorf("%s %s points a CA cert not valid before %v, now: %s", kind, issuerKey, caCert.NotBefore, now)
		}
		if now.After(caCert.NotAfter) {
			return nil, fmt.Errorf("%s %s points a CA cert expired at %v, now: %s", kind, issuerKey, caCert.NotAfter, now)
		}
	}

	return caCerts, err
}
