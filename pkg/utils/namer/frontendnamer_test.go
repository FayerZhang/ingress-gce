/*
Copyright 2019 The Kubernetes Authors.

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

package namer

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const clusterUID = "uid1"

func newIngress(namespace, name string) *v1beta1.Ingress {
	return &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

// TestV1IngressFrontendNamer tests that v1 frontend namer created from load balancer,
// 1. returns expected values.
// 2. returns same values as old namer.
// 3. returns same values as v1 frontend namer created from ingress.
func TestV1IngressFrontendNamer(t *testing.T) {
	longString := "01234567890123456789012345678901234567890123456789"
	testCases := []struct {
		desc      string
		namespace string
		name      string
		// Expected values.
		lbName              string
		targetHTTPProxy     string
		targetHTTPSProxy    string
		sslCert             string
		forwardingRuleHTTP  string
		forwardingRuleHTTPS string
		urlMap              string
	}{
		{
			"simple case",
			"namespace",
			"name",
			"namespace-name--uid1",
			"%s-tp-namespace-name--uid1",
			"%s-tps-namespace-name--uid1",
			"%s-ssl-9a60a5272f6eee97-%s--uid1",
			"%s-fw-namespace-name--uid1",
			"%s-fws-namespace-name--uid1",
			"%s-um-namespace-name--uid1",
		},
		{
			"62 characters",
			// Total combined length of namespace and name is 47.
			longString[:23],
			longString[:24],
			"01234567890123456789012-012345678901234567890123--uid1",
			"%s-tp-01234567890123456789012-012345678901234567890123--uid1",
			"%s-tps-01234567890123456789012-012345678901234567890123--uid1",
			"%s-ssl-4169c63684f5e4cd-%s--uid1",
			"%s-fw-01234567890123456789012-012345678901234567890123--uid1",
			"%s-fws-01234567890123456789012-012345678901234567890123--uid1",
			"%s-um-01234567890123456789012-012345678901234567890123--uid1",
		},
		{
			"63 characters",
			// Total combined length of namespace and name is 48.
			longString[:24],
			longString[:24],
			"012345678901234567890123-012345678901234567890123--uid1",
			"%s-tp-012345678901234567890123-012345678901234567890123--uid1",
			"%s-tps-012345678901234567890123-012345678901234567890123--uid0",
			"%s-ssl-c7616cb0f76c2df2-%s--uid1",
			"%s-fw-012345678901234567890123-012345678901234567890123--uid1",
			"%s-fws-012345678901234567890123-012345678901234567890123--uid0",
			"%s-um-012345678901234567890123-012345678901234567890123--uid1",
		},
		{
			"64 characters",
			// Total combined length of namespace and name is 49.
			longString[:24],
			longString[:25],
			"012345678901234567890123-0123456789012345678901234--uid1",
			"%s-tp-012345678901234567890123-0123456789012345678901234--uid0",
			"%s-tps-012345678901234567890123-0123456789012345678901234--ui0",
			"%s-ssl-537beba3a874a029-%s--uid1",
			"%s-fw-012345678901234567890123-0123456789012345678901234--uid0",
			"%s-fws-012345678901234567890123-0123456789012345678901234--ui0",
			"%s-um-012345678901234567890123-0123456789012345678901234--uid0",
		},
		{
			"long namespace",
			longString,
			"0",
			"01234567890123456789012345678901234567890123456789-0--uid1",
			"%s-tp-01234567890123456789012345678901234567890123456789-0--u0",
			"%s-tps-01234567890123456789012345678901234567890123456789-0--0",
			"%s-ssl-92bdb5e4d378b3ce-%s--uid1",
			"%s-fw-01234567890123456789012345678901234567890123456789-0--u0",
			"%s-fws-01234567890123456789012345678901234567890123456789-0--0",
			"%s-um-01234567890123456789012345678901234567890123456789-0--u0",
		},
		{
			"long name",
			"0",
			longString,
			"0-01234567890123456789012345678901234567890123456789--uid1",
			"%s-tp-0-01234567890123456789012345678901234567890123456789--u0",
			"%s-tps-0-01234567890123456789012345678901234567890123456789--0",
			"%s-ssl-8f3d42933afb5d1c-%s--uid1",
			"%s-fw-0-01234567890123456789012345678901234567890123456789--u0",
			"%s-fws-0-01234567890123456789012345678901234567890123456789--0",
			"%s-um-0-01234567890123456789012345678901234567890123456789--u0",
		},
		{
			"long name and namespace",
			longString,
			longString,
			"01234567890123456789012345678901234567890123456789-012345678900",
			"%s-tp-01234567890123456789012345678901234567890123456789-01230",
			"%s-tps-01234567890123456789012345678901234567890123456789-0120",
			"%s-ssl-a04f7492b36aeb20-%s--uid1",
			"%s-fw-01234567890123456789012345678901234567890123456789-01230",
			"%s-fws-01234567890123456789012345678901234567890123456789-0120",
			"%s-um-01234567890123456789012345678901234567890123456789-01230",
		},
	}
	for _, prefix := range []string{"k8s", "mci"} {
		oldNamer := NewNamerWithPrefix(prefix, clusterUID, "")
		secretHash := fmt.Sprintf("%x", sha256.Sum256([]byte("test123")))[:16]
		for _, tc := range testCases {
			tc.desc = fmt.Sprintf("%s prefix %s", tc.desc, prefix)
			t.Run(tc.desc, func(t *testing.T) {
				key := fmt.Sprintf("%s/%s", tc.namespace, tc.name)
				t.Logf("Ingress key %s", key)
				namer := newV1IngressFrontendNamerFromLBName(oldNamer.LoadBalancer(key), oldNamer)
				tc.targetHTTPProxy = fmt.Sprintf(tc.targetHTTPProxy, prefix)
				tc.targetHTTPSProxy = fmt.Sprintf(tc.targetHTTPSProxy, prefix)
				tc.sslCert = fmt.Sprintf(tc.sslCert, prefix, secretHash)
				tc.forwardingRuleHTTP = fmt.Sprintf(tc.forwardingRuleHTTP, prefix)
				tc.forwardingRuleHTTPS = fmt.Sprintf(tc.forwardingRuleHTTPS, prefix)
				tc.urlMap = fmt.Sprintf(tc.urlMap, prefix)

				// Test behavior of V1 Namer created using load-balancer name.
				if diff := cmp.Diff(tc.lbName, namer.LbName()); diff != "" {
					t.Errorf("namer.LbName() mismatch (-want +got):\n%s", diff)
				}
				targetHTTPProxyName := namer.TargetProxy(HTTPProtocol)
				if diff := cmp.Diff(tc.targetHTTPProxy, targetHTTPProxyName); diff != "" {
					t.Errorf("namer.TargetProxy(HTTPProtocol) mismatch (-want +got):\n%s", diff)
				}
				targetHTTPSProxyName := namer.TargetProxy(HTTPSProtocol)
				if diff := cmp.Diff(tc.targetHTTPSProxy, targetHTTPSProxyName); diff != "" {
					t.Errorf("namer.TargetProxy(HTTPSProtocol) mismatch (-want +got):\n%s", diff)
				}
				sslCertName := namer.SSLCertName(secretHash)
				if diff := cmp.Diff(tc.sslCert, sslCertName); diff != "" {
					t.Errorf("namer.SSLCertName(%q) mismatch (-want +got):\n%s", secretHash, diff)
				}
				httpForwardingRuleName := namer.ForwardingRule(HTTPProtocol)
				if diff := cmp.Diff(tc.forwardingRuleHTTP, httpForwardingRuleName); diff != "" {
					t.Errorf("namer.ForwardingRule(HTTPProtocol) mismatch (-want +got):\n%s", diff)
				}
				httpsForwardingRuleName := namer.ForwardingRule(HTTPSProtocol)
				if diff := cmp.Diff(tc.forwardingRuleHTTPS, httpsForwardingRuleName); diff != "" {
					t.Errorf("namer.ForwardingRule(HTTPSProtocol) mismatch (-want +got):\n%s", diff)
				}
				urlMapName := namer.UrlMap()
				if diff := cmp.Diff(tc.urlMap, urlMapName); diff != "" {
					t.Errorf("namer.UrlMap() mismatch (-want +got):\n%s", diff)
				}

				// Ensure that V1 Namer returns same values as old namer.
				lbName := oldNamer.LoadBalancer(key)
				if diff := cmp.Diff(lbName, namer.LbName()); diff != "" {
					t.Errorf("Got diff between old and V1 namers, lbName mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(oldNamer.TargetProxy(lbName, HTTPProtocol), targetHTTPProxyName); diff != "" {
					t.Errorf("Got diff between old and V1 namers, target http proxy mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(oldNamer.TargetProxy(lbName, HTTPSProtocol), targetHTTPSProxyName); diff != "" {
					t.Errorf("Got diff between old and V1 namers, target https proxy mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(oldNamer.SSLCertName(lbName, secretHash), sslCertName); diff != "" {
					t.Errorf("Got diff between old and V1 namers, SSL cert mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(oldNamer.ForwardingRule(lbName, HTTPProtocol), httpForwardingRuleName); diff != "" {
					t.Errorf("Got diff between old and V1 namers, http forwarding rule mismatch(-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(oldNamer.ForwardingRule(lbName, HTTPSProtocol), httpsForwardingRuleName); diff != "" {
					t.Errorf("Got diff between old and V1 namers, https forwarding rule mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(oldNamer.UrlMap(lbName), urlMapName); diff != "" {
					t.Errorf("Got diff between old and V1 namers, url map mismatch(-want +got):\n%s", diff)
				}

				// Ensure that V1 namer created using ingress returns same values as V1 namer created using lb name.
				namerFromIngress := newV1IngressFrontendNamer(newIngress(tc.namespace, tc.name), oldNamer)
				if diff := cmp.Diff(targetHTTPProxyName, namerFromIngress.TargetProxy(HTTPProtocol)); diff != "" {
					t.Errorf("Got diff for target http proxy (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(targetHTTPSProxyName, namerFromIngress.TargetProxy(HTTPSProtocol)); diff != "" {
					t.Errorf("Got diff for target https proxy (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(sslCertName, namerFromIngress.SSLCertName(secretHash)); diff != "" {
					t.Errorf("Got diff for SSL cert (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(httpForwardingRuleName, namerFromIngress.ForwardingRule(HTTPProtocol)); diff != "" {
					t.Errorf("Got diff for http forwarding rule (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(httpsForwardingRuleName, namerFromIngress.ForwardingRule(HTTPSProtocol)); diff != "" {
					t.Errorf("Got diff for https forwarding rule (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(urlMapName, namerFromIngress.UrlMap()); diff != "" {
					t.Errorf("Got diff url map (-want +got):\n%s", diff)
				}
			})
		}
	}
}