# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package descriptors
COPY package.json ./
# Copy package-lock.json if it exists
RUN if [ -f package-lock.json ]; then COPY package-lock.json ./; fi

# Install build dependencies
RUN npm install

# Copy source code
COPY . .

# Build Vite SPA output
RUN npm run build

# Production server stage
FROM nginx:stable-alpine

# Copy static assets from build stage
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy routing configuration for SPA
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
