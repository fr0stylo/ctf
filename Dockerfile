FROM golang:alpine as builder


WORKDIR /app

RUN apk update && apk upgrade && apk add --no-cache ca-certificates make
RUN update-ca-certificates

COPY . .

RUN make build

FROM scratch

COPY --from=builder /app/service .
COPY --from=builder /app/foo.db .
COPY --from=builder /app/views views
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/


EXPOSE 1323
CMD [ "./service" ]
