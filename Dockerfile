# as we are not having any free tier http/2 deployments we are pushing the py codebase to the dockerhub
# and from there were are fetching and internally we are using grpc for communication to avoid the
# http/2 deployment

FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server cmd/parkinsons/main.go

FROM curiousyeswanth/parkinsons-py:latest

WORKDIR /app

RUN apt-get update && apt-get install -y supervisor && rm -rf /var/lib/apt/lists/*

# Copy go binary
COPY --from=builder /app/server ./go-server
RUN chmod +x ./go-server

# Supervisor config
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

EXPOSE 8080
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]