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

package tags

import (
	"context"
	"reflect"
	"testing"

	svcsdk "github.com/aws/aws-sdk-go-v2/service/rbin"

	svcapitypes "github.com/aws-controllers-k8s/recyclebin-controller/apis/v1alpha1"
)

type fakeTagsClient struct {
	tagCalls   []*svcsdk.TagResourceInput
	untagCalls []*svcsdk.UntagResourceInput
}

func (f *fakeTagsClient) TagResource(
	_ context.Context,
	in *svcsdk.TagResourceInput,
	_ ...func(*svcsdk.Options),
) (*svcsdk.TagResourceOutput, error) {
	f.tagCalls = append(f.tagCalls, in)
	return &svcsdk.TagResourceOutput{}, nil
}

func (f *fakeTagsClient) ListTagsForResource(
	_ context.Context,
	_ *svcsdk.ListTagsForResourceInput,
	_ ...func(*svcsdk.Options),
) (*svcsdk.ListTagsForResourceOutput, error) {
	return &svcsdk.ListTagsForResourceOutput{}, nil
}

func (f *fakeTagsClient) UntagResource(
	_ context.Context,
	in *svcsdk.UntagResourceInput,
	_ ...func(*svcsdk.Options),
) (*svcsdk.UntagResourceOutput, error) {
	f.untagCalls = append(f.untagCalls, in)
	return &svcsdk.UntagResourceOutput{}, nil
}

type noopMetrics struct{}

func (noopMetrics) RecordAPICall(string, string, error) {}

func strPtr(s string) *string {
	return &s
}

func tag(k, v string) *svcapitypes.Tag {
	return &svcapitypes.Tag{Key: strPtr(k), Value: strPtr(v)}
}

func TestSyncResourceTags(t *testing.T) {
	const arn = "arn:aws:rbin:us-east-1:123456789012:rule/abc"

	tests := []struct {
		name         string
		desired      []*svcapitypes.Tag
		latest       []*svcapitypes.Tag
		wantTagKeys  map[string]string
		wantUntagKey []string
	}{
		{
			name:    "no-op when tags are identical",
			desired: []*svcapitypes.Tag{tag("a", "1"), tag("b", "2")},
			latest:  []*svcapitypes.Tag{tag("b", "2"), tag("a", "1")},
		},
		{
			name:        "add new tag",
			desired:     []*svcapitypes.Tag{tag("a", "1"), tag("b", "2")},
			latest:      []*svcapitypes.Tag{tag("a", "1")},
			wantTagKeys: map[string]string{"b": "2"},
		},
		{
			name:         "remove tag",
			desired:      []*svcapitypes.Tag{tag("a", "1")},
			latest:       []*svcapitypes.Tag{tag("a", "1"), tag("b", "2")},
			wantUntagKey: []string{"b"},
		},
		{
			name:        "value change only calls tag, not untag",
			desired:     []*svcapitypes.Tag{tag("a", "new")},
			latest:      []*svcapitypes.Tag{tag("a", "old")},
			wantTagKeys: map[string]string{"a": "new"},
		},
		{
			name:         "add and remove simultaneously",
			desired:      []*svcapitypes.Tag{tag("a", "1"), tag("b", "2")},
			latest:       []*svcapitypes.Tag{tag("a", "1"), tag("c", "3")},
			wantTagKeys:  map[string]string{"b": "2"},
			wantUntagKey: []string{"c"},
		},
		{
			// nil-key entries are dropped by toACKTags; both sides reduce to {"a":"1"},
			// so GetTagsDifference returns no changes and neither API is called.
			name:         "nil key is skipped and results in no API calls",
			desired:      []*svcapitypes.Tag{{Key: nil, Value: strPtr("x")}, tag("a", "1")},
			latest:       []*svcapitypes.Tag{{Key: nil}, tag("a", "1")},
			wantTagKeys:  nil,
			wantUntagKey: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &fakeTagsClient{}
			err := SyncResourceTags(
				context.Background(),
				client,
				noopMetrics{},
				arn,
				tc.desired,
				tc.latest,
			)
			if err != nil {
				t.Fatalf("SyncResourceTags returned error: %v", err)
			}

			gotTags := map[string]string{}
			for _, call := range client.tagCalls {
				if call.ResourceArn == nil || *call.ResourceArn != arn {
					t.Errorf("TagResource called with ARN %v, want %s", call.ResourceArn, arn)
				}
				for _, tg := range call.Tags {
					gotTags[*tg.Key] = *tg.Value
				}
			}
			if len(gotTags) == 0 {
				gotTags = nil
			}
			if !reflect.DeepEqual(gotTags, tc.wantTagKeys) {
				t.Errorf("TagResource tags = %v, want %v", gotTags, tc.wantTagKeys)
			}

			var gotUntag []string
			for _, call := range client.untagCalls {
				gotUntag = append(gotUntag, call.TagKeys...)
			}
			if !reflect.DeepEqual(gotUntag, tc.wantUntagKey) {
				t.Errorf("UntagResource keys = %v, want %v", gotUntag, tc.wantUntagKey)
			}
		})
	}
}
