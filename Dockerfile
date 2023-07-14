FROM golang:1.20 as buildstage

# ca-certificates: for downloading go libraries from the internet
# build-essential: for building our project (make)
# curl: for downloading third party tools
RUN apt-get update && \
    apt-get install --yes --no-install-recommends ca-certificates build-essential curl && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o ./bin/external-dns-stackit-webhook -v cmd/webhook/main.go

FROM gcr.io/distroless/static-debian11

COPY --from=buildstage /build/bin/external-dns-stackit-webhook /external-dns-stackit-webhook

ENTRYPOINT ["/external-dns-stackit-webhook"]
