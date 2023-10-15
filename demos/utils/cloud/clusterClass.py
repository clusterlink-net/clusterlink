# Copyright 2023 The ClusterLink Authors.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import json
import subprocess as sp
class cluster:
  def __init__(self, name, zone,platform,type):
    self.name     = name
    self.zone     = zone
    self.platform = platform
    self.type     = type
    self.ip       = ""
  
  def setClusterIP(self):
    clJson=json.loads(sp.getoutput(f' kubectl get nodes -o json'))
    ip = clJson["items"][0]["status"]["addresses"][1]["address"]
    self.ip = ip