FROM golang:latest

WORKDIR /app

COPY . .

RUN go mod download

RUN go build -o task ./cmd

EXPOSE 80

CMD ["sh", "-c", "./task ${SIGNING_KEY} :80 ${MONGO_URI}"]