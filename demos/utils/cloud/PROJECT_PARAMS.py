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
#Name: PROJECT_PARAMS 
#Desc: contain project parameters
#       - location of metadata.json
#       - project id and container registry for each platform
################################################################
import os, sys
############################### Google cloud Parameters ##########################
GOOGLE_PROJECT_ID    = "multi-cloud-networks-4438"  #PROJECT_ID=sp.getoutput("gcloud info --format='value(config.project)'")
GOOGLE_CONT_REGESTRY = "gcr.io/" + GOOGLE_PROJECT_ID

############################### IBM cloud Parameters ##########################
IBM_NAMESPACE        = "k8s-ns"
IBM_CONT_REGESTRY    = "icr.io/" + IBM_NAMESPACE

############################### Project Parameters ##########################
PROJECT_PATH=os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.realpath(__file__)))))
METADATA_FILE= PROJECT_PATH + "/bin/metadata.json"

