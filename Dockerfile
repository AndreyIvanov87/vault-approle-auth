FROM golang:1.17.1-alpine3.14 AS builder
RUN adduser -u 10001 -D -H app

FROM scratch
WORKDIR /app

COPY --from=builder /etc/passwd /etc/passwd
USER app

COPY $CI_PROJECT_DIR/.bin/* /app/
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

CMD ["/app/vaultauth"]