# syntax = docker/dockerfile:1-experimental

FROM golang:1.18-bullseye

WORKDIR /gist

RUN apt-get update -y && apt-get install git ssh curl libz-dev openssl libssl-dev -y

RUN curl https://cdn.openbsd.org/pub/OpenBSD/OpenSSH/portable/openssh-9.0p1.tar.gz -o openssh-9.0.tar.gz && \
    tar -xvf openssh-9.0.tar.gz && \
    cd  openssh-9.0p1 && \
    ./configure && \
    make && \
    make install

COPY go.* .

RUN go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o ./app ./cmd/ultragist && \
    mv ./app /usr/local/bin/ultragist

RUN --mount=type=cache,target=/root/.cache/go-build \
GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o ./app ./cmd/gist-shell && \
    mv ./app /usr/local/bin/gist-shell


RUN useradd -p "*" --home=/data/gists/ -s /usr/local/bin/gist-shell git


RUN mv ./sshd_config /etc/ssh/sshd_config && \
    mkdir -p /run/sshd


CMD [ "/usr/sbin/sshd", "-D", "-e" ]