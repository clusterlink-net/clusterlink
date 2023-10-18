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

import argparse,json
import subprocess as sp


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Rturn the local port of the service inside the MBG')
    parser.add_argument('-s','--service', help='Service local port inside the MBG', required=True, default="")
    parser.add_argument('-m','--mbg', help='MBG pod name', required=True, default="")
    args = vars(parser.parse_args())

    mbgJson=json.loads(sp.getoutput(f'kubectl exec -i {args["mbg"]} -- cat ./root/.mbg/mbgApp'))
    localPort =(mbgJson["Connections"][args["service"]]["Local"]).split(":")[1]
    externalPort =(mbgJson["Connections"][args["service"]]["External"]).split(":")[1]
    
    print(localPort)