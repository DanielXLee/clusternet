/*
Copyright 2021 The Clusternet Authors.

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

package utils

import (
	"context"
	"testing"

	"github.com/clusternet/clusternet/pkg/apis/apps/v1alpha1"
	"github.com/clusternet/clusternet/pkg/generated/clientset/versioned/fake"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
)

func TestResourceNeedResync(t *testing.T) {
	barX := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"externalTrafficPolicy": "Cluster",
				"ipFamilies": []string{
					"IPv4",
				},
				"ipFamilyPolicy": "SingleStack",
				"ports": []interface{}{
					map[string]interface{}{
						"name":       "tcp-80-80",
						"port":       int64(80),
						"protocol":   "TCP",
						"targetPort": int64(80),
					},
					map[string]interface{}{
						"name":       "tcp-443-443",
						"port":       int64(443),
						"protocol":   "TCP",
						"targetPort": int64(443),
					},
				},
			},
		},
	}
	barY := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"clusterIP": "10.98.177.115", // add this field
				"clusterIPs": []string{ // add this field
					"10.98.177.115",
				},
				"externalTrafficPolicy": "Cluster",
				"ipFamilies": []string{
					"IPv4",
				},
				"ipFamilyPolicy": "SingleStack",
				"ports": []interface{}{
					map[string]interface{}{
						"name":       "tcp-80-80",
						"port":       int64(80),
						"protocol":   "TCP",
						"targetPort": int64(80),
					},
					map[string]interface{}{
						"name":       "tcp-443-443",
						"port":       int64(443),
						"protocol":   "TCP",
						"targetPort": int64(443),
					},
				},
			},
		},
	}
	barZ := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"externalTrafficPolicy": "Cluster",
				"ipFamilies": []string{
					"IPv6", // here is the difference
				},
				"ipFamilyPolicy": "SingleStack",
				"ports": []interface{}{
					map[string]interface{}{
						"name":       "tcp-80-80",
						"port":       int64(80),
						"protocol":   "TCP",
						"targetPort": int64(80),
					},
					map[string]interface{}{
						"name":       "tcp-443-443",
						"port":       int64(443),
						"protocol":   "TCP",
						"targetPort": int64(443),
					},
				},
			},
		},
	}
	barU := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"externalTrafficPolicy": "Cluster",
				"ipFamilies": []string{
					"IPv4",
				},
				"ipFamilyPolicy": "SingleStack",
				"ports": []interface{}{
					map[string]interface{}{
						"name":       "tcp-80-80",
						"port":       int64(80),
						"protocol":   "TCP",
						"targetPort": int64(80),
					},
					map[string]interface{}{
						"name":       "tcp-443-443",
						"port":       int64(443),
						"protocol":   "TCP",
						"targetPort": int64(443),
					},
				},
			},
			"status": map[string]interface{}{
				"availableReplicas":  2,
				"observedGeneration": 1,
			},
		},
	}
	barV := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"externalTrafficPolicy": "Cluster",
				"ipFamilies": []string{
					"IPv4",
				},
				"ipFamilyPolicy": "SingleStack",
				"ports": []interface{}{
					map[string]interface{}{
						"name":       "tcp-80-80",
						"port":       int64(80),
						"protocol":   "TCP",
						"targetPort": int64(80),
					},
					map[string]interface{}{
						"name":       "tcp-443-443",
						"port":       int64(443),
						"protocol":   "TCP",
						"targetPort": int64(443),
					},
				},
			},
			"status": map[string]interface{}{
				"availableReplicas":  2,
				"observedGeneration": 1000,
			},
		},
	}

	tests := []struct {
		label     string // Test name
		x         *unstructured.Unstructured
		y         *unstructured.Unstructured
		wantSync  bool   // Whether the inputs are equal
		reason    string // The reason for the expected outcome
		ignoreAdd bool   // whether or not ignore add action.
	}{
		{
			label:     "fields-populated",
			x:         barX,
			y:         barY,
			wantSync:  false,
			reason:    "won't re-sync because fields are auto populated",
			ignoreAdd: true,
		},
		{
			label:     "fields-removed",
			x:         barY,
			y:         barX,
			wantSync:  true,
			reason:    "should re-sync because fields are removed",
			ignoreAdd: false,
		},
		{
			label:     "fields-changed",
			x:         barX,
			y:         barZ,
			wantSync:  true,
			reason:    "should re-sync because fields get changed",
			ignoreAdd: false,
		},
		{
			label:     "fields-added",
			x:         barX,
			y:         barY,
			wantSync:  true,
			reason:    "should re-sync because add action are not ignored",
			ignoreAdd: false,
		},
		{
			label:     "section-ignored",
			x:         barX,
			y:         barU,
			wantSync:  false,
			reason:    "won't re-sync because status section is ignored",
			ignoreAdd: true,
		},
		{
			label:     "section-ignored",
			x:         barU,
			y:         barV,
			wantSync:  false,
			reason:    "won't re-sync because status section is ignored",
			ignoreAdd: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			var gotEqual bool
			func() {
				gotEqual = ResourceNeedResync(tt.x, tt.y, tt.ignoreAdd)
			}()
			switch {
			case tt.reason == "":
				t.Errorf("reason must be provided")
			case gotEqual != tt.wantSync:
				t.Errorf("Equal = %v, want %v\nreason: %v", gotEqual, tt.wantSync, tt.reason)
			}
		})
	}
}

func TestProtectAutoGeneratedResources(t *testing.T) {
	// create a fake clientset
	objs := []runtime.Object{}
	fakeClientset := fake.NewSimpleClientset(objs...)

	// set up subscription test data
	mockSubs := map[string]*v1alpha1.Subscription{
		"deleting-sub": {
			ObjectMeta: metav1.ObjectMeta{
				Name:              "deleting-sub",
				Namespace:         "deleting-sub-ns",
				DeletionTimestamp: &metav1.Time{},
			},
			Status: v1alpha1.SubscriptionStatus{
				BindingClusters: []string{"mcls1-ns/mcls1", "mcls2-ns/mcls2"},
			},
		},
		"unbound-sub": {
			ObjectMeta: metav1.ObjectMeta{
				Name:              "unbound-sub",
				Namespace:         "unbound-sub-ns",
				DeletionTimestamp: nil,
			},
			Status: v1alpha1.SubscriptionStatus{
				BindingClusters: []string{"mcls1-ns/mcls1"},
			},
		},
		"block-sub": {
			ObjectMeta: metav1.ObjectMeta{
				Name:              "block-sub",
				Namespace:         "block-sub-ns",
				DeletionTimestamp: nil,
			},
			Status: v1alpha1.SubscriptionStatus{
				BindingClusters: []string{"mcls1-ns/mcls1", "mcls2-ns/mcls2"},
			},
		},
	}
	for _, sub := range mockSubs {
		_, err := fakeClientset.AppsV1alpha1().Subscriptions(sub.Namespace).Create(context.TODO(), sub, metav1.CreateOptions{})
		assert.NoError(t, err)
	}

	testCases := []struct {
		name     string
		subRef   klog.ObjectRef
		mclsRef  klog.ObjectRef
		expected error
	}{
		{
			name:     "don't block -- should return nil if subscription is not found",
			subRef:   klog.KRef("non-existent-sub-ns", "non-existent-sub"),
			mclsRef:  klog.KRef("mcls1-ns", "mcls1"),
			expected: nil,
		},
		{
			name:     "don't block -- should return nil if subscription is marked for deletion",
			subRef:   klog.KRef("deleting-sub-ns", "deleting-sub"),
			mclsRef:  klog.KRef("mcls1-ns", "mcls1"),
			expected: nil,
		},
		{
			name:     "don't block -- should return nil if resource is unbound from Subscription",
			subRef:   klog.KRef("unbound-sub-ns", "unbound-sub"),
			mclsRef:  klog.KRef("mcls2-ns", "mcls2"),
			expected: nil,
		},
		{
			name:     "block -- should return error if subscription is not marked for deletion and resource bound to specified ManagedClusterSet",
			subRef:   klog.KRef("block-sub-ns", "block-sub"),
			mclsRef:  klog.KRef("mcls2-ns", "mcls2"),
			expected: errors.New("block deleting current resource until subscription block-sub-ns/block-sub get deleted or unbind from subscription"),
		},
	}
	// Run tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ProtectAutoGeneratedResources(fakeClientset, tc.mclsRef, tc.subRef)
			if tc.expected == nil {
				assert.Equal(t, tc.expected, result)
			} else {
				assert.EqualError(t, result, tc.expected.Error())
			}
		})
	}
}
