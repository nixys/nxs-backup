FROM golang:1.17

ENV CGO_ENABLED 0

RUN apt-get update && \
		apt-get install -yq \
		apt-utils \
    libev4 \
    default-mysql-client\
    postgresql-client

RUN curl https://downloads.percona.com/downloads/Percona-XtraBackup-LATEST/Percona-XtraBackup-8.0.29-22/binary/debian/focal/x86_64/percona-xtrabackup-80_8.0.29-22-1.focal_amd64.deb -o /tmp/xtrabackup.deb && \
    apt -fyq install /tmp/xtrabackup.deb && \
    rm /tmp/xtrabackup.deb

RUN curl https://fastdl.mongodb.org/tools/db/mongodb-database-tools-ubuntu2004-x86_64-100.5.4.deb -o /tmp/mongotools.deb && \
    apt -fyq install /tmp/mongotools.deb && \
    rm /tmp/mongotools.deb

RUN cd /tmp && \
    wget http://download.redis.io/redis-stable.tar.gz && \
    tar xvzf redis-stable.tar.gz && \
    cd redis-stable && \
    make && \
    cp src/redis-cli /usr/local/bin/ && \
    chmod 755 /usr/local/bin/redis-cli && \
    cd && rm -rf /tmp/redis-stable*

RUN useradd nxs-backup -u 1000 -m
USER nxs-backup