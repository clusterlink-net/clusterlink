from tests.utils.mbgAux import runcmd

def useKindCluster(name):
    runcmd(f'kubectl config use-context kind-{name}')
