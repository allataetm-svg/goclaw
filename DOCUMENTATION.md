# 🦞 GoClaw - Documentation / Dokümantasyon

[English](#english) | [Türkçe](#türkçe)

---

## English

### 🌟 Project Overview
GoClaw is a lightweight, local-first personal AI assistant gateway built in Go. It enables users to interact with various AI models (via providers like OpenAI, Anthropic, Gemini, Ollama, OpenRouter, Mistral) through multiple communication channels (Telegram, Terminal/CLI). Central to its design is the concept of **Agents**, where each agent can have its own system prompt, configuration, and set of tools.

### 🏗️ System Architecture
The project is organized into modular packages under the `internal` directory:

| Package | Responsibility |
| :--- | :--- |
| `main.go` | Main entry point handles command-line arguments and modes. |
| `internal/agent` | Agent loading, system prompt generation, and tool definitions (12 tools). |
| `internal/channel` | Communication layers (Telegram, Console) and the **Router** with message processing. |
| `internal/provider` | LLM provider interface (Gemini, Anthropic, OpenAI, Ollama, OpenRouter, Mistral). |
| `internal/config` | Configuration management and persistence. |
| `internal/manage` | TUI-based management dashboard for agents, channels, and security. |
| `internal/onboard` | Initialization wizard for first-time setup. |
| `internal/tui` | Terminal User Interface for direct chatting. |
| `internal/memory` | Enhanced memory system: Ephemeral, Long-term (User & Knowledge), Context Compaction. |
| `internal/scheduler` | Cron/interval/one-shot task scheduling with shell execution. |
| `internal/sessions` | Session tracking and agent-to-agent communication. |
| `internal/skills` | Modular skill packages (builtin/global/workspace). |
| `internal/secrets` | Workspace key-based access control for agents. |
| `internal/heartbeat` | Periodic health check and prompt execution service. |

---

### 🔄 Message Processing Flow
When a user sends a message (e.g., via Telegram), the system follows this sequence:

1.  **Channel Input**: The specific channel (e.g., `TelegramChannel`) receives the event and wraps it in a standard `Message` struct.
2.  **Router Gateway**: The `Router.HandleIncoming` function is triggered.
3.  **Access Control**: Pairing check for unauthorized users (6-digit code, max 3 attempts).
4.  **Command Interception**: Checks for chat commands (`/status`, `/new`, `/compact`, `/usage`, `/agent`, `/help`, etc.).
5.  **Agent Selection**: Retrieves the active agent ID for the session/chat.
6.  **Memory Integration**: User memory is loaded and injected into the context.
7.  **Context Preparation**: History is retrieved, auto-compacted if approaching token limits.
8.  **AI Query**: The request is sent to the configured AI Provider.
9.  **Tool Execution**: Up to 10 tool iterations with results fed back as context.
10. **Async & Interruption**: New messages cancel in-progress tasks immediately.
11. **Reply & Usage**: Response sent via channel, optional usage footer appended.

---

### 🚀 Usage Modes
*   `onboard` — Setup wizard to configure providers and agents.
*   `tui` — Terminal User Interface (Chat).
*   `cli` — Command Line Interface (no TTY required).
*   `gateway` — Multi-channel gateway (Telegram, Console) with scheduler.
*   `manage` — Interactive agent/channel/security management dashboard.
*   `pairing` — User authorization commands.
*   `help` — Show help message.

---

### 💬 Chat Commands (Gateway)
| Command | Description |
| :--- | :--- |
| `/help` | Show available commands |
| `/status` | Session status — agent name, model, token usage |
| `/new` / `/reset` | Reset session (clear history) |
| `/clear` | Clear chat history |
| `/compact` | Manually compact context (summarize old messages) |
| `/usage off\|tokens\|full` | Toggle per-response usage footer |
| `/tokens` | Show token usage estimate |
| `/history` | Show message history |
| `/agent list` | List installed agents |
| `/agent switch <id>` | Switch to a different agent |

---

### 🛠️ Tools (12 Available)
| Tool | Description |
| :--- | :--- |
| `delegate_task` | Delegate tasks to subagents |
| `read_file` | Read file contents |
| `write_file` | Write content to files |
| `shell` | Execute shell commands |
| `reply` | Send messages to users |
| `web_search` | Web search (DuckDuckGo, Tavily, Perplexity) |
| `web_fetch` | Fetch URL content with prompt injection protection |
| `scheduler` | Manage cron/interval/one-shot scheduled tasks |
| `heartbeat` | Periodic health check prompts |
| `skills` | Manage skill packages |
| `sessions` | Session management and agent-to-agent communication |
| `secrets` | Workspace access control |

---

### 🛡️ Security & Pairing
GoClaw includes a pairing system to prevent unauthorized access:
1.  **Enable Pairing**: Via `onboard` or `manage` -> Security.
2.  **Request Access**: Unknown user messages trigger a 6-digit code (max 3 attempts).
3.  **Owner Approval**: Run `goclaw pairing approve <channel> <userID> <code>`.
4.  **Persistent Access**: Approved users are added to whitelist.

---

### 🧠 Memory System
1.  **Ephemeral Memory**: Short-term conversation history in current session.
2.  **Long-term Memory (User Store)**: Persistent user preferences and facts.
3.  **Knowledge Store**: RAG-style document storage for agents.
4.  **Context Compaction**: Auto-summarizes/trims when token limits approached.
5.  **Gateway Integration**: Memory is automatically injected into agent context.

---

---

## Türkçe

### 🌟 Proje Genel Bakış
GoClaw, Go diliyle geliştirilmiş, hafif ve "yerel-öncelikli" bir kişisel yapay zeka asistan ağ geçididir. Kullanıcıların farklı iletişim kanalları (Telegram, Terminal) üzerinden çeşitli yapay zeka modelleriyle (OpenAI, Anthropic, Gemini, Ollama, OpenRouter, Mistral) etkileşime girmesini sağlar.

### 🏗️ Sistem Mimarisi
Proje, `internal` dizini altında 13 modüler paketten oluşur:

| Paket | Görev |
| :--- | :--- |
| `main.go` | Ana giriş noktası; komut satırı argümanlarını yönetir. |
| `internal/agent` | Ajan yükleme, sistem prompt oluşturma ve 12 araç (tool) tanımı. |
| `internal/channel` | İletişim katmanları ve Router ile mesaj işleme. |
| `internal/provider` | Yapay zeka sağlayıcı arayüzü (6 sağlayıcı). |
| `internal/config` | Yapılandırma yönetimi. |
| `internal/manage` | TUI tabanlı yönetim paneli (ajan, kanal, güvenlik). |
| `internal/memory` | Gelişmiş bellek sistemi ve bağlam sıkıştırma. |
| `internal/scheduler` | Zamanlanmış görev yönetimi (cron/aralık/tek seferlik). |
| `internal/sessions` | Oturum takibi ve ajanlar arası iletişim. |
| `internal/skills` | Modüler yetenek paketleri. |
| `internal/secrets` | Çalışma alanı erişim kontrolü. |
| `internal/heartbeat` | Periyodik sağlık kontrolü servisi. |

---

### 💬 Sohbet Komutları (Gateway)
| Komut | Açıklama |
| :--- | :--- |
| `/help` | Komutları göster |
| `/status` | Oturum durumu — ajan, model, token kullanımı |
| `/new` / `/reset` | Oturumu sıfırla |
| `/clear` | Sohbet geçmişini temizle |
| `/compact` | Bağlamı sıkıştır |
| `/usage off\|tokens\|full` | Kullanım bilgisi göster/gizle |
| `/tokens` | Token tahmini |
| `/agent list` | Ajanları listele |
| `/agent switch <id>` | Ajan değiştir |

---

### 🛡️ Güvenlik ve Eşleştirme (Pairing)
1.  **Etkinleştirme**: `onboard` veya `manage` -> Security.
2.  **Erişim Talebi**: Tanınmayan kullanıcıya 6 haneli kod üretilir (max 3 deneme).
3.  **Sahip Onayı**: `goclaw pairing approve <kanal> <kullanıcıID> <kod>`
4.  **Kalıcı Yetki**: Onaylanan kullanıcı beyaz listeye eklenir.

---

### 🧠 Gelişmiş Bellek Sistemi
1.  **Kısa Süreli Bellek**: Mevcut oturumdaki sohbet geçmişi.
2.  **Uzun Süreli Bellek**: Kullanıcı tercihleri ve bilgileri.
3.  **Bilgi Deposu**: RAG tarzı doküman deposu.
4.  **Bağlam Sıkıştırma**: Token limitleri yaklaşınca otomatik özet/kırpma.
5.  **Gateway Entegrasyonu**: Bellek otomatik olarak ajan bağlamına enjekte edilir.
