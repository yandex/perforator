openssl genrsa -out ca.key 2048

openssl req -x509 -subj "/CN=clickhouse" -nodes -key ca.key -days 36500 -out ca.crt

openssl req -newkey rsa:2048 -nodes -subj "/CN=clickhouse" -addext "subjectAltName = DNS:clickhouse" -keyout server.key -out server.csr

openssl x509 -req -in server.csr -out server.crt -CAcreateserial -CA ca.crt -CAkey ca.key -days 36500

openssl dhparam -out dhparam.pem 2048
