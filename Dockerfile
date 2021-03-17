FROM ros:foxy as communication_link_builder

ARG BUILD_NUMBER
ARG DISTRIBUTION
ARG ARCHITECTURE

# Install build dependencies
RUN apt-get update -y && apt-get install -y --no-install-recommends \
    curl \
    build-essential \
    fakeroot \
    git-core \
    golang \
    libgstreamer1.0-0 \
    libgstreamer-plugins-base1.0-dev \
    libgstreamer-plugins-bad1.0-dev \
    libgstreamer-plugins-good1.0-dev \
    && rm -rf /var/lib/apt/lists/*

# Install px4-msgs
RUN curl -fsSL https://github.com/tiiuae/fog_sw/releases/download/ver_ssrc_rel_2/ros-foxy-px4-msgs_2.0.1-0focal_amd64.deb -o ros-foxy-px4-msgs.deb \
    && dpkg -i ros-foxy-px4-msgs.deb \
    && rm ros-foxy-px4-msgs.deb

WORKDIR /build

COPY . .

RUN cd packaging/ \
    && ./package.sh ${BUILD_NUMBER} ${DISTRIBUTION} ${ARCHITECTURE}

FROM scratch

COPY --from=communication_link_builder /build/*.deb /packages/
