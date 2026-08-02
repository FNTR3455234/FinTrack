# Imagen del frontend. Node compila; nginx sirve.
#
# En la imagen final no hay ni Node ni node_modules: un bundle ya compilado son
# archivos estaticos, y servirlos no necesita nada mas que un servidor web. Se
# pasa de unos 500 MB a unos 55 MB.

# --- Etapa 1: compilar ------------------------------------------------------
FROM node:22-alpine AS compilacion

WORKDIR /origen

# El manifiesto primero, igual que con go.mod en el backend: cambiar un
# componente no debe obligar a reinstalar las dependencias.
#
# npm ci y no npm install: instala exactamente las versiones del
# package-lock.json y falla si el lock no cuadra con el package.json. En una
# compilacion repetible eso es lo que se quiere.
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund

COPY . .
RUN npm run build

# --- Etapa 2: servir --------------------------------------------------------
FROM nginx:1.27-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY seguridad.conf /etc/nginx/seguridad.conf
COPY --from=compilacion /origen/dist /usr/share/nginx/html

# nginx corre como usuario sin privilegios, tambien el proceso maestro.
#
# La imagen oficial ya baja los procesos de trabajo al usuario nginx, pero deja
# el maestro como root para poder abrir el puerto 80. Aqui se escucha en el 8080
# (ver nginx.conf) y asi no hace falta root en ningun momento; a cambio hay que
# darle al usuario los directorios en los que nginx escribe.
RUN touch /var/run/nginx.pid \
    && chown -R nginx:nginx /var/cache/nginx /var/run/nginx.pid /usr/share/nginx/html

USER nginx
EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --quiet --spider http://127.0.0.1:8080/ || exit 1

# daemon off deja a nginx en primer plano: si se demonizara, el proceso
# principal del contenedor terminaria de inmediato y Docker lo daria por
# apagado.
CMD ["nginx", "-g", "daemon off;"]
