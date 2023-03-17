# SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
#
# SPDX-License-Identifier: Apache-2.0

FROM alpine:3.8

RUN apk upgrade --update --no-cache

RUN addgroup -S helmit && adduser -S helmit -G helmit

USER helmit

ADD _output/bin/helmit-bench /usr/local/bin/helmit-bench

WORKDIR /home/helmit

ENTRYPOINT ["helmit-bench"]
