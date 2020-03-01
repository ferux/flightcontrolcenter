FROM golang:1.13-alpine

LABEL version="0.1"
LABEL author="Aleksandr Trushkin <atrushkin@outlook.com>"

# Install requires first
RUN apk add --no-cache make git ca-certificates curl protobuf protobuf-dev unzip

# ENV PROTOZIP=protoc-3.10.1-linux-x86_64.zip
# WORKDIR /tmp/protobuf
# RUN curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v3.10.1/${PROTOZIP}
# RUN unzip -o ${PROTOZIP} -d /usr/local 'include/*'
# RUN rm -f ${PROTOZIP}
