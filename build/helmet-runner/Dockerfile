FROM alpine:3.8

RUN apk upgrade --update --no-cache

RUN addgroup -S helmet && adduser -S helmet -G helmet

USER helmet

ADD _output/bin/helmet-runner /usr/local/bin/helmet-runner

WORKDIR /home/helmet

ENTRYPOINT ["helmet-runner"]