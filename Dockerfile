FROM alpine

COPY ./bin/brick-starter-kit /bin/brick-starter-kit

EXPOSE 80

ENTRYPOINT [ "/bin/brick-starter-kit" ]
