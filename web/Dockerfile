# 前端: node 编译 → nginx 静态服务
FROM node:20-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .

# Vite 构建时把 API base 设为相对路径,通过 Nginx 反代
ARG VITE_API_BASE=/api/v1
ENV VITE_API_BASE=${VITE_API_BASE}

RUN npm run build

# ── 运行时: nginx 服务 dist/ ──
FROM nginx:1.27-alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]