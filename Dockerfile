FROM alpine
RUN apk --no-cache add ca-certificates
WORKDIR /root/
ADD kubeconfig .
ADD swagger-ui-dist swagger-ui-dist
CMD ["./kubeconfig", "-incluster=true", "-swagger-ui-dist=/root/swagger-ui-dist"]