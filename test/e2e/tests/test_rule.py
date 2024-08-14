# Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You may
# not use this file except in compliance with the License. A copy of the
# License is located at
#
# 	 http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.

"""Integration tests for the RecycleBin Rule API.
"""

import pytest
import time
import logging

from acktest import tags
from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_recycle_binresource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.tests.helper import RecycleBinHelper

RESOURCE_PLURAL = "rules"

CREATE_WAIT_AFTER_SECONDS = 20
UPDATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 60

@pytest.fixture
def basic_rule():
    resource_name = random_suffix_name("rbin-rule", 24)

    replacements = REPLACEMENT_VALUES.copy()
    replacements["RULE_NAME"] = resource_name

    resource_data = load_recycle_binresource(
        "rule",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    # Create the k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)

    time.sleep(CREATE_WAIT_AFTER_SECONDS)

    # Get latest rule CR
    cr = k8s.wait_resource_consumed_by_controller(ref)

    assert cr is not None
    assert k8s.get_resource_exists(ref)

    yield (ref, cr)

    # Try to delete, if doesn't already exist
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
    except:
        pass

@service_marker
class TestRule:
    def test_basic(self, recycle_bin_client, basic_rule):
        (ref, cr) = basic_rule

        rule_arn = cr["status"]["ackResourceMetadata"]["arn"]
        rule_identifier = cr["status"]["identifier"]

        rbin_helper = RecycleBinHelper(recycle_bin_client)
        # verify that rule exists
        assert rbin_helper.rule_exists(rule_identifier)

        tags_dict = tags.to_dict(
            cr["spec"]["tags"],
            key_member_name = 'key',
            value_member_name = 'value'
        )
        rule_tags = rbin_helper.get_resource_tags(rule_arn)
        tags.assert_ack_system_tags(
            tags=rule_tags,
        )
        tags.assert_equal_without_ack_tags(
            actual=tags_dict,
            expected=rule_tags,
        )

        # Update retention period
        patch = {
            "spec": {
                "retentionPeriod": {
                    "retentionPeriodValue": 100
                }
            }
        }
        k8s.patch_custom_resource(ref, patch)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)
        cr = k8s.wait_resource_consumed_by_controller(ref)
        
        # Get rule from AWS
        rule = rbin_helper.get_rule(rule_identifier)
        assert rule is not None
        assert rule["RetentionPeriod"]["RetentionPeriodValue"] == 100
        

        # updates tags
        # deleting k1 and k2, updating k3 value and adding two new tags
        new_tags = [
            {
                "key": "k3",
                "value": "v3-new",
            },
            {
                "key": "k4",
                "value": "v4",
            },
            {
                "key": "k5",
                "value": "v5",
            }
        ]
        cr["spec"]["tags"] = new_tags
        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        tags_dict = tags.to_dict(
            cr["spec"]["tags"],
            key_member_name = 'key',
            value_member_name = 'value'
        )
        rule_tags = rbin_helper.get_resource_tags(rule_arn)
        tags.assert_ack_system_tags(
            tags=rule_tags,
        )
        tags.assert_equal_without_ack_tags(
            actual=tags_dict,
            expected=rule_tags,
        )

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check rule doesn't exist
        assert not rbin_helper.rule_exists(rule_identifier)