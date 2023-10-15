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
import shutil

projDir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))))
pluginsFol=f'{projDir}/bin/plugins/'

#Download plugins bridge for flannel
def createCniBridge():   
    if not os.path.exists(f"{pluginsFol}/bin/bridge"):
        print("Start building plugins for flannel")
        os.makedirs(pluginsFol, exist_ok=True)
        os.chdir(pluginsFol)
        os.system('git clone https://github.com/containernetworking/plugins.git')
        os.chdir(f'{pluginsFol}/plugins')
        os.system(f'./build_linux.sh')
        os.chdir(projDir)
        shutil.copytree(f'{projDir}/bin/plugins/plugins/bin', f'{projDir}/bin/plugins/bin')
        shutil.rmtree(f'{pluginsFol}/plugins')
        os.chdir(projDir)
    else:
        print(f"file {projDir}/bin/plugins/bridge exist")

#Create kind config file with plugins for flannel
def createKindCfgForflunnel():
    cfgFile=f'{pluginsFol}/flannel-config.yaml'
    if not os.path.exists(cfgFile):
        with open(f"{projDir}/demos/utils/manifests/kind/flannel/flannel-config.yaml", 'r') as file:
            lines = file.readlines()

        # replace the line you want to modify (line 3 in this example)
        lines[12] = f'  - hostPath: {projDir}/bin/plugins/bin \n'

        with open(cfgFile, 'w') as file:
            file.writelines(lines)        

if __name__ == "__main__":
    createCniBridge()
    createKindCfgForflunnel()