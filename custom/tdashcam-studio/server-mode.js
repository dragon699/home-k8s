(() => {
    const config = window.TDASHCAM_SERVER;
    if (!config || !config.mediaUrl) return;

    const CLIP_DIRS = ['RecentClips', 'SavedClips', 'SentryClips'];
    const FILE_PATTERN = /(\.mp4|^event\.json|^thumb\.png)$/;

    class ServerFile {
        constructor(name, relativePath, url, size, lastModified) {
            this.name = name;
            this.webkitRelativePath = relativePath;
            this.url = url;
            this.path = url;
            this.size = size || 0;
            this.lastModified = lastModified || 0;
            this.type = name.endsWith('.mp4') ? 'video/mp4'
                : name.endsWith('.json') ? 'application/json'
                : name.endsWith('.png') ? 'image/png' : '';
        }

        async fetchOk() {
            const response = await fetch(this.url);
            if (!response.ok) throw new Error(`HTTP ${response.status} for ${this.url}`);
            return response;
        }

        async text() {
            return (await this.fetchOk()).text();
        }

        async arrayBuffer() {
            return (await this.fetchOk()).arrayBuffer();
        }
    }

    const encodePath = (relativePath) => relativePath.split('/').filter(Boolean).map(encodeURIComponent).join('/');

    async function listDir(relativePath) {
        const encoded = encodePath(relativePath);
        const response = await fetch(config.mediaUrl + (encoded ? encoded + '/' : ''));
        if (!response.ok) throw new Error(`HTTP ${response.status} listing ${relativePath || '/'}`);
        return response.json();
    }

    async function findLibraryDirs() {
        if (Array.isArray(config.libraryDirs) && config.libraryDirs.length > 0) return config.libraryDirs;
        const excluded = new Set(config.excludeDirs || []);
        const libraries = [];
        const nested = [];
        for (const entry of await listDir('')) {
            if (entry.type !== 'directory' || excluded.has(entry.name)) continue;
            if (CLIP_DIRS.includes(entry.name)) libraries.push(entry.name);
            else nested.push(entry.name);
        }
        for (const parent of nested) {
            for (const entry of await listDir(parent)) {
                if (entry.type === 'directory' && CLIP_DIRS.includes(entry.name)) libraries.push(`${parent}/${entry.name}`);
            }
        }
        return libraries;
    }

    async function scanLibrary() {
        const files = [];
        const addFile = (directory, entry) => {
            if (entry.type !== 'file' || !FILE_PATTERN.test(entry.name)) return;
            const relativePath = `${directory}/${entry.name}`;
            files.push(new ServerFile(entry.name, relativePath, config.mediaUrl + encodePath(relativePath), entry.size, Date.parse(entry.mtime) || 0));
        };

        const eventDirs = [];
        for (const library of await findLibraryDirs()) {
            for (const entry of await listDir(library)) {
                if (entry.type === 'directory') eventDirs.push(`${library}/${entry.name}`);
                else addFile(library, entry);
            }
        }

        let next = 0;
        const worker = async () => {
            while (next < eventDirs.length) {
                const directory = eventDirs[next++];
                try {
                    for (const entry of await listDir(directory)) addFile(directory, entry);
                } catch (e) {
                    console.warn('[ServerMode] Failed to list', directory, e);
                }
            }
        };
        await Promise.all(Array.from({ length: 8 }, worker));
        return files;
    }

    async function loadLibrary(viewer) {
        const loading = document.createElement('div');
        loading.id = 'server-loading';
        loading.style.cssText = 'position:fixed;top:50%;left:50%;transform:translate(-50%,-50%);padding:20px;background:rgba(0,0,0,0.8);color:white;border-radius:10px;z-index:9999';
        loading.innerText = 'Scanning files...';
        document.body.appendChild(loading);

        try {
            const files = await scanLibrary();
            console.log('[ServerMode] Files found:', files.length);
            if (files.length > 0) await viewer.handleFolderSelection(files);
        } catch (e) {
            console.error('[ServerMode] Library scan failed:', e);
            alert('Error reading files: ' + e.message);
        } finally {
            loading.remove();
        }
    }

    const originalGetFileUrl = getFileUrl;
    getFileUrl = (file) => (file instanceof ServerFile ? file.url : originalGetFileUrl(file));

    const originalFetchFile = VideoClipProcessor.prototype.fetchFileAsUint8Array;
    VideoClipProcessor.prototype.fetchFileAsUint8Array = function (file) {
        return originalFetchFile.call(this, file instanceof ServerFile ? file.url : file);
    };

    document.addEventListener('DOMContentLoaded', () => {
        if (window.viewer) loadLibrary(window.viewer);
    });
})();
