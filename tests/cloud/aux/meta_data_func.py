################################################################
#Name: meta_data_func 
#Desc: contain functions to mange metadata.json files 
#      (update, delete , get data)
#      the metadat.json file contain information on 
#      each server: type hot,client, proxy)
#      ip information, platform and regions
################################################################

import json 
import os
from clusterClass import cluster

def getIpPort(file,cluster):
    ip_pfx   = "ip"   if cluster.type in ["target","host"] else cluster.type+"_ip"
    port_pfx = "port" if cluster.type in ["target","host"] else cluster.type+"_port"    

    with open(file) as json_file:
        data = json.load(json_file)    
        for key in data.keys():
            if (cluster.name in key) and (cluster.zone in key ):
                ip   = data[key][ip_pfx]
                port = data[key][port_pfx]
        return ip, port

def get_platform(file,name,zone):
    platform="Not exist"
    with open(file) as json_file:
        data = json.load(json_file)    
        for key in data.keys():
            if (name in key) and (zone in key):
                platform = data[key]["cluster_platform"]
        return platform
    return platform

def delete_cluster_meta_data(file,name,zone):
    with open(file) as json_file:
        data = json.load(json_file)    
        data.pop(name+"_"+zone)
    json_file.close()
    #rewrite json
    with open(file, 'w') as f:
        json.dump(data, f,sort_keys=True, indent=4, separators=(',', ': '))

def update_metadata(file,cluster_name,dic_val):
    data = {}
    if os.path.isfile(file) and (os.stat(file).st_size != 0): 
        with open(file) as json_file:
            data = json.load(json_file)

    data.update({cluster_name: dic_val})

    with open(file, 'w+') as f:
        json.dump(data, f,sort_keys=True, indent=4, separators=(',', ': '))
    
def get_first_item_metadata(file):
    with open(file) as json_file:
        data = json.load(json_file)    
        first_key =list(data.keys())[0]
        name     = data[first_key]["cluster_name"]
        zone     = data[first_key]["cluster_zone"]
        platform = data[first_key]["cluster_platform"]
    return name, zone, platform

def is_empty_metadata(file):
    with open(file) as json_file:
        data = json.load(json_file)    
        if data :
            return False
        else:
            return True
def len_metadata(file):
    with open(file) as json_file:
        data = json.load(json_file)    
        return len(data.keys())


