# Contenedor propio de la prueba de extremo a extremo. La version de la imagen debe
# coincidir con la de @playwright/test en package.json. No forma parte de la aplicacion.
FROM mcr.microsoft.com/playwright:v1.56.1-noble
WORKDIR /e2e
COPY package.json ./
RUN npm install --no-audit --no-fund
COPY playwright.config.ts ./
COPY tests ./tests
CMD ["npx", "playwright", "test"]
