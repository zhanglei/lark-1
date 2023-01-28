#!/usr/bin/env bash
# 初始化执行一次

docker-compose -f docker-compose-elk.yaml up -d
docker-compose -f docker-compose-flink.yaml up -d
docker-compose -f docker-compose-lark.yaml up -d