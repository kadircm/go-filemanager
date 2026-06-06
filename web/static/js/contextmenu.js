// ============================================
// Context Menu Logic + Folder Browser for Copy/Move
// ============================================

let contextTarget = null;

function showContextMenu(event, element) {
    event.preventDefault();
    event.stopPropagation();

    contextTarget = element;
    const menu = document.getElementById('contextMenu');
    if (!menu) return;

    // Position menu
    const x = event.clientX;
    const y = event.clientY;

    menu.style.left = x + 'px';
    menu.style.top = y + 'px';
    menu.classList.add('show');

    // Adjust if off screen
    const rect = menu.getBoundingClientRect();
    if (rect.right > window.innerWidth) {
        menu.style.left = (x - rect.width) + 'px';
    }
    if (rect.bottom > window.innerHeight) {
        menu.style.top = (y - rect.height) + 'px';
    }

    // Show/hide edit option based on file type
    const editItem = document.getElementById('ctxEdit');
    const downloadItem = document.getElementById('ctxDownload');
    const isDir = element.dataset.isdir === 'true';

    if (editItem) {
        editItem.style.display = isDir ? 'none' : '';
    }
    if (downloadItem) {
        downloadItem.style.display = isDir ? 'none' : '';
    }

    // Highlight the row
    if (element.classList.contains('file-row')) {
        document.querySelectorAll('.file-row').forEach(r => r.classList.remove('selected'));
        element.classList.add('selected');
    }
}

// Close context menu on click elsewhere
document.addEventListener('click', function(e) {
    const menu = document.getElementById('contextMenu');
    if (menu && !menu.contains(e.target)) {
        menu.classList.remove('show');
    }
});

// Context Menu Actions
function ctxOpen() {
    hideContextMenu();
    if (contextTarget) openItem(contextTarget);
}

function ctxEdit() {
    hideContextMenu();
    if (!contextTarget) return;
    const path = contextTarget.dataset.path;
    window.location.href = '/editor' + path;
}

function ctxRename() {
    hideContextMenu();
    if (!contextTarget) return;
    showRenameModal(contextTarget.dataset.path, contextTarget.dataset.name);
}

// ============================================
// Folder Browser for Copy/Move
// ============================================

let folderBrowserCurrentPath = '/';
let folderBrowserMode = 'copy'; // 'copy' or 'move'
let folderBrowserSourcePath = '';

function buildBrowserBreadcrumb(path) {
    const parts = path.split('/').filter(p => p !== '');
    let html = `<span class="browser-breadcrumb-item" onclick="browserNavigate('/')" style="cursor:pointer;color:var(--primary);">📁 Ana Dizin</span>`;
    let currentPath = '';
    for (const part of parts) {
        currentPath += '/' + part;
        const pathCopy = currentPath;
        html += `<span class="breadcrumb-separator" style="margin:0 4px;color:var(--text-muted);">›</span>`;
        html += `<span class="browser-breadcrumb-item" onclick="browserNavigate('${pathCopy}')" style="cursor:pointer;color:var(--primary);">${part}</span>`;
    }
    return html;
}

async function browserNavigate(path) {
    folderBrowserCurrentPath = path;
    const result = await apiCall('/api/files/browse?path=' + encodeURIComponent(path));

    if (!result.success || !result.data) {
        showToast('Dizin yüklenemedi', 'error');
        return;
    }

    renderFolderBrowser(result.data.files || [], path);
}

function renderFolderBrowser(files, path) {
    const breadcrumbHtml = buildBrowserBreadcrumb(path);

    // Separate folders and files
    const folders = files.filter(f => f.is_dir);
    const fileItems = files.filter(f => !f.is_dir);

    let listHtml = '';

    // Parent directory link
    if (path !== '/') {
        const parentPath = path.split('/').slice(0, -1).join('/') || '/';
        listHtml += `<div class="browser-item browser-folder" onclick="browserNavigate('${parentPath}')" style="cursor:pointer;">
            <span class="icon">⬆️</span> <span>..</span> <span class="text-muted" style="font-size:0.8rem;margin-left:auto;">(Üst Dizin)</span>
        </div>`;
    }

    // Folders
    for (const folder of folders) {
        const folderPath = folder.path;
        listHtml += `<div class="browser-item browser-folder" onclick="browserSelectFolder('${folderPath}')" ondblclick="browserNavigate('${folderPath}')" style="cursor:pointer;" data-path="${folderPath}">
            <span class="icon">📁</span> <span>${folder.name}</span>
            <span class="text-muted" style="font-size:0.8rem;margin-left:auto;">Klasör</span>
        </div>`;
    }

    // Files (shown but not selectable as destination)
    for (const file of fileItems) {
        listHtml += `<div class="browser-item browser-file" style="opacity:0.6;">
            <span class="icon">${getFileIcon(file.category)}</span> <span>${file.name}</span>
            <span class="text-muted" style="font-size:0.8rem;margin-left:auto;">${file.size_human}</span>
        </div>`;
    }

    if (folders.length === 0 && fileItems.length === 0 && path !== '/') {
        listHtml = '<div class="browser-item" style="justify-content:center;color:var(--text-muted);">Bu klasör boş</div>';
    }

    const selectedDisplay = document.getElementById('browserSelectedPath');
    if (selectedDisplay) {
        selectedDisplay.textContent = folderBrowserCurrentPath;
    }

    const breadcrumbEl = document.getElementById('browserBreadcrumb');
    if (breadcrumbEl) {
        breadcrumbEl.innerHTML = breadcrumbHtml;
    }

    const listEl = document.getElementById('browserFileList');
    if (listEl) {
        listEl.innerHTML = listHtml;
    }
}

function getFileIcon(category) {
    const icons = {
        'image': '🖼️', 'video': '🎬', 'audio': '🎵',
        'code': '💻', 'document': '📄', 'archive': '📦',
        'folder': '📁', 'other': '📎'
    };
    return icons[category] || '📎';
}

function browserSelectFolder(path) {
    folderBrowserCurrentPath = path;

    // Highlight selected folder
    document.querySelectorAll('.browser-item').forEach(item => {
        item.classList.remove('browser-selected');
    });
    const items = document.querySelectorAll(`.browser-item[data-path="${path}"]`);
    items.forEach(item => item.classList.add('browser-selected'));

    const selectedDisplay = document.getElementById('browserSelectedPath');
    if (selectedDisplay) {
        selectedDisplay.textContent = path;
    }
}

function showFolderBrowserModal(mode, sourcePath) {
    folderBrowserMode = mode;
    folderBrowserSourcePath = sourcePath;
    folderBrowserCurrentPath = '/';

    const title = mode === 'copy' ? 'Kopyala' : 'Taşı';
    const actionText = mode === 'copy' ? 'Kopyala' : 'Taşı';
    const sourceName = sourcePath.split('/').pop();

    document.getElementById('modalTitle').textContent = title;
    document.getElementById('modalBody').innerHTML = `
        <div class="form-group" style="margin-bottom:8px;">
            <label class="form-label">Kaynak</label>
            <div class="mono" style="font-size:0.85rem;padding:8px 12px;background:var(--bg-card);border-radius:8px;border:1px solid var(--border-color);word-break:break-all;">${sourcePath}</div>
        </div>
        <div class="form-group" style="margin-bottom:8px;">
            <label class="form-label">Hedef Klasör</label>
            <div id="browserBreadcrumb" style="padding:6px 0;font-size:0.85rem;display:flex;align-items:center;flex-wrap:wrap;gap:2px;"></div>
            <div id="browserFileList" style="max-height:280px;overflow-y:auto;border:1px solid var(--border-color);border-radius:10px;background:var(--bg-card);">
                <div style="padding:20px;text-align:center;color:var(--text-muted);">Yükleniyor...</div>
            </div>
        </div>
        <div class="form-group" style="margin-bottom:0;">
            <label class="form-label" style="font-size:0.8rem;color:var(--text-muted);">Seçili Hedef:</label>
            <div id="browserSelectedPath" class="mono" style="font-size:0.85rem;color:var(--primary);padding:4px 0;">/</div>
        </div>
        <style>
            .browser-item {
                display: flex;
                align-items: center;
                gap: 8px;
                padding: 8px 12px;
                border-bottom: 1px solid var(--border-color);
                transition: all 0.15s;
                font-size: 0.88rem;
            }
            .browser-item:last-child { border-bottom: none; }
            .browser-folder:hover {
                background: var(--primary-glow);
            }
            .browser-selected {
                background: var(--primary-glow) !important;
                border-left: 3px solid var(--primary);
            }
            .browser-breadcrumb-item:hover {
                text-decoration: underline;
            }
        </style>
    `;
    document.getElementById('modalFooter').innerHTML = `
        <button class="btn btn-secondary" onclick="closeModal()">İptal</button>
        <button class="btn btn-primary" onclick="executeFileBrowserAction()">${actionText}</button>
    `;
    document.getElementById('modalOverlay').classList.add('active');

    // Load root directory
    browserNavigate('/');
}

async function executeFileBrowserAction() {
    const sourcePath = folderBrowserSourcePath;
    const sourceName = sourcePath.split('/').pop();
    let destFolder = folderBrowserCurrentPath;

    // Build destination path
    let destPath = destFolder === '/' ? '/' + sourceName : destFolder + '/' + sourceName;

    // Check if destination exists
    const checkResult = await apiCall('/api/files/check-exists?path=' + encodeURIComponent(destPath));

    if (checkResult.success && checkResult.data && checkResult.data.exists) {
        // Show overwrite confirmation
        const overwriteChoice = await showOverwriteDialog(destPath, checkResult.data.is_dir);
        if (overwriteChoice === 'cancel') return;
        if (overwriteChoice === 'rename') {
            // Auto-rename: add _copy suffix
            const ext = sourceName.includes('.') ? '.' + sourceName.split('.').pop() : '';
            const nameWithoutExt = ext ? sourceName.slice(0, -ext.length) : sourceName;
            destPath = destFolder === '/'
                ? '/' + nameWithoutExt + '_copy' + ext
                : destFolder + '/' + nameWithoutExt + '_copy' + ext;
        }
        // overwriteChoice === 'overwrite' -> proceed as is
        await doFileBrowserAction(sourcePath, destPath, overwriteChoice === 'overwrite');
    } else {
        await doFileBrowserAction(sourcePath, destPath, false);
    }
}

function showOverwriteDialog(destPath, isDir) {
    return new Promise((resolve) => {
        const itemType = isDir ? 'klasör' : 'dosya';
        const fileName = destPath.split('/').pop();

        document.getElementById('modalTitle').textContent = '⚠️ Hedef Mevcut';
        document.getElementById('modalBody').innerHTML = `
            <div style="text-align:center;padding:16px 0;">
                <div style="font-size:2.5rem;margin-bottom:12px;">⚠️</div>
                <p style="font-size:0.95rem;margin-bottom:8px;">Hedef konumda <strong>"${fileName}"</strong> adında bir ${itemType} zaten mevcut.</p>
                <p class="text-muted" style="font-size:0.85rem;word-break:break-all;">${destPath}</p>
            </div>
        `;
        document.getElementById('modalFooter').innerHTML = `
            <button class="btn btn-secondary" onclick="window._overwriteResolve('cancel')">İptal</button>
            <button class="btn btn-secondary" onclick="window._overwriteResolve('rename')" style="background:var(--info-bg);color:var(--primary);">Yeniden Adlandır</button>
            <button class="btn btn-primary" onclick="window._overwriteResolve('overwrite')" style="background:var(--critical);border-color:var(--critical);">Üzerine Yaz</button>
        `;

        window._overwriteResolve = (choice) => {
            delete window._overwriteResolve;
            resolve(choice);
        };
    });
}

async function doFileBrowserAction(sourcePath, destPath, overwrite) {
    const url = folderBrowserMode === 'copy' ? '/api/files/copy' : '/api/files/move';
    const method = folderBrowserMode === 'copy' ? 'POST' : 'PUT';

    const result = await apiCall(url, {
        method: method,
        body: JSON.stringify({ source: sourcePath, destination: destPath, overwrite: overwrite })
    });

    if (result.success) {
        const actionText = folderBrowserMode === 'copy' ? 'Kopyalandı' : 'Taşındı';
        showToast(actionText, 'success');
        closeModal();
        setTimeout(refreshFiles, 300);
    } else {
        showToast(result.error || 'İşlem başarısız', 'error');
    }
}

function ctxCopy() {
    hideContextMenu();
    if (!contextTarget) return;
    showFolderBrowserModal('copy', contextTarget.dataset.path);
}

function ctxMove() {
    hideContextMenu();
    if (!contextTarget) return;
    showFolderBrowserModal('move', contextTarget.dataset.path);
}

function ctxDownload() {
    hideContextMenu();
    if (!contextTarget) return;
    const path = contextTarget.dataset.path;
    window.location.href = '/api/files/download' + path;
}

function ctxInfo() {
    hideContextMenu();
    if (!contextTarget) return;
    showFileInfo(contextTarget.dataset.path);
}

function ctxDelete() {
    hideContextMenu();
    if (!contextTarget) return;
    deleteItem(contextTarget.dataset.path);
}

function ctxChangeOwner() {
    hideContextMenu();
    if (!contextTarget) return;
    showChangeOwnerModal(contextTarget.dataset.path);
}

function ctxChangePermissions() {
    hideContextMenu();
    if (!contextTarget) return;
    showChangePermissionsModal(contextTarget.dataset.path);
}

function hideContextMenu() {
    const menu = document.getElementById('contextMenu');
    if (menu) menu.classList.remove('show');
}
