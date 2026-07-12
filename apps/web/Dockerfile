FROM node:22-alpine AS build
WORKDIR /app

COPY package*.json ./
COPY apps/web/package.json apps/web/package.json
COPY packages packages
RUN npm install

COPY apps/web apps/web
RUN npm run build -w @cashier/web

EXPOSE 3001
CMD ["npm", "run", "start", "-w", "@cashier/web"]
