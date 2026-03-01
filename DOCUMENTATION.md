# 🦞 GoClaw - Documentation / Dokümantasyon

[English](#english) | [Türkçe](#türkçe)

---

## English

### 🌟 Project Overview
GoClaw is a lightweight, local-first personal AI assistant gateway built in Go. It enables users to interact with various AI models (via providers like OpenAI, Anthropic, Gemini, Ollama) through multiple communication channels (Telegram, Terminal/CLI). Central to its design is the concept of **Agents**, where each agent can have its own system prompt, configuration, and set of tools.

### 🏗️ System Architecture
The project is organized into modular packages under the `internal` directory:

| Package | Responsibility |
| :--- | :--- |
| `cmd/` | Main entry point handles command-line arguments and modes. |
| `internal/agent` | Logic for agent loading, system prompt generation, and tool definitions. |
| `internal/channel` | Communication layers (Telegram, Console) and the **Router** that orchestrates message flows. |
| `internal/provider` | Interface for AI service providers (Gemini, Anthropic, OpenAI, etc.). |
| `internal/config` | Configuration management and persistence. |
| `internal/manage` | Logic for the TUI-based management dashboard. |
| `internal/onboard` | Initialization wizard for first-time setup. |
| `internal/tui` | Terminal User Interface for direct chatting. |
| `internal/memory` | Conversation history management. |

---

### 🔄 Message Processing Flow
When a user sends a message (e.g., via Telegram), the system follows this sequence:

1.  **Channel Input**: The specific channel (e.g., `TelegramChannel`) receives the event and wraps it in a standard `Message` struct.
2.  **Router Gateway**: The `Router.HandleIncoming` function is triggered.
3.  **Command Interception**:
    *   The system checks if the message is a **routing command** (e.g., `/agent list`, `/agent switch`).
    *   If matched, the command is executed immediately (e.g., changing the active agent for that user), and a reply is sent directly without querying the AI.
4.  **Agent Selection**:
    *   If not a command, the Router retrieves the active agent ID for the session/chat.
    *   The Agent's workspace, provider constraints, and model details are loaded.
5.  **Context Preparation**:
    *   The conversation history (memory) is retrieved.
    *   A System Prompt is injected if it's a new session or after a specific turn count to maintain context.
6.  **AI Query**: The request is sent to the configured AI Provider (e.g., Gemini).
8.  **Tool Execution**:
    *   If a tool call is found and the agent has permissions, the tool (e.g., `read_file`, `shell`) is executed.
    *   The tool's output is appended to the history.
    *   **Multi-Turn Loop**: The Router continues to query the AI with tool results (up to 5 turns) to generate intermediate replies or a final response.
9.  **Async & Interruption**:
    *   Processing runs asynchronously.
    *   If a user sends a new message while the agent is still "thinking" or running a tool, the previous task is **cancelled** immediately to prioritize the new instruction.
    *   Agents can use the `reply` tool to send multiple messages during a single turn.
10. **Delivery**: The final response or intermediate updates are sent back to the user via the original channel.

---

### 🚀 Usage Modes
*   `onboard`: Run this first to set up providers and your first agent.
*   `tui`: Chat with your default agent directly in the terminal.
*   `gateway`: Start the background process that listens for Telegram and Console messages.
*   `manage`: Access the dashboard to create, edit, or delete agents and channels.

---

## Türkçe

### 🌟 Proje Genel Bakış
GoClaw, Go diliyle geliştirilmiş, hafif ve "yerel-öncelikli" bir kişisel yapay zeka asistan ağ geçididir. Kullanıcıların farklı iletişim kanalları (Telegram, Terminal) üzerinden çeşitli yapay zeka modelleriyle (OpenAI, Anthropic, Gemini, Ollama vb.) etkileşime girmesini sağlar. Projenin merkezinde, her biri kendine has sistem komutuna, yapılandırmasına ve araç setine sahip olabilen **Ajanlar (Agents)** kavramı yer alır.

### 🏗️ Sistem Mimarisi
Proje, `internal` dizini altındaki modüler paketlerden oluşur:

| Paket | Görev |
| :--- | :--- |
| `main.go` | Ana giriş noktası; komut satırı argümanlarını ve çalışma modlarını yönetir. |
| `internal/agent` | Ajan yükleme, sistem mesajı (prompt) oluşturma ve araç (tool) tanımlama mantığı. |
| `internal/channel` | İletişim katmanları (Telegram, Konsol) ve mesaj akışını yöneten **Router (Yönlendirici)**. |
| `internal/provider` | Yapay zeka servis sağlayıcıları arayüzü (Gemini, Anthropic, OpenAI vb.). |
| `internal/config` | Yapılandırma yönetimi ve kalıcılık. |
| `internal/manage` | TUI tabanlı yönetim paneli mantığı. |
| `internal/onboard` | İlk kurulum sihirbazı. |
| `internal/tui` | Doğrudan sohbet için Terminal Kullanıcı Arayüzü. |
| `internal/memory` | Sohbet geçmişi (bellek) yönetimi. |

---

### 🔄 Mesaj İşleme Akışı
Bir kullanıcı mesaj gönderdiğinde (örneğin Telegram üzerinden), sistem şu adımları izler:

1.  **Kanal Girişi**: İlgili kanal (örneğin `TelegramChannel`) olayı alır ve standart bir `Message` yapısına dönüştürür.
2.  **Yönlendirici (Router)**: `Router.HandleIncoming` fonksiyonu tetiklenir.
3.  **Komut Kontrolü**:
    *   Sistem, mesajın bir **yönlendirme komutu** (Örn: `/agent list`, `/agent switch`) olup olmadığını kontrol eder.
    *   Eğer bir komut ise, yapay zekaya sorulmadan doğrudan işlem yapılır (aktif ajanı değiştirme vb.) ve kullanıcıya cevap verilir.
4.  **Ajan Seçimi**:
    *   Mesaj bir komut değilse, Router kullanıcının aktif seansındaki ajan ID'sini belirler.
    *   Ajanın çalışma alanı, sağlayıcı kısıtlamaları ve model detayları yüklenir.
5.  **Bağlam Hazırlığı**:
    *   Sohbet geçmişi (bellek) belleğe alınır.
    *   Eğer seans yeniyse veya belirli bir mesaj sayısına ulaşıldıysa, bağlamı korumak için Sistem Komutu (System Prompt) geçmişe eklenir.
6.  **Yapay Zeka Sorgusu**: İstek, yapılandırılmış sağlayıcıya (örn: Gemini) gönderilir.
10. **Asenkron İşleme ve İptal Edilebilirlik**:
    *   Mesaj işleme süreci arka planda (asenkron) yürütülür.
    *   Eğer ajan işlem yaparken (düşünürken veya bir araç çalıştırırken) kullanıcı yeni bir mesaj gönderirse, devam eden işlem **iptal edilir** ve yeni talimata öncelik verilir.
    *   Ajanlar `reply` aracını kullanarak tek bir adımda birden fazla mesaj gönderebilirler.
11. **Teslimat**: Nihai cevap veya ara güncellemeler, orijinal kanal üzerinden kullanıcıya iletilir.

---

### 🚀 Çalışma Modları
*   `onboard`: Sağlayıcıları ve ilk ajanınızı kurmak için ilk kez çalıştırın.
*   `tui`: Terminal üzerinden varsayılan ajanınızla doğrudan sohbet edin.
*   `gateway`: Telegram ve Konsol mesajlarını dinleyen arka plan sürecini başlatın.
*   `manage`: Ajanları ve kanalları oluşturmak veya düzenlemek için yönetim panelini açın.
