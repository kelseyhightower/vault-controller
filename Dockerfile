FROM scratch
MAINTAINER Kelsey Hightower <kelsey.hightower@gmail.com>
ADD vault-controller /vault-controller
ENTRYPOINT ["/vault-controller"]
