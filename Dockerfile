FROM alpine:3.8

LABEL description="A whitelist of Erlang and Elixir packages available at https://hex.pm. \
The image build downloads all versions of the packages and their dependencies. \
A simple webserver exposes the endpoints needed for getting the packages via Mix."

#RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

RUN apk update \
    && apk add libc6-compat

RUN mkdir -p /hexdump/packages && \
    mkdir -p /hexdump/tarballs && \
    mkdir -p /hexdump/installs && \
    mkdir -p /app

ADD hexdump/hexdump /app/
ADD hexserver/hexserver /app/
ADD packages.txt /app/

RUN /app/hexdump
RUN rm /app/hexdump

ADD plugins/ /plugins/

EXPOSE 5000

CMD ["/app/hexserver"]
