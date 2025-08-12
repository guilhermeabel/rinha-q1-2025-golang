FROM golang:1.24-alpine AS builder

RUN apk add --update \
	make \
	git \
	librdkafka \
	librdkafka-dev \
	build-base \
	coreutils \
	gcc \
	make \
	musl-dev \
	ca-certificates

WORKDIR /usr/src/app

COPY go.mod go.sum ./

RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ./bin/service ./cmd/payment-processor

FROM public.ecr.aws/docker/library/alpine:3.20 AS final

ENV TZ="America/Sao_Paulo"

COPY --from=builder /etc/ca-certificates.conf /etc/ca-certificates.conf
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/ca-certificates /usr/share/ca-certificates

COPY --from=builder /usr/src/app/bin/service /bin/service

RUN apk --no-cache add tzdata

RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

CMD ["./bin/service"]
