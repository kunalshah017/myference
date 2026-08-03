FROM node:22.21.1-bookworm-slim

ARG CODEX_VERSION=0.146.0

RUN npm install --global "@openai/codex@${CODEX_VERSION}" \
    && npm cache clean --force \
    && useradd --create-home --uid 10001 --shell /usr/sbin/nologin myference

USER 10001:10001
WORKDIR /workspace
ENV HOME=/workspace \
    NO_COLOR=1 \
    TERM=dumb

ENTRYPOINT ["codex"]
