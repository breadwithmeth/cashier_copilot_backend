FROM node:22-alpine AS build
WORKDIR /app
COPY package*.json ./
COPY apps/api/package.json apps/api/package.json
COPY apps/web/package.json apps/web/package.json
COPY packages packages
RUN npm install
COPY apps/api apps/api
RUN npm run generate -w @cashier/database && npm run build -w @cashier/api
CMD ["node","apps/api/dist/main.js"]
