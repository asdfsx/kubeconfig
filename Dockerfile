FROM alpine
ARG binname
ENV binname=${binname:-kubeconfig}
RUN apk --no-cache add ca-certificates
WORKDIR /root/
RUN echo $binname
ADD ${binname} .
CMD ["./kubeconfig", "-incluster=true"]