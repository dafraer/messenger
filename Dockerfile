# stage 1: build
from golang:1.26.1-alpine as builder
workdir /app

#copy and download dependencies first
copy go.mod go.sum ./
run go mod download

#copy the rest
copy . .

run go build -o messenger ./cmd

# stage 2: run
fROM alpine:3.21
WORKDIR /app
COPY --from=builder /app/messenger .
COPY --from=builder /app/frontend .
CMD ["sh", "-c", "./messenger $SIGNING_KEY :8080 $MONGO_URI"]
