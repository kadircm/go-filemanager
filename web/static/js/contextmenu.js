// ============================================
// Context Menu Logic
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
    const category = element.dataset.category;

    if (editItem) {
        if (isDir) {
            editItem.style.display = 'none';
        } else {
            editItem.style.display = '';
        }
    }

    if (downloadItem) {
        if (isDir) {
            downloadItem.style.display = 'none';
        } else {
            downloadItem.style.display = '';
        }
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

function ctxCopy() {
    hideContextMenu();
    if (!contextTarget) return;
    const path = contextTarget.dataset.path;

    document.getElementById('modalTitle').textContent = 'Kopyala';
    document.getElementById('modalBody').innerHTML = `
        <div class="form-group">
            <label class="form-label">Kaynak</label>
            <input type="text" class="form-input" value="${path}" disabled>
        </div>
        <div class="form-group">
            <label class="form-label">Hedef Yol</label>
            <input type="text" class="form-input" id="copyDest" value="${path}_copy" autofocus>
        </div>
    `;
    document.getElementById('modalFooter').innerHTML = `
        <button class="btn btn-secondary" onclick="closeModal()">İptal</button>
        <button class="btn btn-primary" onclick="doCopy('${path}')">Kopyala</button>
    `;
    document.getElementById('modalOverlay').classList.add('active');
}

async function doCopy(source) {
    const dest = document.getElementById('copyDest').value.trim();
    if (!dest) return;

    const result = await apiCall('/api/files/copy', {
        method: 'POST',
        body: JSON.stringify({ source: source, destination: dest })
    });

    if (result.success) {
        showToast('Kopyalandı', 'success');
        closeModal();
        setTimeout(refreshFiles, 300);
    } else {
        showToast(result.error || 'Kopyalanamadı', 'error');
    }
}

function ctxMove() {
    hideContextMenu();
    if (!contextTarget) return;
    const path = contextTarget.dataset.path;

    document.getElementById('modalTitle').textContent = 'Taşı';
    document.getElementById('modalBody').innerHTML = `
        <div class="form-group">
            <label class="form-label">Kaynak</label>
            <input type="text" class="form-input" value="${path}" disabled>
        </div>
        <div class="form-group">
            <label class="form-label">Hedef Yol</label>
            <input type="text" class="form-input" id="moveDest" value="${path}" autofocus>
        </div>
    `;
    document.getElementById('modalFooter').innerHTML = `
        <button class="btn btn-secondary" onclick="closeModal()">İptal</button>
        <button class="btn btn-primary" onclick="doMove('${path}')">Taşı</button>
    `;
    document.getElementById('modalOverlay').classList.add('active');
}

async function doMove(source) {
    const dest = document.getElementById('moveDest').value.trim();
    if (!dest) return;

    const result = await apiCall('/api/files/move', {
        method: 'PUT',
        body: JSON.stringify({ source: source, destination: dest })
    });

    if (result.success) {
        showToast('Taşındı', 'success');
        closeModal();
        setTimeout(refreshFiles, 300);
    } else {
        showToast(result.error || 'Taşınamadı', 'error');
    }
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

function hideContextMenu() {
    const menu = document.getElementById('contextMenu');
    if (menu) menu.classList.remove('show');
}
