FROM golang:latest

WORKDIR /app

COPY . .

RUN go mod download

RUN go build -o task ./cmd


CMD ["sh", "-c", "./task $SIGNING_KEY :8080 $MONGO_URI"]