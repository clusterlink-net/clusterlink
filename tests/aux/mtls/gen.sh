#! /bin/bash

# Generate self signed root CA cert
openssl req -nodes -x509 -days 358000 -newkey rsa:2048 -keyout ca.key -out ca.crt -subj "/CN=IL" -addext "subjectAltName=IP:172.18.0.3,IP:172.18.0.2"

# Generate mbg1 cert to be signed
openssl req -nodes -newkey rsa:2048 -keyout mbg1.key -out mbg1.csr -subj "/CN=IL" -addext "subjectAltName=IP:172.18.0.2"
# Sign the mbg1 cert
openssl x509 -req -in mbg1.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out mbg1.crt -extfile <(printf "subjectAltName=IP:172.18.0.2")

# Generate mbg2 cert to be signed
openssl req -nodes -newkey rsa:2048 -keyout mbg2.key -out mbg2.csr -subj "/CN=IL" -addext "subjectAltName=IP:172.18.0.3"
# Sign the mbg1 cert
openssl x509 -req -in mbg2.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out mbg2.crt -extfile <(printf "subjectAltName=IP:172.18.0.3")
 