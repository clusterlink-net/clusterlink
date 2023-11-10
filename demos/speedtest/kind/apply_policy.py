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
from demos.utils.common import runcmd, printHeader

srcSvc   = "firefox"
destSvc  = "openspeedtest"
denyIAccessPolicy=f"{projDir}/demos/speedtest/testdata/policy/denyToSpeedtest.json"
denyCluster3Policy=f"{projDir}/demos/speedtest/testdata/policy/denyFromGw.json"    

def applyAccessPolicy(cl, policyFile):
    printHeader(f"\n\nApplying policy file {policyFile} to {cl}")
    runcmd(f'gwctl --myid {cl} create policy --type access --policyFile {policyFile}')

def deleteAccessPolicy(cl, policyFile):
    runcmd(f'gwctl delete policy --myid {cl} --type access --policyFile {policyFile}')
    
def applyPolicy(cl, type):
    if type == "show":
        printHeader(f"Show Policies in {cl}")
        runcmd(f'gwctl get policy --myid {cl}')
        return
    
    if cl in ["peer1","peer3"]:
        if type == "deny":
            printHeader(f"Block Traffic in {cl}")
            applyAccessPolicy(cl, denyIAccessPolicy)
        elif type == "allow": # Remove the deny policy
            printHeader(f"Allow Traffic in {cl}")
            deleteAccessPolicy(cl, denyIAccessPolicy)
        else:
            print("Unknown command")
    if cl == "peer2":
        if type == "deny":
            printHeader(f"Block Traffic in {cl}")
            applyAccessPolicy(cl, denyCluster3Policy)
        elif type == "allow": # Remove the deny policy
            printHeader(f"Allow Traffic in {cl}")
            deleteAccessPolicy(cl, denyCluster3Policy)
        else:
            print("Unknown command")


############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-p','--peer', help='Either peer1/peer2/peer3', required=True, default="peer1")
    parser.add_argument('-t','--type', help='Either allow/deny/show', required=False, default="allow")

    args = vars(parser.parse_args())

    cl = args["peer"]
    type = args["type"]


    print(f'Working directory {projDir}')
    os.chdir(projDir)

    applyPolicy(cl, type)
