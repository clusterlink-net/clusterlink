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

import os
import sys
import argparse

projDir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{projDir}')
sys.path.insert(1,f'{projDir}/demos/utils/cloud/')
from demos.bookinfo.test import apply_failover
from demos.utils.cloud import Cluster
testOutputFolder = f"{projDir}/bin/tests/bookinfo"

# cl3 parameters
cl3gcp = Cluster(name="peer3", zone = "us-east4-b"   , platform = "gcp") # Virginia
cl3ibm = Cluster(name="peer3", zone = "wdc04"        , platform = "ibm") # Washington DC

############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-t','--type', help='Either fail/start', required=False, default="fail")
    parser.add_argument('-cloud','--cloud', help='Cloud setup using gcp/ibm', required=False, default="ibm")
    args = vars(parser.parse_args())
    print(f'Working directory {projDir}')
    os.chdir(projDir)
    cl3 = cl3gcp if args["cloud"] in ["gcp"] else cl3ibm
    apply_failover(cl3, args["type"], testOutputFolder)
