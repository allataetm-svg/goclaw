---
name: skill-creator
description: Create or update Skills. Use when designing, structuring, or packaging skills with scripts, references, and assets.
triggers:
  - "create a skill"
  - "new skill"
  - "skill oluştur"
  - "yeni skill"
  - "make a skill"
  - "build a skill"
---

# Skill Creator

This skill provides guidance for creating effective skills for GoClaw.

## What are Skills?

Skills are modular, self-contained packages that extend the agent's capabilities by providing specialized knowledge, workflows, and tools. Think of them as "onboarding guides" for specific domains or tasks—they transform the agent from a general-purpose agent into a specialized agent equipped with procedural knowledge that no model can fully possess.

## Skill Structure

A skill is a directory containing:
- **SKILL.md** (required): Instructions and metadata
- **scripts/** (optional): Executable code the agent runs
- **references/** (optional): Documentation loaded only when needed
- **assets/** (optional): Static files

### SKILL.md Format

```yaml
---
name: my-skill
description: What the skill does and when to use it
version: 1.0.0
author: your-name
triggers:
  - "trigger phrase 1"
  - "trigger phrase 2"
tools:
  - tool_name
env:
  - ENV_VAR
bins:
  - command
---

# Skill Title

Describe what this skill does and when to use it.

## Instructions

1. First step
2. Second step
3. Third step

## Examples

- Example usage 1
- Example usage 2

## Notes

Any additional considerations.
```

## How to Create a Skill

### Step 1: Choose a Clear Scope

Define one narrow outcome:
- Too broad: "Handle all DevOps tasks"
- Good: "Generate a weekly incident summary from logs"

### Step 2: Create the Directory

Skills are stored in:
- Global: `~/.goclaw/skills/`
- Agent-specific: `~/.goclaw/agents/{agent_id}/skills/`

```bash
mkdir -p ~/.goclaw/skills/my-new-skill
```

### Step 3: Write SKILL.md

Follow the format above. Key points:
- **name**: lowercase with hyphens
- **description**: What AND when to use (for agent decision)
- **triggers**: Specific phrases that activate the skill

### Step 4: Test the Skill

1. Ask the agent to use your new skill
2. Verify it loads: check if skill name appears in system prompt
3. Refine instructions based on behavior

## Best Practices

1. **Description matters most**: The agent uses description to decide when to activate
2. **Be specific with triggers**: Avoid generic phrases
3. **Keep instructions clear**: Ordered steps the agent can follow
4. **Include guardrails**: Warn about risky operations
5. **Provide examples**: Show typical usage scenarios

## Environment Variables

Skills can require environment variables. Users configure these in `~/.goclaw/config.json`:

```json
{
  "skills": {
    "entries": {
      "my-skill": {
        "enabled": true,
        "env": {
          "API_KEY": "your-key-here"
        }
      }
    }
  }
}
```

## Tools Integration

Skills can specify required tools. The agent will only use the skill if those tools are available.

## Notes

- Skills use progressive loading: metadata always, full instructions when triggered
- Workspace-scoped skills override global ones
- Test locally before publishing
