FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /my-go-app ./main

FROM alpine:latest


WORKDIR /root/
ENV DOCKER_ENV="Web App Dev"

COPY --from=builder /my-go-app .

EXPOSE 8080

CMD ["./my-go-app"]
