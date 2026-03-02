# Enhanced Memory System

Goclaw now implements a 4-layer memory architecture inspired by OpenClaw and OpenFang.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│ Layer 1: EPHEMERAL (Runtime)                          │
│ - Current conversation in prompt window                │
│ - Auto-trim based on token limit                       │
│ - Fast access, volatile                                │
├─────────────────────────────────────────────────────────┤
│ Layer 2: SESSION (Task-based)                         │
│ - Current task/session state                           │
│ - Goals, entities, steps                                │
│ - Cleared on session end                               │
├─────────────────────────────────────────────────────────┤
│ Layer 3: USER (Long-term)                              │
│ - User preferences, facts                              │
│ - Stored in memories.json                             │
│ - Persistent, cross-session                            │
├─────────────────────────────────────────────────────────┤
│ Layer 4: KNOWLEDGE (Documents)                         │
│ - External knowledge, documents                       │
│ - Keyword-based indexing                               │
│ - Semantic search capability                           │
└─────────────────────────────────────────────────────────┘
```

## Directory Structure

```
~/.goclaw/
├── agents/
│   └── {agent_id}/
│       ├── config.json          # Agent config
│       ├── SOUL.md              # Personality
│       ├── AGENT.md             # Mission/tasks
│       ├── INSTRUCTIONS.md      # Operational rules (NEW)
│       └── capabilities.yaml    # Capabilities (NEW)
├── memory/
│   ├── longterm/
│   │   ├── {agent_id}.json      # User memories
│   │   └── knowledge/
│   │       └── {agent_id}.json  # Knowledge documents
│   └── sessions/
│       └── session_*.json       # Session histories
└── history/
    └── {conversation}.json     # Conversation archives
```

## TUI Commands

### Memory Commands

```bash
/memory store <key> <value>   # Store user memory
/memory recall <query>        # Search memories
/memory list                 # List all memories
/memory delete <id>          # Delete a memory
```

### Knowledge Commands

```bash
/knowledge add <content>      # Add knowledge document
/knowledge search <query>      # Search knowledge
/knowledge list               # List documents
```

### History Commands (existing)

```bash
/history list                 # List conversations
/history load <id>           # Load conversation
/history delete <id>          # Delete conversation
```

## Example Usage

```
> /memory store prefers_dark_mode true
Memory stored: prefers_dark_mode = true

> /memory recall theme
Found Memories:
  [preference] prefers_dark_mode: true

> /knowledge add Python best practices: Use virtual environments
Knowledge document added.

> /knowledge search virtual
Found Knowledge:
  [manual] Python best practices: Use virtual environments
```

## Integration Points

### Agent Workspace Files

- **SOUL.md**: Agent personality and character
- **AGENT.md**: Primary mission and tasks
- **INSTRUCTIONS.md**: Operational rules (NEW)
- **capabilities.yaml**: Tool/skill definitions (NEW)

### Memory API

```go
// Create new memory manager
mem := memory.NewMemory(agentID, systemPrompt, maxTokens)
mem.Initialize()

// Store user preference
mem.StoreUserMemory(memory.MemoryTypePreference, "key", "value", "source", nil)

// Search memories
results := mem.SearchUserMemory("query")

// Add knowledge
mem.AddKnowledgeDocument(content, source, "", nil)
```

## Credits

Inspired by:
- OpenClaw's file-based memory and hybrid retrieval
- OpenFang's production-grade architecture
- NanoBot/PicoClaw's minimalism and speed
