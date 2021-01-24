FROM alpine:latest as build
RUN apk update &&\
    apk upgrade &&\ 
    apk add --no-cache linux-headers alpine-sdk cmake tcl openssl-dev zlib-dev yasm x264-dev x265-dev go libsrt-dev
WORKDIR /tmp
RUN git clone https://github.com/voc/srtrelay.git
RUN git clone https://github.com/FFmpeg/FFmpeg.git
RUN mkdir /opt/ffmpeg
WORKDIR /tmp/srtrelay
RUN go build
RUN cp config.toml.example config.toml
WORKDIR /tmp/FFmpeg
RUN ./configure --enable-libsrt --enable-libx264 --enable-gpl --enable-libx265 --enable-ffplay --pkg-config-flags="--static" --enable-static --prefix=/opt/ffmpeg && make && make install

FROM alpine:latest
ENV LD_LIBRARY_PATH /lib:/usr/lib:/usr/local/lib64
RUN apk update &&\
    apk upgrade &&\
    apk add --no-cache libstdc++ x264-libs x265-libs libsrt &&\
    mkdir -p /conf /logs /record /opt/ffmpeg
COPY --from=build /tmp/srtrelay/srtrelay /usr/local/bin/
COPY --from=build /tmp/srtrelay/config.toml /conf/
COPY --from=build /opt/ffmpeg/ /opt/ffmpeg/
ENV PATH "$PATH:/opt/ffmpeg/bin"
EXPOSE 8080
EXPOSE 1935/udp
ENTRYPOINT [ "srtrelay","-config","/conf/config.toml"]
