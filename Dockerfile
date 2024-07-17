FROM golang:alpine AS builder
RUN apk add --no-cache build-base
RUN apk --no-cache add ca-certificates
WORKDIR /build
COPY . .
RUN go build -ldflags="-s -w" -trimpath -o /dist/rssy .
RUN ldd /dist/rssy | tr -s [:blank:] '\n' | grep ^/ | xargs -I % install -D % /dist/%
RUN ln -s ld-musl-x86_64.so.1 /dist/lib/libc.musl-x86_64.so.1

FROM scratch
COPY --from=builder /dist /
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

CMD ["/rssy"]
