FROM amazonlinux

RUN yum update -y

# install GO

RUN yum install -y wget tar gzip git
RUN wget -q https://storage.googleapis.com/golang/go1.13.1.linux-amd64.tar.gz
ENV GOROOT="/usr/local/go"
RUN tar -xzf go1.13.1.linux-amd64.tar.gz -C /usr/local/
RUN rm go1.13.1.linux-amd64.tar.gz
ENV GOBIN="${GOROOT}/bin"
ENV GOPATH="/go/"
RUN mkdir -p $GOPATH
ENV PATH="${GOPATH}bin:${GOBIN}:${PATH}"

RUN go version

# install optimizers

RUN yum install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm
RUN yum install -y jpegoptim optipng pngquant gifsicle libwebp-tools

# install nodeJS (for svgo)
RUN curl -sL https://rpm.nodesource.com/setup_10.x | bash -
RUN yum install -y nodejs
RUN node --version
RUN npm install -g svgo

# check for optimizers installation

RUN jpegoptim --help
RUN optipng --version
RUN pngquant --version
RUN gifsicle --version
RUN cwebp -version
RUN svgo -v

# install ffmpeg

RUN yum install -y xz
WORKDIR /usr/local/bin/ffmpeg
RUN wget -q https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz
COPY ffmpeg-release-amd64-static.tar.xz.md5 .
RUN md5sum -c ffmpeg-release-amd64-static.tar.xz.md5
RUN tar -xf ffmpeg-release-amd64-static.tar.xz
RUN rm ffmpeg-release-amd64-static.tar.xz
RUN cd ffmpeg-4.2.2-amd64-static
WORKDIR /usr/local/bin/ffmpeg/ffmpeg-4.2.2-amd64-static
RUN cp -a /usr/local/bin/ffmpeg/ffmpeg-4.2.2-amd64-static/ffmpeg /usr/local/bin/ffmpeg/
RUN ln -s /usr/local/bin/ffmpeg/ffmpeg /usr/bin/ffmpeg
RUN ffmpeg -version

# install media-optimizer program

RUN yum install -y make
WORKDIR $GOPATH/src/github.com/bertrandmartel/media-optimizer
COPY . .
RUN make install
RUN make build

ENTRYPOINT ["./media-optimizer"]