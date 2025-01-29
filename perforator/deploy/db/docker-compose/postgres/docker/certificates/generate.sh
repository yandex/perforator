openssl genrsa -out ca.key 2048

openssl req -x509 -subj "/CN=postgres" -nodes -key ca.key -days 36500 -out ca.crt

openssl req -newkey rsa:2048 -nodes -subj "/CN=postgres" -addext "subjectAltName = DNS:postgres" -keyout server.key -out server.csr

openssl x509 -req -in server.csr -out server.crt -CAcreateserial -CA ca.crt -CAkey ca.key -days 36500
