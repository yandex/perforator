Из пода, который находится в кубере, нельзя сходить в localhost ноды, для обхода проблемы можно прибегнуть к следующему решению:

при поднятии docker compose файла докер так же создает подсеть, обычно это 172.17. 0.0/16

Полное имя сети можно найти в списке с помощью команды

    docker network ls

Там будет что-то типа:

    NETWORK ID     NAME                        DRIVER    SCOPE
    b6224d314412   docker_perforator_network   bridge    local

Далее команда 

    docker network inspect docker_perforator_network

... Выдает список всех контейнеров для этой сети. Нас интересует

    "IPv4Address": "172.17.0.3/16"

Этот же адрес можно достатьдля каждого контейнера отдельно командой

    docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' postgres 

... где вместо postgres имя соответствующего контейнера

Этот адрес далее вбивается в конфиг с соответствующим портом

При перезапуске пода в докере его адрес меняется, чтобы не пришлось перезагружать все приложение в кубере можно создать Service и привязать к нему Endpoint (при перезапуске пода придется менять endpoint) Пример:

```
apiVersion: v1
kind: Service
metadata:
  name: minio-service
  namespace: perforator
spec:
   ports:
   - protocol: TCP
     port: 9002
---
apiVersion: v1
kind: Endpoints
metadata:
  name: minio-service
  namespace: perforator
subsets:
  - addresses:
    - ip: 172.17.0.3
    ports:
      - port: 9002
```







ссылки:
https://www.freecodecamp.org/news/how-to-get-a-docker-container-ip-address-explained-with-examples/
https://stackoverflow.com/questions/65123401/how-to-access-hosts-localhost-from-inside-kubernetes-cluster