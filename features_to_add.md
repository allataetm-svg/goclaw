# Features to Add

Bu dosya, GoClaw'a eklenecek yeni özellikleri listeler.

## Planlanan Özellikler

### 1. Model Fallback/Retry Sistemi ✅ (Eklendi)
Birincil model başarısız olduğunda fallback model'e geçiş yapabilme özelliği.

**Detaylar:**
- Provider config'te `retry` ve `fallbacks` alanları
- Agent config'te `fallbacks` alanı (provider:model format)
- Retry mekanizması: max_retries ve base_delay_ms (exponential backoff)
- FallbackProvider wrapper ile otomatik geçiş

**Kullanım:**
```json
// Provider config
{
  "id": "openai",
  "retry": { "max_retries": 3, "base_delay_ms": 1000 },
  "fallbacks": ["ollama:llama3"]
}

// Agent config
{
  "model": "openai:gpt-4o",
  "fallbacks": ["ollama:llama3", "openai:gpt-3.5-turbo"]
}
```

---

### 2. Cron/Scheduler Tool
Zamanlanmış görevler ve hatırlatıcılar oluşturma özelliği.

**Detaylar:**
- `cron` tool: add, list, remove, enable, disable aksiyonları
- `at_seconds`: tek seferlik hatırlatıcılar
- `every_seconds`: tekrarlayan görevler
- `cron_expr`: cron formatında zamanlama
- `command`: shell komutu çalıştırma

---

### 3. Heartbeat Service
Periyodik sağlık kontrolü ve otomatik prompt gönderme özelliği.

**Detaylar:**
- Yapılandırılabilir aralık (minimum 5 dakika)
- Belirli bir prompt'u periyodik olarak çalıştırma
- Sonuçları channel'a gönderme

---

### 4. Skills Sistemi
Modüler skill paketleri ile agent yeteneklerini genişletme.

**Detaylar:**
- Workspace/global/builtin skills desteği
- ClawHub registry entegrasyonu (ileride)
- Skill arama ve yükleme tool'u
- SKILL.md formatında skill tanımlama

---

### 5. Web Search ✅ (Eklendi)
LLM'in web araması yapabilmesi için tool desteği.

**Detaylar:**
- **DuckDuckGo**: Ücretsiz, varsayılan (HTML scraping + Lite API fallback)
- **Tavily**: AI için optimize edilmiş (API key gerekli)
- **Perplexity**: AI-powered arama (API key gerekli)
- Yapılandırılabilir max_results

**Kullanım:**
```
// Agent tool çağrısı
CALL: web_search({"query": "latest AI news 2026", "provider": "duckduckgo", "max_results": 5})

// Provider config (Tavily/Perplexity için)
{
  "providers": [
    { "id": "tavily", "api_key": "your-api-key" },
    { "id": "perplexity", "api_key": "your-api-key", "base_url": "https://api.perplexity.ai" }
  ]
}
```

---

### 6. Web Fetch
URL'lerden içerik çekme tool'u.

**Detaylar:**
- Verilen URL'den içerik alma
- HTML/Markdown parsing
- Boyut limitleri (aşırı büyük içerikleri kesme)

**Güvenlik - Prompt Injection Koruması:**
- URL'lerden gelen içeriklerde potansiyel prompt injection girişimleri tespit etme
- Şüpheli kalıplar: sistem komutları, tool çağrıları, rol atama girişimleri
- Karalisteye alınmış kalıplar: `IGNORE_PREVIOUS_INSTRUCTIONS`, `SYSTEM:`, `You are now`, `DAN` (Do Anything Now), vb.
- İçerik temizleme: şüpheli kalıpları kaldırma veya kullanıcıyı uyarma
- Yapılandırılabilir güvenlik seviyeleri (strict, moderate, off)

---

### 7. Browser Control
Tarayıcı kontrolü ile web otomasyonu.

**Detaylar:**
- Chrome/Chromium CDP (Chrome DevTools Protocol) entegrasyonu
- Web sayfalarını görüntüleme (snapshot)
- Form doldurma, tıklama gibi actions
- Dosya upload desteği
- Multiple profiles

---

### 8. Sessions Tools
Agentlar arası iletişim ve koordinasyon.

**Detaylar:**
- `sessions_list`: Aktif sessionları listeleme
- `sessions_history`: Session geçmişini getirme
- `sessions_send`: Başka bir session'a mesaj gönderme
- `sessions_spawn`: Yeni session oluşturma

---

### 9. Runtime/Security Özellikleri

**Session Pruning:**
- Uzun sessionlarda context sıkıştırma
- Otomatik özet oluşturma
- Token limit yönetimi

**OAuth Authentication:**
- OpenAI OAuth entegrasyonu
- Token yenileme
- Multiple auth profiles

**Presence & Typing Indicators:**
- Online/offline durumu
- Yazıyor göstergesi
- Channel'a özel yapılandırma

**Usage Tracking:**
- Token kullanımı takibi
- Maliyet hesaplama
- Raporlama

---

### 10. Control UI (Web Dashboard)
Gateway için web tabanlı yönetim arayüzü.

**Detaylar:**
- Agent yönetimi
- Channel durumu izleme
- Log görüntüleme
- Konfigürasyon düzenleme
- Health check

---

### 11. Workspace Secrets Sistemi
Agent'ın workspace dışına erişimini kontrol eden güvenlik sistemi.

**Detaylar:**
- `manage` menüsünde "Secrets" bölümü ekle
- Her agent için ayrı Workspace Key tanımlanabilir
- Agent workspace dışına erişmek istediğinde kullanıcıdan key istenir
- **Kural**: Agent asla key'i kendi söylemez - kullanıcı manuel olarak girer
- Key doğru ise session başına bir kez izin verilir
- Yanlış key denemeleri loglanır

**Kullanım Senaryosu:**
```
Kullanıcı: /home altındaki dosyaları listele
Agent: Bu workspace dışı bir konum. Erişim için Workspace Key'inizi girmeniz gerekiyor.
Kullanıcı: (key'i manuel girer)
Agent: (artık erişim sağlanır)
```

---

### 12. Memory Sistemi Geliştirme
Mevcut memory sisteminin iyileştirilmesi.

**Detaylar:**
- (Agent ile tartışılarak belirlenecek)
- Olası yönler:
  - SQLite destekli kalıcı hafıza
  - Vector embeddings ile semantic search
  - Session özetleme/compaction
  - Cross-session memory

---

## Not

**Subagent/Spawn Tool**: GoClaw'da `delegate_task` tool'u zaten mevcut. Bu özellik, agentların sub-agent çağırmasını sağlar. Ayrıca bir ekleme gerektirmez.

**Docker Sandbox**: OpenClaw'daki Docker sandbox özelliği eklenmeyecek (GoClaw'ın hafiflik hedefine aykırı).

---

# Silinecek Özellikler

## Eski Pairing Sistemi
Mevcut `manageSecurity` fonksiyonundaki "Pairing Code" alanı kaldırılacak.

**Değişiklikler:**
- `config.PairingCode` alanı silinecek
- `manage` menüsündeki Security bölümündeki "Pairing Code" input'u kaldırılacak
- Yeni sistem: Her kullanıcı için özel onay kodu üretilir (`goclaw pairing approve <channel> <userID> <code>`)
- Eski `/pair <code>` komutu yerine otomatik kod üretimi kullanılacak

## Not
Pairing sisteminin kendisi kalır - sadece eski "tek şifre" sistemi yerine kullanıcı başına kod sistemine geçildi.
