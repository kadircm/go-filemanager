// ============================================
// File Manager - File List UI Logic
// ============================================

let currentView = localStorage.getItem('view') || 'list';
let selectedFiles = [];

// Initialize view
document.addEventListener('DOMContentLoaded', function() {
    setView(currentView);
});

// View Toggle
function setView(view) {
    currentView = view;
    localStorage.setItem('view', view);

    const listView = document.getElementById('fileListView');
    const gridView = document.getElementById('fileGridView');
    const listBtn = document.getElementById('viewList');
    const gridBtn = document.getElementById('viewGrid');

    if (view === 'list') {
        if (listView) listView.classList.remove('hidden');
        if (gridView) gridView.classList.add('hidden');
        if (listBtn) listBtn.classList.add('active');
        if (gridBtn) gridBtn.classList.remove('active');
    } else {
        if (listView) listView.classList.add('hidden');
        if (gridView) gridView.classList.remove('hidden');
        if (listBtn) listBtn.classList.remove('active');
        if (gridBtn) gridBtn.classList.add('active');
    }
}

// Open item (double click)
function openItem(element) {
    const path = element.dataset.path;
    const isDir = element.dataset.isdir === 'true';
    const category = element.dataset.category;

    if (isDir) {
        window.location.href = '/files' + path;
    } else if (category === 'code' || category === 'document') {
        // Check if it's a text file
        const ext = path.split('.').pop().toLowerCase();
        const textExts = ['txt', 'log', 'csv', 'md', 'json', 'xml', 'yaml', 'yml',
                         'toml', 'ini', 'conf', 'cfg', 'env', 'sh', 'bash',
                         'go', 'py', 'js', 'ts', 'jsx', 'tsx', 'html', 'htm', 'css',
                         'scss', 'sql', 'php', 'java', 'c', 'cpp', 'h', 'rs', 'rb',
                         'lua', 'r', 'swift', 'kt', 'scala', 'pl', 'ex', 'exs',
                         'vue', 'dockerfile', 'makefile', 'gitignore'];
        if (textExts.includes(ext) || category === 'code') {
            window.location.href = '/editor' + path;
        } else {
            window.location.href = '/api/files/download' + path;
        }
    } else if (['image', 'video', 'audio'].includes(category)) {
        window.location.href = '/media' + path;
    } else {
        // Try to open as text, fallback to download
        const ext = path.split('.').pop().toLowerCase();
        const textExts = ['txt', 'log', 'csv', 'md', 'conf', 'cfg', 'ini', 'env'];
        if (textExts.includes(ext)) {
            window.location.href = '/editor' + path;
        } else {
            window.location.href = '/api/files/download' + path;
        }
    }
}

// Refresh Files
function refreshFiles() {
    window.location.reload();
}

// Select All Toggle
function toggleSelectAll() {
    const selectAll = document.getElementById('selectAll');
    const checkboxes = document.querySelectorAll('.file-checkbox');
    const rows = document.querySelectorAll('.file-row');

    checkboxes.forEach(cb => cb.checked = selectAll.checked);
    rows.forEach(row => {
        if (selectAll.checked) {
            row.classList.add('selected');
        } else {
            row.classList.remove('selected');
        }
    });

    updateSelection();
}

function updateSelection() {
    selectedFiles = [];
    const rows = document.querySelectorAll('.file-row');
    rows.forEach(row => {
        const cb = row.querySelector('.file-checkbox');
        if (cb && cb.checked) {
            row.classList.add('selected');
            selectedFiles.push({
                path: row.dataset.path,
                name: row.dataset.name,
                isDir: row.dataset.isdir === 'true'
            });
        } else {
            row.classList.remove('selected');
        }
    });
}

// New Folder Modal
function showNewFolderModal() {
    document.getElementById('modalTitle').textContent = 'Yeni Klasör Oluştur';
    document.getElementById('modalBody').innerHTML = `
        <div class="form-group">
            <label class="form-label">Klasör Adı</label>
            <input type="text" class="form-input" id="newFolderName" placeholder="klasor-adi" autofocus>
        </div>
    `;
    document.getElementById('modalFooter').innerHTML = `
        <button class="btn btn-secondary" onclick="closeModal()">İptal</button>
        <button class="btn btn-primary" onclick="createNewFolder()">Oluştur</button>
    `;
    document.getElementById('modalOverlay').classList.add('active');
    setTimeout(() => document.getElementById('newFolderName')?.focus(), 100);

    // Enter key support
    document.getElementById('newFolderName').addEventListener('keydown', function(e) {
        if (e.key === 'Enter') createNewFolder();
    });
}

async function createNewFolder() {
    const name = document.getElementById('newFolderName').value.trim();
    if (!name) {
        showToast('Klasör adı gerekli', 'warning');
        return;
    }

    const result = await apiCall('/api/files/mkdir', {
        method: 'POST',
        body: JSON.stringify({ path: window.currentPath, name: name })
    });

    if (result.success) {
        showToast('Klasör oluşturuldu: ' + name, 'success');
        closeModal();
        setTimeout(refreshFiles, 300);
    } else {
        showToast(result.error || 'Klasör oluşturulamadı', 'error');
    }
}

// New File Modal
function showNewFileModal() {
    document.getElementById('modalTitle').textContent = 'Yeni Dosya Oluştur';
    document.getElementById('modalBody').innerHTML = `
        <div class="form-group">
            <label class="form-label">Dosya Adı</label>
            <input type="text" class="form-input" id="newFileName" placeholder="dosya.txt" autofocus>
        </div>
    `;
    document.getElementById('modalFooter').innerHTML = `
        <button class="btn btn-secondary" onclick="closeModal()">İptal</button>
        <button class="btn btn-primary" onclick="createNewFile()">Oluştur</button>
    `;
    document.getElementById('modalOverlay').classList.add('active');
    setTimeout(() => document.getElementById('newFileName')?.focus(), 100);

    document.getElementById('newFileName').addEventListener('keydown', function(e) {
        if (e.key === 'Enter') createNewFile();
    });
}

async function createNewFile() {
    const name = document.getElementById('newFileName').value.trim();
    if (!name) {
        showToast('Dosya adı gerekli', 'warning');
        return;
    }

    const result = await apiCall('/api/files/create', {
        method: 'POST',
        body: JSON.stringify({ path: window.currentPath, name: name })
    });

    if (result.success) {
        showToast('Dosya oluşturuldu: ' + name, 'success');
        closeModal();
        setTimeout(refreshFiles, 300);
    } else {
        showToast(result.error || 'Dosya oluşturulamadı', 'error');
    }
}

// Rename Modal
function showRenameModal(path, currentName) {
    document.getElementById('modalTitle').textContent = 'Yeniden Adlandır';
    document.getElementById('modalBody').innerHTML = `
        <div class="form-group">
            <label class="form-label">Yeni Ad</label>
            <input type="text" class="form-input" id="renameInput" value="${currentName}" autofocus>
        </div>
    `;
    document.getElementById('modalFooter').innerHTML = `
        <button class="btn btn-secondary" onclick="closeModal()">İptal</button>
        <button class="btn btn-primary" onclick="renameItem('${path}')">Yeniden Adlandır</button>
    `;
    document.getElementById('modalOverlay').classList.add('active');

    setTimeout(() => {
        const input = document.getElementById('renameInput');
        if (input) {
            input.focus();
            // Select name without extension
            const dotIndex = currentName.lastIndexOf('.');
            if (dotIndex > 0) {
                input.setSelectionRange(0, dotIndex);
            } else {
                input.select();
            }
        }
    }, 100);

    document.getElementById('renameInput').addEventListener('keydown', function(e) {
        if (e.key === 'Enter') renameItem(path);
    });
}

async function renameItem(path) {
    const newName = document.getElementById('renameInput').value.trim();
    if (!newName) {
        showToast('Ad gerekli', 'warning');
        return;
    }

    const result = await apiCall('/api/files/rename', {
        method: 'PUT',
        body: JSON.stringify({ path: path, new_name: newName })
    });

    if (result.success) {
        showToast('Yeniden adlandırıldı', 'success');
        closeModal();
        setTimeout(refreshFiles, 300);
    } else {
        showToast(result.error || 'Yeniden adlandırılamadı', 'error');
    }
}

// Delete
async function deleteItem(path) {
    if (!confirm('Bu öğeyi çöp kutusuna taşımak istediğinize emin misiniz?')) return;

    const cleanPath = path.startsWith('/') ? path.substring(1) : path;
    const result = await apiCall('/api/files/' + cleanPath, {
        method: 'DELETE'
    });

    if (result.success) {
        showToast('Çöp kutusuna taşındı', 'success');
        setTimeout(refreshFiles, 300);
    } else {
        showToast(result.error || 'Silinemedi', 'error');
    }
}

// File Info Modal
async function showFileInfo(path) {
    const cleanPath = path.startsWith('/') ? path.substring(1) : path;
    const result = await apiCall('/api/files/info/' + cleanPath);

    if (result.success && result.data) {
        const info = result.data;
        document.getElementById('modalTitle').textContent = 'Dosya Bilgisi';
        document.getElementById('modalBody').innerHTML = `
            <table style="width: 100%; font-size: 0.88rem;">
                <tr><td class="text-muted" style="padding: 6px 0; width: 100px;">Ad:</td><td><strong>${info.name}</strong></td></tr>
                <tr><td class="text-muted" style="padding: 6px 0;">Yol:</td><td class="mono" style="font-size: 0.8rem; word-break: break-all;">${info.path}</td></tr>
                <tr><td class="text-muted" style="padding: 6px 0;">Tür:</td><td>${info.is_dir ? 'Klasör' : info.category}</td></tr>
                <tr><td class="text-muted" style="padding: 6px 0;">Boyut:</td><td>${info.size_human}</td></tr>
                <tr><td class="text-muted" style="padding: 6px 0;">Değiştirilme:</td><td>${info.mod_time_str}</td></tr>
                <tr><td class="text-muted" style="padding: 6px 0;">İzinler:</td><td class="mono">${info.permissions}</td></tr>
                ${info.mime_type ? '<tr><td class="text-muted" style="padding: 6px 0;">MIME:</td><td>' + info.mime_type + '</td></tr>' : ''}
            </table>
        `;
        document.getElementById('modalFooter').innerHTML = `
            <button class="btn btn-secondary" onclick="closeModal()">Kapat</button>
        `;
        document.getElementById('modalOverlay').classList.add('active');
    } else {
        showToast('Dosya bilgisi alınamadı', 'error');
    }
}
