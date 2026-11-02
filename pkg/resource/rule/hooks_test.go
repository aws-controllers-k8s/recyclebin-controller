// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package rule

import (
	"testing"

	svcapitypes "github.com/aws-controllers-k8s/recyclebin-controller/apis/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	"github.com/aws/aws-sdk-go-v2/aws"
)

func ruleWithTags(tags ...*svcapitypes.Tag) *resource {
	return &resource{
		ko: &svcapitypes.Rule{
			Spec: svcapitypes.RuleSpec{Tags: tags},
		},
	}
}

func ruleTag(k, v string) *svcapitypes.Tag {
	return &svcapitypes.Tag{Key: aws.String(k), Value: aws.String(v)}
}

func TestCustomPreCompareTags(t *testing.T) {
	cases := []struct {
		name      string
		a         *resource // desired
		b         *resource // latest
		wantDelta bool
	}{
		{
			name:      "no-op: identical tags in different order",
			a:         ruleWithTags(ruleTag("env", "prod"), ruleTag("team", "x")),
			b:         ruleWithTags(ruleTag("team", "x"), ruleTag("env", "prod")),
			wantDelta: false,
		},
		{
			name:      "add-only: desired has extra tag",
			a:         ruleWithTags(ruleTag("env", "prod"), ruleTag("team", "x")),
			b:         ruleWithTags(ruleTag("env", "prod")),
			wantDelta: true,
		},
		{
			name:      "remove-only: latest has extra tag",
			a:         ruleWithTags(ruleTag("env", "prod")),
			b:         ruleWithTags(ruleTag("env", "prod"), ruleTag("old", "v1")),
			wantDelta: true,
		},
		{
			name:      "value-change: same key different value",
			a:         ruleWithTags(ruleTag("env", "prod")),
			b:         ruleWithTags(ruleTag("env", "staging")),
			wantDelta: true,
		},
		{
			name:      "simultaneous add and remove",
			a:         ruleWithTags(ruleTag("env", "prod"), ruleTag("new", "v2")),
			b:         ruleWithTags(ruleTag("env", "prod"), ruleTag("old", "v1")),
			wantDelta: true,
		},
		{
			name:      "no-op: both nil",
			a:         ruleWithTags(),
			b:         ruleWithTags(),
			wantDelta: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			delta := ackcompare.NewDelta()
			customPreCompare(delta, tc.a, tc.b)
			got := delta.DifferentAt("Spec.Tags")
			if got != tc.wantDelta {
				t.Errorf("customPreCompare() DifferentAt(Spec.Tags) = %v, want %v", got, tc.wantDelta)
			}
		})
	}
}
