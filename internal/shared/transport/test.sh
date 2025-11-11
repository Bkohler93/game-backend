docker compose up -d redis && \
( go test -v -run Ops -timeout 10s ./... || true )&& \
docker compose down