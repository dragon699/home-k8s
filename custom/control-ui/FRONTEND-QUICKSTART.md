# Quick Start Guide - Downloader Connector Frontend

This guide will help you get the frontend up and running quickly to connect to your Downloader API.

## What You Got

A complete React frontend with:
- ✅ Action Center UI (matching your screenshot design)
- ✅ Infrastructure Actions section
- ✅ Application monitoring with torrent view
- ✅ Docker support for production deployment
- ✅ Development environment with hot-reload

## Folder Structure

```
/home/martin/stuff/home-k8s/fetch-api/connectors/downloader/
├── frontend/                    # Your new frontend folder
│   ├── src/                    # Source code
│   │   ├── components/         # React components
│   │   └── services/           # API integration
│   ├── Dockerfile              # Production container
│   ├── nginx.conf              # Nginx configuration
│   └── package.json            # Dependencies
├── docker-compose.frontend.yaml # Runs both API + Frontend
├── start-frontend.sh           # Quick dev start script
└── start-frontend-docker.sh    # Quick prod start script
```

## Option 1: Development Mode (Recommended for Testing)

**Start the frontend in development mode with hot-reload:**

```bash
cd /home/martin/stuff/home-k8s/fetch-api/connectors/downloader
./start-frontend.sh
```

Or manually:
```bash
cd frontend
npm install
npm run dev
```

**Access:**
- Frontend: http://localhost:3000
- Your API should be running on: http://localhost:8080

**Note:** Make sure your Go API is running first!

## Option 2: Docker Production Mode

**Run both API and frontend as Docker containers:**

```bash
cd /home/martin/stuff/home-k8s/fetch-api/connectors/downloader
./start-frontend-docker.sh
```

Or manually:
```bash
docker-compose -f docker-compose.frontend.yaml up -d
```

**Access:**
- Frontend: http://localhost:3000
- API: http://localhost:8080

**To stop:**
```bash
docker-compose -f docker-compose.frontend.yaml down
```

## Option 3: Docker Frontend Only

**Build and run just the frontend container:**

```bash
cd /home/martin/stuff/home-k8s/fetch-api/connectors/downloader/frontend
docker build -t downloader-frontend .
docker run -p 3000:80 downloader-frontend
```

**Access:** http://localhost:3000

**Important:** Update `nginx.conf` to point to your API location if it's not at `http://downloader-api:8080`

## Connecting to Your API

### Development Mode
The Vite dev server proxies API requests. If your API is not on port 8080, edit `frontend/vite.config.js`:

```javascript
server: {
  proxy: {
    '/api': {
      target: 'http://localhost:YOUR_PORT',  // Change this
      changeOrigin: true,
    }
  }
}
```

### Production Mode (Docker)
Edit `frontend/nginx.conf` and update the proxy_pass lines:

```nginx
location /api/ {
    proxy_pass http://YOUR_API_HOST:YOUR_PORT/api/;
    ...
}
```

## Testing the Setup

1. **Start your Go API first**
2. **Start the frontend** using one of the options above
3. **Open browser** to http://localhost:3000
4. **Check the tabs:**
   - **Infrastructure**: Should show the action cards
   - **Applications**: Should fetch and display torrents from your API

## Customization

### Change Colors
Edit `frontend/tailwind.config.js` to modify the purple theme

### Add New API Endpoints
1. Add functions to `frontend/src/services/api.js`
2. Use them in your components

### Modify UI
Edit components in `frontend/src/components/`

## Troubleshooting

**Frontend can't connect to API:**
- Check API is running: `curl http://localhost:8080/api/health`
- Check CORS is enabled in your Go API (it should be based on the code)
- Check proxy configuration matches your API port

**npm install fails:**
- Make sure you have Node.js 18+ installed: `node --version`
- Try `npm cache clean --force` then `npm install` again

**Docker build fails:**
- Make sure Docker is running: `docker ps`
- Check you have enough disk space: `df -h`

## Next Steps

- Deploy to Kubernetes using manifests in `/manifests`
- Create an Ingress for external access
- Add authentication/authorization
- Connect to your actual Kubernetes cluster for real infrastructure actions

## Support

For issues, check:
- Frontend logs: Browser DevTools Console
- API logs: Terminal where API is running
- Docker logs: `docker-compose -f docker-compose.frontend.yaml logs -f`

---

Created: February 2026
