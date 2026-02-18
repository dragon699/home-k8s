# Downloader Connector Frontend

A modern React-based frontend for the Downloader Connector API, featuring an Action Center-style interface for managing infrastructure and application resources.

## Features

- **Infrastructure Actions**: Manage Kubernetes resources (deploy, scale, update, restart)
- **Application Monitoring**: View and monitor torrent downloads
- **Modern UI**: Clean, responsive design with Tailwind CSS
- **Real-time Updates**: Automatic refresh and status updates
- **Docker Support**: Fully containerized with nginx for production

## Project Structure

```
frontend/
├── src/
│   ├── components/
│   │   ├── ActionCenter.jsx        # Main container component
│   │   ├── InfrastructureActions.jsx  # Infrastructure management UI
│   │   └── ApplicationActions.jsx     # Application monitoring UI
│   ├── services/
│   │   └── api.js                   # API integration layer
│   ├── App.jsx                      # Root application component
│   ├── main.jsx                     # Application entry point
│   └── index.css                    # Global styles
├── public/                          # Static assets
├── Dockerfile                       # Production container build
├── nginx.conf                       # Nginx configuration
├── docker-compose.frontend.yaml     # Docker compose setup
├── package.json                     # Dependencies and scripts
├── vite.config.js                   # Vite build configuration
└── tailwind.config.js               # Tailwind CSS configuration
```

## Getting Started

### Development Mode

1. **Install dependencies:**
   ```bash
   cd frontend
   npm install
   ```

2. **Set up environment variables:**
   ```bash
   cp .env.example .env
   # Edit .env and set VITE_API_URL if needed
   ```

3. **Start development server:**
   ```bash
   npm run dev
   ```

   The frontend will run on `http://localhost:3000` with hot-reload enabled.

### Production Build

1. **Build the application:**
   ```bash
   npm run build
   ```

2. **Preview production build:**
   ```bash
   npm run preview
   ```

## Docker Deployment

### Build and Run with Docker

1. **Build the frontend image:**
   ```bash
   docker build -t downloader-frontend ./frontend
   ```

2. **Run the container:**
   ```bash
   docker run -p 3000:80 downloader-frontend
   ```

   Access the frontend at `http://localhost:3000`

### Using Docker Compose (Recommended)

Run both the API and frontend together:

```bash
docker-compose -f docker-compose.frontend.yaml up -d
```

This will start:
- API backend on `http://localhost:8080`
- Frontend on `http://localhost:3000`

To stop:
```bash
docker-compose -f docker-compose.frontend.yaml down
```

## API Integration

The frontend connects to the backend API through environment variables:

- **Development**: Uses Vite proxy (configured in `vite.config.js`)
- **Production**: Uses nginx reverse proxy (configured in `nginx.conf`)

### Environment Variables

- `VITE_API_URL`: API URL used for dev proxy and runtime nginx upstream for `/api` and `/torrents` in the production container (required at runtime in production)
- `VITE_JELLYFIN_URL`: Jellyfin link URL used by the card header icon
- `VITE_QBITTORRENT_URL`: qBittorrent link URL used by the card header icon

## Available Scripts

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run preview` - Preview production build locally

## Technologies

- **React 18**: UI framework
- **Vite**: Build tool and dev server
- **Tailwind CSS**: Utility-first CSS framework
- **Nginx**: Production web server
- **Docker**: Containerization

## Configuration

### API Endpoint Configuration

In development, update `vite.config.js`:
```javascript
server: {
  proxy: {
    '/api': {
      target: 'http://your-api-url:8080',
      changeOrigin: true,
    }
  }
}
```

In production, update `nginx.conf`:
```nginx
location /api/ {
    proxy_pass http://your-api-service:8080/api/;
    ...
}
```

## Contributing

1. Make changes to components in `src/components/`
2. Update API service in `src/services/api.js` if adding new endpoints
3. Test in development mode
4. Build and test production container
5. Update this README if needed

## License

Same as parent project
