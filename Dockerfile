FROM gcr.io/distroless/static-debian11:nonroot

COPY external-dns-stackit-webhook /external-dns-stackit-webhook

ENTRYPOINT ["/external-dns-stackit-webhook"]
