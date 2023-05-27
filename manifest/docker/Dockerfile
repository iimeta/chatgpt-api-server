FROM loads/alpine:3.8

###############################################################################
#                                INSTALLATION
###############################################################################

ENV WORKDIR                 /app
ADD resource                $WORKDIR/
COPY frontend/dist          $WORKDIR/public/xyhelper
ADD ./temp/linux_amd64/main $WORKDIR/main
RUN chmod +x $WORKDIR/main

###############################################################################
#                                   START
###############################################################################
WORKDIR $WORKDIR
CMD ./main
