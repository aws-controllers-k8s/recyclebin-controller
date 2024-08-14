# Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You may
# not use this file except in compliance with the License. A copy of the
# License is located at
#
#	 http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.

"""Helper functions for RecycleBin e2e tests
"""

import logging

class RecycleBinHelper:
    def __init__(self, recycle_binclient):
        self.recycle_binclient = recycle_binclient

    def get_rule(self, identifier: str) -> dict:
        try:
            resp = self.recycle_binclient.get_rule(
                Identifier=identifier
            )
            return resp

        except Exception as e:
            logging.debug(e)
            return None

    def get_resource_tags(self, rule_arn: str):
        resource_tags = self.recycle_binclient.list_tags_for_resource(
            ResourceArn=rule_arn,
        )
        return resource_tags['Tags']

    def rule_exists(self, identifier) -> bool:
        return self.get_rule(identifier) is not None