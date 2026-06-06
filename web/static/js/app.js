// ============================================
// Go File Manager - Main Application JS
// ============================================

// Theme Management
function initTheme() {
    const saved = localStorage.getItem('theme') || 'dark';
    document.documentElement.setAttribute('data-theme', saved);
    updateThemeIcon(saved);
}

function toggleTheme() {
    const current = document.documentElement.getAttribute('data-theme');
    const next = current === 'dark' ? 'light' : 'dark';
    document.documentElement.setAttribute('data-theme', next);
    localStorage.setItem('theme', next);
    updateThemeIcon(next);

    // Update CodeMirror theme if editor is present
    if (window.editor) {
        window.editor.setOption('theme', next === 'dark' ? 'dracula' : 'default');
    }
}

function updateThemeIcon(theme) {
    const toggleBtns = document.querySelectorAll('.theme-toggle');
    toggleBtns.forEach(btn => {
        btn.textContent = theme === 'dark' ? '☀️' : '🌙';
    });
}

// Toast Notification System
function showToast(message, type = 'info', duration = 3000) {
    const container = document.getElementById('toastContainer');
    if (!container) return;

    const icons = {
        success: '✓',
        error: '✕',
        warning: '⚠',
        info: 'ℹ'
    };

    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    toast.innerHTML = `
        <span class="toast-icon">${icons[type] || 'ℹ'}</span>
        <span class="toast-message">${message}</span>
        <button class="toast-close" onclick="this.parentElement.remove()">✕</button>
    `;

    container.appendChild(toast);

    setTimeout(() => {
        toast.classList.add('toast-exit');
        setTimeout(() => toast.remove(), 250);
    }, duration);
}

// API Helper
async function apiCall(url, options = {}) {
    const defaults = {
        headers: {
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest',
        },
    };

    if (window.csrfToken) {
        defaults.headers['X-CSRF-Token'] = window.csrfToken;
    }

    const config = {
        ...defaults,
        ...options,
        headers: {
            ...defaults.headers,
            ...options.headers,
        },
    };

    try {
        const response = await fetch(url, config);
        const data = await response.json();
        return data;
    } catch (error) {
        console.error('API Error:', error);
        return { success: false, error: 'Sunucuya bağlanılamadı' };
    }
}

// Sidebar Toggle (Mobile)
function toggleSidebar() {
    const sidebar = document.getElementById('sidebar');
    if (sidebar) {
        sidebar.classList.toggle('open');
    }
}

// Close sidebar on outside click (mobile)
document.addEventListener('click', function(e) {
    const sidebar = document.getElementById('sidebar');
    const toggle = document.getElementById('menuToggle');
    if (sidebar && sidebar.classList.contains('open') &&
        !sidebar.contains(e.target) && e.target !== toggle) {
        sidebar.classList.remove('open');
    }
});

// Keyboard Shortcuts
document.addEventListener('keydown', function(e) {
    // Ctrl+K - Focus search
    if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
        e.preventDefault();
        const searchInput = document.getElementById('quickSearch') || document.getElementById('searchInput');
        if (searchInput) searchInput.focus();
    }

    // Escape - Close modals, menus
    if (e.key === 'Escape') {
        closeAllMenus();
    }
});

function closeAllMenus() {
    // Close context menu
    const ctx = document.getElementById('contextMenu');
    if (ctx) ctx.classList.remove('show');

    // Close user dropdown
    const userDrop = document.getElementById('userDropdown');
    if (userDrop) userDrop.classList.remove('show');

    // Close modal
    const modal = document.getElementById('modalOverlay');
    if (modal) modal.classList.remove('active');

    // Close upload overlay
    const upload = document.getElementById('uploadOverlay');
    if (upload) upload.classList.remove('active');
}

// Quick Search from Header
document.addEventListener('DOMContentLoaded', function() {
    const quickSearch = document.getElementById('quickSearch');
    if (quickSearch) {
        let timeout;
        quickSearch.addEventListener('keydown', function(e) {
            if (e.key === 'Enter') {
                e.preventDefault();
                const query = this.value.trim();
                if (query) {
                    window.location.href = '/search/?q=' + encodeURIComponent(query);
                }
            }
        });
    }

    // User menu toggle
    const userMenu = document.getElementById('userMenu');
    if (userMenu) {
        userMenu.addEventListener('click', function(e) {
            e.stopPropagation();
            const dropdown = document.getElementById('userDropdown');
            if (dropdown) {
                const rect = userMenu.getBoundingClientRect();
                dropdown.style.left = rect.left + 'px';
                dropdown.style.bottom = (window.innerHeight - rect.top + 8) + 'px';
                dropdown.style.top = 'auto';
                dropdown.classList.toggle('show');
            }
        });
    }
});

// Show Change Password Modal
function showChangePasswordModal() {
    const modalOverlay = document.getElementById('modalOverlay');
    if (!modalOverlay) return;

    document.getElementById('modalTitle').textContent = 'Şifre Değiştir';
    document.getElementById('modalBody').innerHTML = `
        <div class="form-group">
            <label class="form-label">Mevcut Şifre</label>
            <input type="password" class="form-input" id="oldPassword" placeholder="Mevcut şifreniz">
        </div>
        <div class="form-group">
            <label class="form-label">Yeni Şifre</label>
            <input type="password" class="form-input" id="newPassword" placeholder="Yeni şifre (min 6 karakter)">
        </div>
    `;
    document.getElementById('modalFooter').innerHTML = `
        <button class="btn btn-secondary" onclick="closeModal()">İptal</button>
        <button class="btn btn-primary" onclick="changePassword()">Değiştir</button>
    `;
    modalOverlay.classList.add('active');
    closeAllMenus();
}

async function changePassword() {
    const oldPassword = document.getElementById('oldPassword').value;
    const newPassword = document.getElementById('newPassword').value;

    if (!oldPassword || !newPassword) {
        showToast('Tüm alanları doldurun', 'warning');
        return;
    }

    const result = await apiCall('/api/user/password', {
        method: 'POST',
        body: JSON.stringify({ old_password: oldPassword, new_password: newPassword })
    });

    if (result.success) {
        showToast('Şifre değiştirildi', 'success');
        closeModal();
    } else {
        showToast(result.error || 'Şifre değiştirilemedi', 'error');
    }
}

function closeModal() {
    const modal = document.getElementById('modalOverlay');
    if (modal) modal.classList.remove('active');
}

// Initialize
initTheme();
