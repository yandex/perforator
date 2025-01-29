# Generate the private key
openssl genrsa -out private.key 2048

# Generate a certificate signing request (CSR)
openssl req -new -key private.key -out server.csr -subj "/CN=minio"

# Generate a self-signed certificate
openssl x509 -req -days 36500 -in server.csr -signkey private.key -out public.crt
