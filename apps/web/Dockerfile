FROM node:22-alpine AS build
WORKDIR /app
COPY package*.json ./
COPY apps/web/package.json apps/web/package.json
COPY packages packages
RUN npm install
COPY apps/web apps/web
ARG VITE_API_URL=/api
RUN npm run build -w @cashier/web
FROM nginx:1.27-alpine
COPY --from=build /app/apps/web/dist /usr/share/nginx/html
COPY apps/web/nginx.conf /etc/nginx/conf.d/default.conf
