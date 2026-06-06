// ============================================
// Drag & Drop File Upload
// ============================================

document.addEventListener('DOMContentLoaded', function() {
    const contentArea = document.querySelector('.content-area');
    if (!contentArea) return;

    let dragCounter = 0;

    contentArea.addEventListener('dragenter', function(e) {
        e.preventDefault();
        dragCounter++;
        showUploadOverlay();
    });

    contentArea.addEventListener('dragover', function(e) {
        e.preventDefault();
    });

    contentArea.addEventListener('dragleave', function(e) {
        dragCounter--;
        if (dragCounter === 0) {
            hideUploadOverlay();
        }
    });

    contentArea.addEventListener('drop', function(e) {
        e.preventDefault();
        dragCounter = 0;

        const files = e.dataTransfer.files;
        if (files.length > 0) {
            uploadFiles(files);
        } else {
            hideUploadOverlay();
        }
    });
});

function triggerUpload() {
    const input = document.getElementById('fileInput');
    if (input) input.click();
}

function handleFileSelect(event) {
    const files = event.target.files;
    if (files.length > 0) {
        uploadFiles(files);
    }
    event.target.value = ''; // Reset
}

function showUploadOverlay() {
    const overlay = document.getElementById('uploadOverlay');
    if (overlay) overlay.classList.add('active');
}

function hideUploadOverlay() {
    const overlay = document.getElementById('uploadOverlay');
    if (overlay) overlay.classList.remove('active');
}

async function uploadFiles(files) {
    const overlay = document.getElementById('uploadOverlay');
    const progress = document.getElementById('uploadProgress');
    const progressFill = document.getElementById('progressFill');
    const progressText = document.getElementById('progressText');

    if (progress) progress.style.display = 'block';

    const formData = new FormData();
    formData.append('path', window.currentPath || '/');

    for (let i = 0; i < files.length; i++) {
        formData.append('files', files[i]);
    }

    try {
        const xhr = new XMLHttpRequest();

        xhr.upload.addEventListener('progress', function(e) {
            if (e.lengthComputable) {
                const percent = Math.round((e.loaded / e.total) * 100);
                if (progressFill) progressFill.style.width = percent + '%';
                if (progressText) progressText.textContent = `Yükleniyor... ${percent}%`;
            }
        });

        xhr.addEventListener('load', function() {
            hideUploadOverlay();
            if (progress) progress.style.display = 'none';
            if (progressFill) progressFill.style.width = '0%';

            try {
                const result = JSON.parse(xhr.responseText);
                if (result.success) {
                    showToast(result.message || 'Dosyalar yüklendi', 'success');
                    setTimeout(refreshFiles, 500);
                } else {
                    showToast(result.error || 'Yükleme hatası', 'error');
                }
            } catch(e) {
                showToast('Yükleme hatası', 'error');
            }
        });

        xhr.addEventListener('error', function() {
            hideUploadOverlay();
            if (progress) progress.style.display = 'none';
            showToast('Yükleme başarısız', 'error');
        });

        xhr.open('POST', '/api/files/upload');
        if (window.csrfToken) {
            xhr.setRequestHeader('X-CSRF-Token', window.csrfToken);
        }
        xhr.setRequestHeader('X-Requested-With', 'XMLHttpRequest');
        xhr.send(formData);

    } catch (error) {
        hideUploadOverlay();
        if (progress) progress.style.display = 'none';
        showToast('Yükleme hatası: ' + error.message, 'error');
    }
}
