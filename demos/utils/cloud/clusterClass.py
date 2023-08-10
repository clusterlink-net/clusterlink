import json
import subprocess as sp
class cluster:
  def __init__(self, name, zone,platform,type):
    self.name     = name
    self.zone     = zone
    self.platform = platform
    self.type     = type
    self.ip       = ""
  
  def setClusterIP(self):
    clJson=json.loads(sp.getoutput(f' kubectl get nodes -o json'))
    ip = clJson["items"][0]["status"]["addresses"][1]["address"]
    self.ip = ip