FROM docker.io/debian:bullseye-slim

# tencent client requires curl to run
RUN apt-get update -y && apt-get install -y ca-certificates
RUN useradd -ms /bin/bash controller

COPY --chown=controller:controller tke-auth-controller /home/controller/tke-auth-controller

# prevent executable grant previlege
USER controller

ENTRYPOINT ["/home/controller/tke-auth-controller"]
