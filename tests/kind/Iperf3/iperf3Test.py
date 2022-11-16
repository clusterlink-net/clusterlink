import os,sys,time
import subprocess as sp
import netifaces as ni

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))


def waitPod(name):
    start_cond="false"
    time.sleep(3) #Iinital start
    while(start_cond != "true"):
        cmd=f"kubectl get pods -l app={name} -o jsonpath" + "=\'{.items[0].status.containerStatuses[0].ready}\'"
        start_cond =sp.getoutput(cmd)
        if (start_cond != "true"):
            print (f"Waiting for pod  {name} to start")
            time.sleep(7)
        else:
            time.sleep(5)
            break

def getPodName(prefix):
    podName=sp.getoutput(f'kubectl get pods -o name | fgrep {prefix}| cut -d\'/\' -f2')
    return podName

def runcmd(cmd):
    print(cmd)
    os.system(cmd)

def getIp(Interface):
    ip = ni.ifaddresses(Interface)[ni.AF_INET][0]['addr']
    print(f"local Ip addrees:{ip}")
    return ip

############################### MAIN ##########################
if __name__ == "__main__":
    print("\n\nStart Kind Test\n\n")
    print("Start pre-setting")
    ipAddr=getIp("eth0")
    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)
    ### clean 
    print("Clean old kinds")
    os.system("make clean-kind-mbg")

    ###Run first Mbg
    print("\n\nStart building MBG1")
    os.system("make run-kind-mbg1")
    waitPod("mbg")
    podMbg1= getPodName("mbg")
    runcmd(f'kubectl exec -i {podMbg1} -- ./mbg start --id "mbg1" --ip {ipAddr} --cport "30100" --exposePortRange "30101" &')
    time.sleep(5)
    runcmd(f'kubectl exec -i {podMbg1} -- ./mbg addGw --id "hostGw" --ip {ipAddr}:30300')

    ###Run Second Mbg
    print("\n\nStart building MBG2")
    os.system("make run-kind-mbg2")
    waitPod("mbg")
    podMbg2 = getPodName("mbg")
    runcmd(f'kubectl exec -i {podMbg2} --  ./mbg start --id "mbg2" --ip {ipAddr} --cport "30200" --exposePortRange "30201" &')
    time.sleep(5)
    runcmd(f'kubectl exec -i {podMbg2} -- ./mbg addMbg --id "mbg1" --ip {ipAddr} --cport "30100"')
    runcmd(f'kubectl exec -i {podMbg2} -- ./mbg hello')
    runcmd(f'kubectl exec -i {podMbg2} -- ./mbg addGw --id "destGw" --ip {ipAddr}:30400')

    ###Run host
    print("\n\nStart building Gateway-host")
    os.system("make run-kind-host")
    waitPod("gateway-mbg")
    podhost= getPodName("gateway-mbg")
    runcmd(f'kubectl exec -i {podhost} -- ./gateway start --id "hostGw"  --ip {ipAddr} --mbgIP {ipAddr}:30100 &')
    runcmd(f'kubectl exec -i {podhost} -- ./gateway addService --serviceId iperfIsrael --serviceIp :5001')

    ###Run dest
    print("\n\nStart building Gateway-destination")
    os.system("make run-kind-dest")
    waitPod("gateway-mbg")
    podest= getPodName("gateway-mbg")
    runcmd(f'kubectl exec -i {podest} -- ./gateway start --id "destGw"  --ip {ipAddr} --mbgIP {ipAddr}:30200 &')
    runcmd(f'kubectl exec -i {podest} -- ./gateway addService --serviceId iperfIndia --serviceIp {ipAddr}:30401')

    # #Expose service
    print("\n\nStart exposing connection")
    runcmd(f'kubectl exec -i {podest} -- ./gateway expose --serviceId iperfIndia')

    #Connect service
    print("\n\nStart connection")
    runcmd(f'kubectl config use-context kind-gw-host')
    runcmd(f'kubectl exec -i {podhost} -- ./gateway connect --serviceId iperfIsrael  --serviceIdDest iperfIndia &')
    time.sleep(30)
    print("\n\nStart Iperf3 testing")
    podIperf3= getPodName("iperf3-clients")
    runcmd(f'kubectl exec -i {podIperf3} --  iperf3 -c gateway-iperf3-service -p 5001')
    





