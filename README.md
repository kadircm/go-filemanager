# Go File Manager

Modern, web tabanlı dosya yöneticisi. Go ile yazılmış, tek binary olarak çalışır.

## Özellikler

- 📂 **Dosya Yönetimi** - Oluşturma, silme, taşıma, kopyalama, yeniden adlandırma
- ⬆️ **Dosya Yükleme** - Drag & drop desteği ile çoklu dosya yükleme
- 🗑️ **Çöp Kutusu** - Silinen dosyaları geri yükleme imkanı
- ✏️ **Kod Editörü** - CodeMirror ile syntax highlighting (20+ dil)
- 🎬 **Medya Oynatıcı** - HTTP 206 Partial Content ile video/audio streaming
- 🔍 **Arama** - Dosya/klasör araması ve filtreleme
- 👥 **Çoklu Kullanıcı** - Admin/user rolleri, kullanıcı bazlı erişim
- 🔒 **Güvenlik** - CSRF koruması, rate limiting, path traversal engelleme
- 📋 **Audit Log** - Tüm işlemlerin kaydı
- 🌙 **Dark/Light Tema** - Modern, responsive arayüz

## Hızlı Başlangıç

### Derleme

```bash
go build -o filemanager .
```

### Çalıştırma

```bash
./filemanager --port 8080 --root /var/www
```

### İlk Giriş

Varsayılan admin hesabı:
- Kullanıcı: `admin`
- Şifre: `admin123`

> ⚠️ İlk girişten sonra şifrenizi değiştirin!

## CLI Parametreleri

| Parametre | Varsayılan | Açıklama |
|-----------|-----------|----------|
| `--port` | 8080 | Sunucu portu |
| `--root` | / | Kök dosya dizini |
| `--admin-user` | admin | İlk admin kullanıcı adı |
| `--admin-pass` | admin123 | İlk admin şifresi |
| `--db` | ~/.filemanager/data.db | Veritabanı dosyası |
| `--trash-dir` | ~/.filemanager_trash | Çöp kutusu dizini |
| `--max-upload` | 1024 | Maks yükleme boyutu (MB) |
| `--rate-limit` | 60 | Dakika başı istek limiti |

## Teknolojiler

- **Backend:** Go, Fiber v2, SQLite (pure Go)
- **Frontend:** Vanilla CSS, JavaScript, CodeMirror
- **Güvenlik:** bcrypt, CSRF, rate limiting
- **Build:** Tek binary (go:embed ile tüm dosyalar gömülü)

## Ekran Görüntüleri

### Login Sayfası
Modern glassmorphism tasarımlı login ekranı.

### Dosya Yöneticisi
Liste ve grid görünüm, breadcrumb navigasyon, sağ tık context menü.

### Kod Editörü
Syntax highlighting, satır numaraları, arama/değiştirme.

## Lisans

MIT
