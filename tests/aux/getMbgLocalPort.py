import argparse,json
import subprocess as sp


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Rturn the local port of the service inside the MBG')
    parser.add_argument('-s','--service', help='Service local port inside the MBG', required=True, default="")
    parser.add_argument('-m','--mbg', help='MBG pod name', required=True, default="")
    args = vars(parser.parse_args())

    mbgJson=json.loads(sp.getoutput(f'kubectl exec -i {args["mbg"]} -- cat ./root/.mbgApp'))
    localPort =(mbgJson["Connections"][args["service"]]["Local"]).split(":")[1]
    externalPort =(mbgJson["Connections"][args["service"]]["External"]).split(":")[1]
    
    print(localPort)