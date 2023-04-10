FROM golang:1.20 as builder

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o controller .

FROM scratch
WORKDIR /
COPY --from=builder /app/controller /controller

ENTRYPOINT ["./controller"]

