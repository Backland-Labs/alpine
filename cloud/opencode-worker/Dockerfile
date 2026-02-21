FROM node:20-alpine

RUN apk add --no-cache git bash curl jq

RUN addgroup -S opencode && adduser -S opencode -G opencode

WORKDIR /app

RUN npm install -g opencode

RUN mkdir -p /home/opencode/.local/share/opencode && \
    mkdir -p /home/opencode/.config/opencode && \
    chown -R opencode:opencode /home/opencode/.local && \
    chown -R opencode:opencode /home/opencode/.config

COPY --chown=opencode:opencode entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

USER opencode

ENV NODE_ENV=production
ENV OPENCODE_PORT=3000

EXPOSE 3000

ENTRYPOINT ["/app/entrypoint.sh"]
