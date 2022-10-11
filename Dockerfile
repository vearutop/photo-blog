FROM alpine

COPY ./bin/photo-blog /bin/photo-blog

EXPOSE 80

ENTRYPOINT [ "/bin/photo-blog" ]
