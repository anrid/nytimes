#!/bin/bash

docker run -d --name my-es7 -p 9200:9200 -p 9300:9300 \
  -e "discovery.type=single-node" \
  -e "xpack.security.enabled=false" \
  -e "xpack.security.http.ssl.enabled=false" \
  -e "bootstrap.memory_lock=true" --ulimit memlock=-1:-1 \
  elasticsearch:7.17.9
