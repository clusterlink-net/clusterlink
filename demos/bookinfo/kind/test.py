#!/usr/bin/env python3
# Copyright (c) The ClusterLink Authors.
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

# Copyright (c) 2022 The ClusterLink Authors.
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

# Copyright (C) The ClusterLink Authors.
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

##############################################################################################
# Name: Bookinfo
# Info: support bookinfo application with gwctl inside the clusters
#       In this we create three kind clusters
#       1) cluster1- contain gw, gwctl,product and details microservices (bookinfo services)
#       2) cluster2- contain gw, gwctl, review-v2 and rating microservices (bookinfo services)
#       3) cluster3- contain gw, gwctl, review-v3 and rating microservices (bookinfo services)
##############################################################################################

import os
import sys
import argparse
projDir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{projDir}')

from demos.utils.common import printHeader
from demos.utils.kind import Cluster
from demos.bookinfo.test import bookInfoDemo

testOutputFolder = f"{projDir}/bin/tests/bookinfo"

############################### MAIN ##########################
if __name__ == "__main__":
   parser = argparse.ArgumentParser(description='Description of your program')
   parser.add_argument('-l','--logLevel', help='The log level. One of fatal, error, warn, info, debug.', required=False, default="info")
   parser.add_argument('-d','--dataplane', help='Which dataplane to use envoy/go', required=False, default="envoy")
   args = vars(parser.parse_args())

   printHeader("\n\nStart Kind Test\n\n")
   #GW parameters
   cl1           = Cluster(name="peer1")
   cl2           = Cluster(name="peer2")
   cl3           = Cluster(name="peer3")

   bookInfoDemo(cl1, cl2, cl3, testOutputFolder, args["logLevel"], args["dataplane"])
   print(f"Productpage1 url: http://{cl1.ip}:30001/productpage")
   print(f"Productpage2 url: http://{cl1.ip}:30002/productpage")

