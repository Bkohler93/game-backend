# docker run -p 6379:6379 -d --name "$1" redis:7.4.5-alpine3.21
docker run -p 6379:6379 -d --name "$1" redis/redis-stack-server:latest 