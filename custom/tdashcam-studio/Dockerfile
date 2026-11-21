FROM dupengcheng66666/tdashcam-studio:1.1.23@sha256:ebe5e8182bcb1c9ccc5f555a09751b2c755a22816502df912d5ff7df4a3fb23d

COPY default.conf /etc/nginx/conf.d/default.conf
COPY configure-server-mode.js /usr/share/nginx/configure-server-mode.js
COPY server-mode.js /usr/share/nginx/html/server-mode.js

RUN cd /usr/share/nginx/html && \
    printf 'window.TDASHCAM_SERVER = null;\n' > server-config.js && \
    sed -i 's|<script src="script.js"></script>|<script src="server-config.js"></script><script src="script.js"></script><script src="server-mode.js"></script>|' index.html && \
    grep -q 'server-mode.js' index.html && \
    printf '#!/bin/sh\nnode /usr/share/nginx/configure-server-mode.js\n' > /docker-entrypoint.d/98-configure-server-mode.sh && \
    chmod +x /docker-entrypoint.d/98-configure-server-mode.sh
