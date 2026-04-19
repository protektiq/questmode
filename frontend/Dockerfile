# syntax=docker/dockerfile:1
FROM cgr.dev/chainguard/node:latest AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM cgr.dev/chainguard/nginx:latest
COPY --from=builder /app/dist /usr/share/nginx/html
COPY default.conf /etc/nginx/conf.d/default.conf
