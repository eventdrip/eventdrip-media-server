FROM golang:1.13.10-buster

WORKDIR /usr/src/eventdrip

EXPOSE 8001
EXPOSE 1935
EXPOSE 7935

RUN apt-get update
RUN apt-get install -y autoconf gnutls-dev

RUN go get github.com/golang/glog
RUN go get github.com/google/uuid
RUN go get -d github.com/livepeer/lpms/core
RUN go get -d github.com/livepeer/lpms/stream
RUN go get -d github.com/livepeer/lpms/segmenter
RUN go get -d github.com/livepeer/m3u8

RUN useradd -m aptly
USER aptly
RUN echo $HOME

COPY . .

RUN bash ./scripts/install_ffmpeg.sh

CMD [ "bash", "./scripts/run.sh" ]

