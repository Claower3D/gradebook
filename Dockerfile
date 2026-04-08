FROM golang:1.22-bullseye

WORKDIR /app
COPY . .

RUN apt-get update && apt-get install -y gcc
RUN cd cmd/server && CGO_ENABLED=1 go build -o /app/gradebook .

EXPOSE 8080
ENV PORT=8080

CMD ["/app/gradebook"]