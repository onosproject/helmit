FROM alpine:3.8

RUN apk upgrade --update --no-cache

RUN addgroup -S helmit && adduser -S helmit -G helmit

USER helmit

ADD _output/bin/helmit-tests /usr/local/bin/helmit-tests

WORKDIR /home/helmit

ENTRYPOINT ["helmit-tests"]