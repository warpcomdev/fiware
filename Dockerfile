FROM golang:1.21 as builder
# -------------------------

WORKDIR /src
COPY . .

env CGO_ENABLED=0
RUN go mod download
RUN go generate ./...

WORKDIR /src/cmd/fiware
RUN go build -o /bin/app

FROM scratch
# ----------

COPY --from=builder /bin/app /bin/app

WORKDIR /tmp
ENTRYPOINT ["/bin/app"]
