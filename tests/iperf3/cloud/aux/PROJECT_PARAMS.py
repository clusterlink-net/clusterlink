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
PROJECT_PATH=os.path.dirname(os.path.dirname(os.path.dirname(os.path.realpath(__file__))))
METADATA_FILE= PROJECT_PATH + "/bin/metadata.json"

