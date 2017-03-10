FROM alpine:3.5

LABEL description="A complete dump of the Erlang and Elixir packages \
available at https://hex.pm. A simple webserver exposes the endpoints \
needed for getting the packages via Mix."

RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

RUN mkdir -p /hexdump/packages && \
    mkdir -p /hexdump/tarballs && \
    mkdir -p /hexdump/installs && \
    mkdir -p /app

ADD hexdump/hexdump /app/
ADD hexserver/hexserver /app/

RUN /app/hexdump
RUN rm /app/hexdump

EXPOSE 5000

CMD ["/app/hexserver"]
