#!/usr/bin/env python3
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

import os
import sys
import argparse

projDir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{projDir}')
from demos.bookinfo.test import applyPolicy
from demos.utils.kind import Cluster

############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-p','--peer', help='Either peer1/peer2/peer3', required=False, default="peer1")
    parser.add_argument('-t','--type', help='Either round-robin/same/diff/clean/show', required=False, default="round-robin")

    args = vars(parser.parse_args())
    print(f'Working directory {projDir}')
    os.chdir(projDir)
    cl = Cluster(name=args["peer"])
    cl.set_kube_config()
    applyPolicy(cl, args["type"])
