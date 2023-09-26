# FROM loads/alpine:3.8
FROM ubuntu

###############################################################################
#                                INSTALLATION
###############################################################################

# RUN apk add libc6-compat
# RUN ln -s /lib/libc.musl-x86_64.so.1 /lib/ld-linux-x86-64.so.2
ENV WORKDIR                 /app
ADD resource                $WORKDIR/
COPY config.prod.yaml     $WORKDIR/config.yaml
COPY frontend/dist          $WORKDIR/resource/public/xyhelper
ADD ./temp/linux_amd64/main $WORKDIR/main
RUN chmod +x $WORKDIR/main

###############################################################################
#                                   START
###############################################################################
WORKDIR $WORKDIR
CMD ./main
