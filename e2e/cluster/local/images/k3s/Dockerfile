# Install the all-in-one binary so we can copy our run-time images into the image
# which helps avoid pulling them when running e2e tests.
ARG BCI="registry.suse.com/bci/bci-base:15.6"
FROM ${BCI} AS k3s
ARG ARCH
ARG K3S_VERSION="v1.32.5+k3s1"
RUN set -x \
   && zypper -n in \
   ca-certificates \
   curl \
   && if [ "${ARCH?required}" != "amd64" ]; then \
   K3S_SUFFIX="-${ARCH}"; \
   fi \
   && curl -fsSL "https://github.com/k3s-io/k3s/releases/download/${K3S_VERSION}/k3s${K3S_SUFFIX}" > /bin/k3s \
   && chmod +x /bin/k3s \
   && ln -s /bin/k3s /bin/kubectl \
   && ln -s /bin/k3s /bin/ctr \
   && k3s --version

COPY scratch/*-${ARCH}.tar /var/lib/rancher/k3s/agent/images/

VOLUME /var/lib/cni
VOLUME /var/lib/kubelet
VOLUME /var/lib/rancher/k3s
VOLUME /var/log

ENTRYPOINT ["k3s"]
CMD ["server"]
