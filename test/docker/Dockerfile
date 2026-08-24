# Throwaway Ubuntu image to exercise install/uninstall end to end.
FROM ubuntu:24.04
RUN apt-get update && apt-get install -y --no-install-recommends zsh git ca-certificates \
 && rm -rf /var/lib/apt/lists/* \
 && useradd -m -s /bin/zsh tester
COPY --chown=tester:tester . /home/tester/dotfiles
ARG TARGETARCH=amd64
COPY --chown=tester:tester dist/dotfiles-linux-${TARGETARCH:-amd64} /usr/local/bin/dotfiles
USER tester
WORKDIR /home/tester/dotfiles
ENV HOME=/home/tester DOTFILES=/home/tester/dotfiles
ENTRYPOINT ["/bin/bash", "test/docker/e2e.sh"]
