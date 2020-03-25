FROM alpine:3.8

RUN apk upgrade --update --no-cache

RUN addgroup -S helmet && adduser -S helmet -G helmet

USER helmet

ADD _output/bin/helmet-tests /usr/local/bin/helmet-tests

WORKDIR /home/helmet

ENTRYPOINT ["helmet-tests"]