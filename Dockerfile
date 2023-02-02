FROM ubuntu:focal

# tencent client requires curl to run
RUN apt-get update -y && apt-get install -y ca-certificates
RUN useradd -ms /bin/bash controller

COPY --from=BUILD /root/tke-auth-controller /home/controller/tke-auth-controller

# requires when running docker build on windows(WSL) machine.
RUN chmod 777 /home/controller/tke-auth-controller && chown controller:controller /home/controller/tke-auth-controller

# prevent executable grant previlege
USER controller

ENTRYPOINT ["/home/controller/tke-auth-controller"]
