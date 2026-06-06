// ============================================
// Trash UI Logic
// ============================================

async function restoreItem(id) {
    if (!confirm('Bu öğeyi orijinal konumuna geri yüklemek istediğinize emin misiniz?')) return;

    const result = await apiCall('/api/trash/restore', {
        method: 'POST',
        body: JSON.stringify({ id: id })
    });

    if (result.success) {
        showToast('Geri yüklendi', 'success');
        setTimeout(() => location.reload(), 500);
    } else {
        showToast(result.error || 'Geri yüklenemedi', 'error');
    }
}

async function deletePermanent(id) {
    if (!confirm('Bu öğeyi kalıcı olarak silmek istediğinize emin misiniz? Bu işlem geri alınamaz!')) return;

    const result = await apiCall('/api/trash/' + id, {
        method: 'DELETE'
    });

    if (result.success) {
        showToast('Kalıcı olarak silindi', 'success');
        setTimeout(() => location.reload(), 500);
    } else {
        showToast(result.error || 'Silinemedi', 'error');
    }
}

async function emptyTrash() {
    if (!confirm('Çöp kutusundaki TÜM öğeleri kalıcı olarak silmek istediğinize emin misiniz? Bu işlem geri alınamaz!')) return;

    const result = await apiCall('/api/trash', {
        method: 'DELETE'
    });

    if (result.success) {
        showToast('Çöp kutusu boşaltıldı', 'success');
        setTimeout(() => location.reload(), 500);
    } else {
        showToast(result.error || 'Çöp kutusu boşaltılamadı', 'error');
    }
}
