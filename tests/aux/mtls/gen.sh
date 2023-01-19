#! /bin/bash
# IP:10.244.0.4,10.244.0.2 -mbg and mbgctl inside the same kind cluster, IP:172.18.0.2,172.18.0.3 - mbgctl is outside the MBG cluster
kindaddress="IP:10.244.0.2,IP:10.244.0.3,IP:10.244.0.4,IP:10.244.0.5,IP:10.244.0.6,IP:10.244.0.7,IP:10.244.0.8"
# Generate self signed root CA cert
openssl req -nodes -x509 -days 358000 -newkey rsa:2048 -keyout ca.key -out ca.crt -subj "/CN=IL" -addext "subjectAltName=IP:172.18.0.3,$kindaddress"

# Generate mbg1 cert to be signed
openssl req -nodes -newkey rsa:2048 -keyout mbg1.key -out mbg1.csr -subj "/CN=IL" -addext "subjectAltName=IP:172.18.0.2,$kindaddress"
# Sign the mbg1 cert
openssl x509 -req -in mbg1.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out mbg1.crt -extfile <(printf "subjectAltName=IP:172.18.0.2,$kindaddress")

# Generate mbg2 cert to be signed
openssl req -nodes -newkey rsa:2048 -keyout mbg2.key -out mbg2.csr -subj "/CN=IL" -addext "subjectAltName=IP:172.18.0.3,$kindaddress"
# Sign the mbg1 cert
openssl x509 -req -in mbg2.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out mbg2.crt -extfile <(printf "subjectAltName=IP:172.18.0.3,$kindaddress")
 