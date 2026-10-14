# syntax=docker/dockerfile:1
FROM sstefanov/actual-mcp:v1.14.0@sha256:36ae0add0f7419ad29f2a4778368b774658648586c2b7f74b47d1c6c8bf2a305
RUN npm install --omit=dev --no-audit --no-fund @actual-app/api@26.10.0 && npm cache clean --force
RUN sed -i "s/new Intl.NumberFormat('en-US', {/new Intl.NumberFormat(process.env.ACTUAL_LOCALE || 'en-US', {/; s/currency: 'USD',/currency: process.env.ACTUAL_CURRENCY || 'USD',/" build/utils.js \
 && grep -q "process.env.ACTUAL_CURRENCY" build/utils.js && grep -q "process.env.ACTUAL_LOCALE" build/utils.js
