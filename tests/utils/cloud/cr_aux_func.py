################################################################
#Name: step_aux_func 
#Desc: contain auxiliary functions for managing container image 
#      on each type of platform (IBM,GCP)          
#
################################################################

from PROJECT_PARAMS import GOOGLE_CONT_REGESTRY , IBM_CONT_REGESTRY
import os

def get_plarform_container_reg(platform):
  if (platform == "gcp"):
    container_reg = GOOGLE_CONT_REGESTRY
  elif (platform == "ibm"):
    container_reg = IBM_CONT_REGESTRY
  else:
    print("Plarform is not supported")
    exit(1)
  return  container_reg


def connect_platform_container_reg(platform):
  if (platform == "gcp"):
    print("connect to gcp container registry")
  elif (platform == "ibm"):
    print("connect to IBM container registry")
    os.system("ibmcloud cr login") #conect to ibm docker
  else:
    print("Plarform is not supported")
    exit(1)


#replace the container registry ip according to the proxy platform.
def replace_source_image(file,image,platform):
    f = open(file, "r")
    lines = f.readlines()
    for idx,line in enumerate(lines):
        if ("image:" in line) and (image in line):
          container_reg= get_plarform_container_reg(platform)
          line_s=line.split(':')[0]
          lines[idx] =line_s+": {}/{} \n".format(container_reg,image)
    f.close()
    f = open(file, "w")
    f.writelines(lines)
    f.close()