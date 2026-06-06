// ============================================
// Chunk Upload System - Performance Optimized
// ============================================

const CHUNK_SIZE = 2 * 1024 * 1024; // 2MB chunks
const MAX_RETRIES = 3;
const CONCURRENT_CHUNKS = 3;

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

function generateUploadId() {
    return 'upload_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
}

async function uploadFiles(files) {
    const progress = document.getElementById('uploadProgress');
    const progressFill = document.getElementById('progressFill');
    const progressText = document.getElementById('progressText');

    if (progress) progress.style.display = 'block';

    let totalUploaded = 0;
    let totalFiles = files.length;
    let completedFiles = 0;

    for (let i = 0; i < files.length; i++) {
        const file = files[i];

        if (progressText) {
            progressText.textContent = `${file.name} yükleniyor... (${i + 1}/${totalFiles})`;
        }

        try {
            if (file.size > CHUNK_SIZE) {
                // Chunk upload for large files
                await uploadFileChunked(file, (percent) => {
                    const overallPercent = Math.round(((completedFiles + percent / 100) / totalFiles) * 100);
                    if (progressFill) progressFill.style.width = overallPercent + '%';
                    if (progressText) {
                        progressText.textContent = `${file.name} yükleniyor... ${percent}% (${i + 1}/${totalFiles})`;
                    }
                });
            } else {
                // Standard upload for small files
                await uploadFileStandard(file, (percent) => {
                    const overallPercent = Math.round(((completedFiles + percent / 100) / totalFiles) * 100);
                    if (progressFill) progressFill.style.width = overallPercent + '%';
                    if (progressText) {
                        progressText.textContent = `${file.name} yükleniyor... ${percent}% (${i + 1}/${totalFiles})`;
                    }
                });
            }
            completedFiles++;
        } catch (error) {
            console.error(`Upload failed for ${file.name}:`, error);
            showToast(`${file.name} yüklenemedi: ${error.message}`, 'error');
        }
    }

    hideUploadOverlay();
    if (progress) progress.style.display = 'none';
    if (progressFill) progressFill.style.width = '0%';

    if (completedFiles > 0) {
        showToast(`${completedFiles}/${totalFiles} dosya yüklendi`, 'success');
        setTimeout(refreshFiles, 500);
    }
}

// Standard upload for small files (< CHUNK_SIZE)
function uploadFileStandard(file, onProgress) {
    return new Promise((resolve, reject) => {
        const formData = new FormData();
        formData.append('path', window.currentPath || '/');
        formData.append('files', file);

        const xhr = new XMLHttpRequest();

        xhr.upload.addEventListener('progress', function(e) {
            if (e.lengthComputable) {
                const percent = Math.round((e.loaded / e.total) * 100);
                onProgress(percent);
            }
        });

        xhr.addEventListener('load', function() {
            try {
                const result = JSON.parse(xhr.responseText);
                if (result.success) {
                    resolve(result);
                } else {
                    reject(new Error(result.error || 'Yükleme hatası'));
                }
            } catch(e) {
                reject(new Error('Yanıt okunamadı'));
            }
        });

        xhr.addEventListener('error', function() {
            reject(new Error('Bağlantı hatası'));
        });

        xhr.open('POST', '/api/files/upload');
        if (window.csrfToken) {
            xhr.setRequestHeader('X-CSRF-Token', window.csrfToken);
        }
        xhr.setRequestHeader('X-Requested-With', 'XMLHttpRequest');
        xhr.send(formData);
    });
}

// Chunked upload for large files (>= CHUNK_SIZE)
async function uploadFileChunked(file, onProgress) {
    const uploadId = generateUploadId();
    const totalChunks = Math.ceil(file.size / CHUNK_SIZE);
    let uploadedChunks = 0;

    // Process chunks with concurrency control
    const chunkIndexes = Array.from({ length: totalChunks }, (_, i) => i);

    // Upload chunks in batches
    for (let batchStart = 0; batchStart < chunkIndexes.length; batchStart += CONCURRENT_CHUNKS) {
        const batch = chunkIndexes.slice(batchStart, batchStart + CONCURRENT_CHUNKS);
        const promises = batch.map(chunkIndex => uploadChunkWithRetry(file, uploadId, chunkIndex, totalChunks));

        const results = await Promise.all(promises);
        uploadedChunks += results.length;

        const percent = Math.round((uploadedChunks / totalChunks) * 100);
        onProgress(percent);

        // Check if last batch completed the upload
        for (const result of results) {
            if (result && result.data && result.data.completed) {
                onProgress(100);
                return result;
            }
        }
    }
}

// Upload a single chunk with retry logic
async function uploadChunkWithRetry(file, uploadId, chunkIndex, totalChunks) {
    let lastError;

    for (let attempt = 0; attempt < MAX_RETRIES; attempt++) {
        try {
            return await uploadSingleChunk(file, uploadId, chunkIndex, totalChunks);
        } catch (error) {
            lastError = error;
            console.warn(`Chunk ${chunkIndex} attempt ${attempt + 1} failed:`, error.message);
            // Wait before retry (exponential backoff)
            if (attempt < MAX_RETRIES - 1) {
                await new Promise(resolve => setTimeout(resolve, 1000 * (attempt + 1)));
            }
        }
    }

    throw lastError;
}

// Upload a single chunk
function uploadSingleChunk(file, uploadId, chunkIndex, totalChunks) {
    return new Promise((resolve, reject) => {
        const start = chunkIndex * CHUNK_SIZE;
        const end = Math.min(start + CHUNK_SIZE, file.size);
        const chunk = file.slice(start, end);

        const formData = new FormData();
        formData.append('upload_id', uploadId);
        formData.append('chunk_index', chunkIndex.toString());
        formData.append('total_chunks', totalChunks.toString());
        formData.append('filename', file.name);
        formData.append('path', window.currentPath || '/');
        formData.append('chunk', chunk, `chunk_${chunkIndex}`);

        const xhr = new XMLHttpRequest();

        xhr.addEventListener('load', function() {
            try {
                const result = JSON.parse(xhr.responseText);
                if (result.success) {
                    resolve(result);
                } else {
                    reject(new Error(result.error || 'Chunk yükleme hatası'));
                }
            } catch(e) {
                reject(new Error('Yanıt okunamadı'));
            }
        });

        xhr.addEventListener('error', function() {
            reject(new Error('Bağlantı hatası'));
        });

        xhr.open('POST', '/api/files/upload/chunk');
        if (window.csrfToken) {
            xhr.setRequestHeader('X-CSRF-Token', window.csrfToken);
        }
        xhr.setRequestHeader('X-Requested-With', 'XMLHttpRequest');
        xhr.send(formData);
    });
}
