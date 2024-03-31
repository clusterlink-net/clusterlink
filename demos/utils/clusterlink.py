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

from kubernetes import client, config
from demos.utils.common import createFolder,runcmdDir,ProjDir
from demos.utils.k8s import waitPod

CL_CLI    = ProjDir + "/bin/clusterlink "
CLUSTELINK_NS="clusterlink-system"
CLUSTELINK_OPERATOR_NS="clusterlink-operator"


class CRDObject:
    def __init__(self):
        self.group = 'clusterlink.net'
        self.version = 'v1alpha1'
        self.api_instance = None

    # set_kube_config load a kubeconfig file and create the Kubernetes API client.
    def set_kube_config(self):
        config.load_kube_config()
        self.api_instance = client.CustomObjectsApi()

    def create_object(self, kind, name, namespace, body):
        try:
            self.api_instance.create_namespaced_custom_object(self.group, self.version, namespace, kind, body)
            print(f"{kind} '{name}' created successfully.")
        except Exception as e:
            print(f"Error creating {kind} '{name}': {e}")

    def update_object(self, kind, name, namespace, body):
        try:
            self.api_instance.patch_namespaced_custom_object(self.group, self.version, namespace, kind, name, body)
            print(f"{kind} '{name}' updated successfully.")
        except Exception as e:
            print(f"Error updating {kind} '{name}': {e}")

    def delete_object(self, kind, name, namespace):
        try:
            self.api_instance.delete_namespaced_custom_object(self.group, self.version, namespace, kind, name)
            print(f"{kind} '{name}' deleted successfully.")
        except Exception as e:
            print(f"Error deleting {kind} '{name}': {e}")

# Imports Peers contains all the commands for managing peer CRD.
class Peers(CRDObject):
    def __init__(self, namespace):
        self.namespace=namespace
        super().__init__()

    def set_peer_object(self, name, host, port):
        peer_object = {
            "apiVersion": f"{self.group}/{self.version}",
            "kind": "Peer",
            "metadata": {"name": name, "namespace": self.namespace},
            "spec": {
                "gateways": [{"host": host, "port": port}]
            }
        }
        return peer_object

    def create(self, name, host, port):
        peer_object = self.set_peer_object(name, host, port)
        self.create_object('peers', name, self.namespace, peer_object)

    def update_peer(self, name, host, port):
        peer_object = self.set_peer_object(name, host, port)
        self.update_object('peers', name, self.namespace, peer_object)

    def delete_peer(self, name):
        self.delete_object('peers', name, self.namespace)

# Imports class contains all the commands for managing import CRD.
class Imports(CRDObject):
    def __init__(self):
        super().__init__()

    def set_import_object(self, name, namespace, port, peer_name, export_name, export_namespace, load_balancing=""):
        import_object = {
            "apiVersion": "clusterlink.net/v1alpha1",
            "kind": "Import",
            "metadata": {"name": name, "namespace": namespace},
            "spec": {
                "port": port,
                "sources": []
            }
        }

        # Check if peer_name, export_name, and export_namespace are lists and have the same length.
        if isinstance(peer_name, list) and isinstance(export_name, list) and isinstance(export_namespace, list):
            if len(peer_name) == len(export_name) == len(export_namespace):
                for idx in range(len(peer_name)):
                    import_object["spec"]["sources"].append({
                        "peer": peer_name[idx],
                        "exportName": export_name[idx],
                        "exportNamespace": export_namespace[idx]
                    })
            else:
                raise ValueError("peer_name, export_name, and export_namespace must have the same length")
        else:
            import_object["spec"]["sources"].append({
                "peer": peer_name,
                "exportName": export_name,
                "exportNamespace": export_namespace
            })
        if load_balancing != "":
            import_object["spec"]["lbScheme"] = load_balancing

        return import_object

    def create(self, name, namespace, port, peer_name, export_name, export_namespace, load_balancing=""):
        import_object = self.set_import_object(name, namespace, port, peer_name, export_name, export_namespace, load_balancing)
        self.create_object('imports', name, namespace, import_object)

    def update(self, name, namespace, port, peer_name, export_name, export_namespace, load_balancing=""):
        import_object = self.set_import_object(name, namespace, port, peer_name, export_name, export_namespace, load_balancing)
        self.update_object('imports', name, namespace, import_object)

    def delete(self, name, namespace):
        self.delete_object('imports', name, namespace)

# Exports class contains all the commands for managing export CRD.
class Exports(CRDObject):
    def __init__(self):
        super().__init__()

    def set_export_object(self, name, namespace, port, host=""):
        export_object = {
            "apiVersion": f"{self.group}/{self.version}",
            "kind": "Export",
            "metadata": {"name": name, "namespace": namespace},
            "spec": {
                "host": host,
                "port": port
            }
        }
        return export_object

    def create(self, name, namespace, port, host=""):
        export_object = self.set_export_object(name, namespace, port, host)
        self.create_object('exports', name, namespace, export_object)

    def update(self, name, namespace, port, host=""):
        export_object = self.set_export_object(name, namespace, port, host)
        self.update_object('exports', name, namespace, export_object)

    def delete(self, name, namespace):
        self.delete_object('exports', name, namespace)

# Policies class contains all the commands for managing policy CRD.
class Policies(CRDObject):
    def __init__(self):
        super().__init__()

    def set_policy_object(self, name, namespace, action, from_attribute, to_attribute):
        policy_object = {
            "apiVersion": f"{self.group}/{self.version}",
            "kind": "AccessPolicy",
            "metadata": {"name": name, "namespace": namespace},
            "spec": {
                "action": action,
                "from": from_attribute,
                "to": to_attribute
            }
        }
        return policy_object

    def create(self, name, namespace, action, from_attribute, to_attribute):
        policy_object = self.set_policy_object(name, namespace, action, from_attribute, to_attribute)
        self.create_object('accesspolicies', name, namespace, policy_object)

    def update(self, name, namespace, action, from_attribute, to_attribute):
        policy_object = self.set_policy_object(name, namespace, action, from_attribute, to_attribute)
        self.update_object('accesspolicies', name, namespace, policy_object)

    def delete(self, name, namespace):
        self.delete_object('accesspolicies', name, namespace)

class PrivilegedPolicies(CRDObject):
    def __init__(self):
        super().__init__()

    def set_policy_object(self, name, namespace, action, from_attribute, to_attribute):
        policy_object = {
            "apiVersion": f"{self.group}/{self.version}",
            "kind": "PrivilegedAccessPolicy",
            "metadata": {"name": name, "namespace": namespace},
            "spec": {
                "action": action,
                "from": from_attribute,
                "to": to_attribute
            }
        }
        return policy_object

    def create(self, name, namespace, action, from_attribute, to_attribute):
        policy_object = self.set_policy_object(name, namespace, action, from_attribute, to_attribute)
        self.create_object('privilegedaccesspolicies', name, namespace, policy_object)

    def update(self, name, namespace, action, from_attribute, to_attribute):
        policy_object = self.set_policy_object(name, namespace, action, from_attribute, to_attribute)
        self.update_object('privilegedaccesspolicies', name, namespace, policy_object)

    def delete(self, name, namespace):
        self.delete_object('privilegedaccesspolicies', name, namespace)

# ClusterLink class contains all the commands for deploying ClusterLink CRDs.
class ClusterLink:
    def __init__(self, namespace):
        self.namespace = namespace
        self.peers    = Peers(namespace)
        self.exports  = Exports()
        self.imports  = Imports()
        self.policies = Policies()
        self.privileged_policies = PrivilegedPolicies()

    # set_kube_config set config for all objects.
    def set_kube_config(self):
        self.peers.set_kube_config()
        self.exports.set_kube_config()
        self.imports.set_kube_config()
        self.policies.set_kube_config()
        self.privileged_policies.set_kube_config()

    # create_fabric creates fabric certificates using ClusterLink CLI.
    def create_fabric(self, dir):
        createFolder(dir)
        runcmdDir(f"{CL_CLI} create fabric",dir)

    # create_peer_cert creates peer certificates and yaml using ClusterLink CLI.
    def create_peer_cert(self, name, dir, logLevel="info", dataplane="envoy", container_reg="", CRDMode=True):
        flag = f"--container-registry={container_reg} " if container_reg != "" else ""
        flag += "--crd-mode=true " if CRDMode else ""
        runcmdDir(f"{CL_CLI} create peer-cert --name {name} --log-level {logLevel} --dataplane-type {dataplane} {flag}",dir)

    # deploy_peer deploys clusterlink to the cluster using ClusterLink CLI.
    def deploy_peer(self, name, dir, container_reg ="", ingress_type="", ingress_port=0):
        flag = f"--container-registry={container_reg} " if container_reg != "" else ""
        flag += f"--ingress={ingress_type} " if ingress_type != "" else ""
        flag += f"--ingress-port={ingress_port} " if ingress_port != 0 else ""
        runcmdDir(f"{CL_CLI} deploy peer --name {name} --autostart  {flag} ",dir)
        waitPod("cl-controlplane", CLUSTELINK_NS)
        waitPod("cl-dataplane", CLUSTELINK_NS)
