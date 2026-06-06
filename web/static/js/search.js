// ============================================
// Search UI Logic
// ============================================

let currentFilter = '';

document.addEventListener('DOMContentLoaded', function() {
    const searchInput = document.getElementById('searchInput');
    if (searchInput) {
        searchInput.addEventListener('keydown', function(e) {
            if (e.key === 'Enter') {
                performSearch();
            }
        });

        // Auto-search if query param exists
        if (searchInput.value.trim()) {
            performSearch();
        }
    }
});

function setFilter(element) {
    document.querySelectorAll('.filter-chip').forEach(c => c.classList.remove('active'));
    element.classList.add('active');
    currentFilter = element.dataset.type;
    performSearch();
}

async function performSearch() {
    const query = document.getElementById('searchInput').value.trim();
    if (!query) {
        showToast('Arama sorgusu girin', 'warning');
        return;
    }

    const searchTrash = document.getElementById('searchTrash')?.checked;
    let url = `/api/search?q=${encodeURIComponent(query)}`;

    if (currentFilter) url += `&type=${currentFilter}`;
    if (searchTrash) url += `&trash=true`;

    const resultsDiv = document.getElementById('searchResults');
    if (resultsDiv) {
        resultsDiv.innerHTML = '<div class="text-center text-muted" style="padding: 40px;">Aranıyor...</div>';
    }

    const result = await apiCall(url);

    if (result.success && result.data) {
        renderSearchResults(result.data);
    } else {
        if (resultsDiv) {
            resultsDiv.innerHTML = `<div class="empty-state">
                <div class="empty-icon">❌</div>
                <div class="empty-title">Arama hatası</div>
                <div class="empty-text">${result.error || 'Bilinmeyen hata'}</div>
            </div>`;
        }
    }
}

function renderSearchResults(data) {
    const resultsDiv = document.getElementById('searchResults');
    if (!resultsDiv) return;

    const results = data.results || [];

    if (results.length === 0) {
        resultsDiv.innerHTML = `<div class="empty-state">
            <div class="empty-icon">🔍</div>
            <div class="empty-title">Sonuç bulunamadı</div>
            <div class="empty-text">"${data.query}" araması için sonuç yok</div>
        </div>`;
        return;
    }

    let html = `<div class="search-results-count mb-2">${results.length} sonuç bulundu</div>`;
    html += '<table class="file-table"><thead><tr>';
    html += '<th>Ad</th><th style="width:200px;">Konum</th><th style="width:100px;">Boyut</th>';
    html += '<th style="width:160px;">Tarih</th><th style="width:80px;">Tür</th></tr></thead><tbody>';

    const iconMap = {
        folder: '📁', image: '🖼️', video: '🎬', audio: '🎵',
        code: '💻', document: '📄', archive: '📦', other: '📎'
    };

    results.forEach(item => {
        const icon = iconMap[item.category] || '📎';
        const link = item.is_dir ? `/files${item.path}` :
                     ['code', 'document'].includes(item.category) ? `/editor${item.path}` :
                     ['image', 'video', 'audio'].includes(item.category) ? `/media${item.path}` :
                     `/api/files/download${item.path}`;

        html += `<tr style="cursor:pointer" ondblclick="location.href='${link}'">`;
        html += `<td><div class="file-name-cell">`;
        html += `<div class="file-icon ${item.category}">${icon}</div>`;
        html += `<a href="${link}" class="file-name">${item.name}</a>`;
        html += `</div></td>`;
        html += `<td class="text-muted mono" style="font-size:0.78rem;">${item.parent_dir}</td>`;
        html += `<td class="file-size">${item.is_dir ? '—' : item.size_human}</td>`;
        html += `<td class="file-date">${item.mod_time_str}</td>`;
        html += `<td><span class="filter-chip" style="cursor:default;font-size:0.72rem;">${item.category}</span></td>`;
        html += `</tr>`;
    });

    html += '</tbody></table>';
    resultsDiv.innerHTML = html;
}
