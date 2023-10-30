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

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')
from demos.utils.common import runcmd, printHeader

srcSvc   = "firefox"
destSvc  = "openspeedtest"
denyIAccessPolicy=f"{proj_dir}/demos/speedtest/testdata/policy/denyToSpeedtest.json"
denyGw3Policy=f"{proj_dir}/demos/speedtest/testdata/policy/denyFromGw.json"    

def applyAccessPolicy(gw, policyFile):
    printHeader(f"\n\nApplying policy file {policyFile} to {gw}")
    runcmd(f'gwctl --myid {gw} create policy --type access --policyFile {policyFile}')

def deleteAccessPolicy(gw, policyFile):
    runcmd(f'gwctl delete policy --myid {gw} --type access --policyFile {policyFile}')
    
def applyPolicy(gw, type):
    if type == "show":
        printHeader(f"Show Policies in {gw}")
        runcmd(f'gwctl get policy --myid {gw}')
        return
    
    if gw in ["peer1","peer3"]:
        if type == "deny":
            printHeader(f"Block Traffic in {gw}")
            applyAccessPolicy(gw, denyIAccessPolicy)
        elif type == "allow": # Remove the deny policy
            printHeader(f"Allow Traffic in {gw}")
            deleteAccessPolicy(gw, denyIAccessPolicy)
        else:
            print("Unknown command")
    if gw == "peer2":
        if type == "deny":
            printHeader(f"Block Traffic in {gw}")
            applyAccessPolicy(gw, denyGw3Policy)
        elif type == "allow": # Remove the deny policy
            printHeader(f"Allow Traffic in {gw}")
            deleteAccessPolicy(gw, denyGw3Policy)
        else:
            print("Unknown command")


############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-p','--peer', help='Either peer1/peer2/peer3', required=True, default="peer1")
    parser.add_argument('-t','--type', help='Either allow/deny/show', required=False, default="allow")

    args = vars(parser.parse_args())

    gw = args["peer"]
    type = args["type"]


    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    applyPolicy(gw, type)
