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
import subprocess as sp
from colorama import Fore
from colorama import Style
ProjDir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

# Log Functions
# runcmd runs os system command.
def runcmd(cmd):
    print(f'{Fore.YELLOW}{cmd} {Style.RESET_ALL}')
    os.system(cmd)

# runcmdDir runs os system command in specific directory.
def runcmdDir(cmd,dir):
    print(f'{Fore.YELLOW}{cmd} {Style.RESET_ALL}')
    sp.run(cmd, shell=True, cwd=dir, check=True)

# runcmdb runs os system command in the background.
def runcmdb(cmd):
    print(f'{Fore.YELLOW}{cmd} {Style.RESET_ALL}')
    os.system(cmd + ' &')

# printHeader runs os system command in the background.
def printHeader(msg):
    print(f'{Fore.GREEN}{msg} {Style.RESET_ALL}')

# createFolder creates folder.
def createFolder(name):
    if os.path.exists(name):
        shutil.rmtree(name)
    os.makedirs(name)

# app cluster contains the application service information.
class app:
    def __init__(self, name, namespace, host, port):
        self.name      = name
        self.namespace = namespace
        self.host      = host
        self.port      = port