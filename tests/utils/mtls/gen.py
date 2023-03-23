
import os
import tempfile
import tempfile
# Example: For using Ip address 
# kindPodAddress=",".join(f"IP:10.244.0.{x}" for x in range(2,30))
# kindAddress=",".join(f"IP:172.18.0.{x}" for x in range(2,10))
# address=kindAddress +","+ kindPodAddress+ 

mbgctlAddress=",".join(f"DNS:mbgctl{x}" for x in range(0,10))
mbgAddress=",".join(f"DNS:mbg{x}" for x in range(0,10))
address=mbgctlAddress +","+ mbgAddress+","+"DNS:localhost"
subject_alt_name = f"subjectAltName={address}"
print(subject_alt_name)
with tempfile.NamedTemporaryFile(mode="w") as temp:
    temp.write(subject_alt_name)
    temp.flush()

    # Generate self signed root CA cert
    os.system(f'openssl req -nodes -x509 -days 358000 -newkey rsa:2048 -keyout ca.key -out ca.crt -subj "/CN=IL" -addext "subjectAltName={address}"')

    # Generate mbg1 cert to be signed
    os.system(f'openssl req -nodes -newkey rsa:2048 -keyout mbg1.key -out mbg1.csr -subj "/CN=IL" -addext "subjectAltName={address}"')
    # Sign the mbg1 cert
    os.system(f'openssl x509 -req  -days 358000 -in mbg1.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out mbg1.crt -extfile {temp.name}')

    # Generate mbg2 cert to be signed
    os.system(f'openssl req -nodes -newkey rsa:2048 -keyout mbg2.key -out mbg2.csr -subj "/CN=IL" -addext "subjectAltName={address}"')
    # Sign the mbg2 cert
    os.system(f'openssl x509 -req  -days 358000 -in mbg2.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out mbg2.crt -extfile {temp.name} ')

    # Generate mbg3 cert to be signed
    os.system(f'openssl req -nodes -newkey rsa:2048 -keyout mbg3.key -out mbg3.csr -subj "/CN=IL" -addext "subjectAltName={address}"')
    # Sign the mbg3 cert
    os.system(f'openssl x509 -req  -days 358000 -in mbg3.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out mbg3.crt -extfile {temp.name} ')
