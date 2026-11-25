# syntax=docker/dockerfile:1

# ---- Build stage: typecheck and bundle static assets ----
FROM node:22-alpine AS build
WORKDIR /src

# Install dependencies first so this layer is cached between source edits.
COPY package.json package-lock.json ./
RUN npm ci

ARG VITE_API_URL=http://localhost:8080/api
ENV VITE_API_URL=${VITE_API_URL}

COPY . .
RUN npm run build

# ---- Runtime stage: serve SPA with security headers as non-root ----
FROM node:22-alpine
WORKDIR /app

RUN npm install -g serve@14.2.8 && npm cache clean --force

COPY --from=build --chown=node:node /src/dist ./dist
COPY --chown=node:node serve.json ./serve.json

USER node
EXPOSE 3000
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=3 \
  CMD ["node", "-e", "fetch('http://127.0.0.1:3000/').then(r => process.exit(r.ok ? 0 : 1)).catch(() => process.exit(1))"]
CMD ["serve", "-s", "dist", "-l", "tcp://0.0.0.0:3000", "-c", "/app/serve.json", "--no-clipboard"]
