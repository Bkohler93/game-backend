docker compose up -d redis && \
( go test -v -cover ./*.go  || true )&& \
docker compose down