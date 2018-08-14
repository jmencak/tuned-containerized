FROM centos:7

### Setup user for build execution and application runtime
ENV APP_ROOT=/var/lib/tuned
ENV PATH=${APP_ROOT}/bin:${PATH} HOME=${APP_ROOT}
WORKDIR ${APP_ROOT}
COPY app-root/ ${APP_ROOT}

### Install required packages
RUN INSTALL_PKGS="tuned kernel-tools PyYAML git patch" && \
    yum -y --setopt=tsflags=nodocs install -y ${INSTALL_PKGS} && \
    rpm -V ${INSTALL_PKGS} && \
    curl -sL -o ~/bin/gimme https://raw.githubusercontent.com/travis-ci/gimme/master/gimme && \
    chmod +x ~/bin/gimme && \
    gimme 1.10.3 && \
    mkdir -p ~/bin && \
    export GOROOT="/var/lib/tuned/.gimme/versions/go1.10.3.linux.amd64" && \
    PATH="${GOROOT}/bin:${PATH}" && \
    go get k8s.io/apimachinery/pkg/api/errors \
           k8s.io/client-go/kubernetes \
           github.com/fsnotify/fsnotify && \
    GOOS=linux go build -o ./bin/tuned-wait src/tuned-wait.go && \
    patch -p1 -d /usr/lib/python*/site-packages/tuned/daemon/ < src/tuned.diff && \ 
    sed -i 's/^\s*daemon.*$/daemon = 0/' /etc/tuned/tuned-main.conf && \
    rm -rf ~/.cache ~/.gimme ~/go ~/src bin/gimme /tmp/* && \
    touch /etc/sysctl.conf && \
    yum -y remove git patch && \
    yum clean all && \
    rm -rf /var/cache/yum

ENTRYPOINT [ "/var/lib/tuned/run" ]
