docker compose up -d redis && \
( go test -v -run StreamListener -timeout 10s ./... || true )&& \
docker compose down