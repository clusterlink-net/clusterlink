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

################################################################
#Name: time_func 
#Desc: Functions to calculate tests duration
################################################################
import time
from datetime import datetime

def test_start_time():
    start_time = datetime.now()
    start_time_s = start_time.strftime("%H:%M:%S")
    return start_time
    
def test_end_time(start_time):
    end_time    = datetime.now()
    end_time_s  = end_time.strftime("%H:%M:%S")
    test_time_s =end_time-start_time
    print("Test start {} Test end {} total test time {}".format(start_time,end_time, test_time_s))