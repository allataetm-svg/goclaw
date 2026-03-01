package manage

import (
	"fmt"

	"github.com/allataetm-svg/goclaw/internal/agent"
	"github.com/allataetm-svg/goclaw/internal/config"
	"github.com/charmbracelet/huh"
)

// Run starts the interactive agent management interface
func Run() {
	for {
		var action string
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("🦞 Agent Management Dashboard").
					Options(
						huh.NewOption("List Agents", "list"),
						huh.NewOption("Add New Agent (Main/Sub)", "add"),
						huh.NewOption("Manage Channels", "channels"),
						huh.NewOption("Exit", "exit"),
					).
					Value(&action),
			),
		).Run()

		if err != nil || action == "exit" {
			return
		}

		switch action {
		case "list":
			listAndManageAgents()
		case "add":
			addNewAgentWizard()
		case "channels":
			manageChannels()
		}
	}
}

func listAndManageAgents() {
	agents, err := agent.ListAgents()
	if err != nil {
		fmt.Printf("Error listing agents: %v\n", err)
		return
	}

	if len(agents) == 0 {
		fmt.Println("No agents found.")
		return
	}

	options := make([]huh.Option[string], 0, len(agents)+1)
	for _, a := range agents {
		options = append(options, huh.NewOption(fmt.Sprintf("%s (%s [%s])", a.Name, a.ID, a.Type), a.ID))
	}
	options = append(options, huh.NewOption("<- Back", "back"))

	var selectedID string
	huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select an Agent to manage:").
				Options(options...).
				Value(&selectedID),
		),
	).Run()

	if selectedID == "" || selectedID == "back" {
		return
	}

	manageSingleAgent(selectedID)
}

func manageSingleAgent(id string) {
	ws, err := agent.LoadAgentWorkspace(id)
	if err != nil {
		fmt.Printf("Error loading agent: %v\n", err)
		return
	}

	for {
		var action string
		huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(fmt.Sprintf("Managing Agent: %s", ws.Config.Name)).
					Options(
						huh.NewOption("Edit Soul (SOUL.md)", "soul"),
						huh.NewOption("Edit Mission (AGENT.md)", "agent"),
						huh.NewOption("Change Model", "model"),
						huh.NewOption("Delete Agent", "delete"),
						huh.NewOption("<- Back", "back"),
					).
					Value(&action),
			),
		).Run()

		if action == "back" || action == "" {
			return
		}

		switch action {
		case "soul":
			var newSoul = ws.Soul
			huh.NewForm(
				huh.NewGroup(
					huh.NewText().
						Title("Edit SOUL.md").
						Value(&newSoul),
				),
			).Run()
			if newSoul != ws.Soul {
				ws.Soul = newSoul
				agent.SaveAgentWorkspace(ws)
			}
		case "agent":
			var newMission = ws.Agent
			huh.NewForm(
				huh.NewGroup(
					huh.NewText().
						Title("Edit AGENT.md").
						Value(&newMission),
				),
			).Run()
			if newMission != ws.Agent {
				ws.Agent = newMission
				agent.SaveAgentWorkspace(ws)
			}
		case "model":
			// We need allModels here too, but to keep it simple for now...
			var newModel string
			huh.NewForm(huh.NewGroup(huh.NewInput().Title("New Model Name (provider:model):").Value(&newModel))).Run()
			if newModel != "" {
				ws.Config.Model = newModel
				agent.SaveAgentWorkspace(ws)
			}
		case "delete":
			var confirm bool
			huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("Are you sure you want to delete this agent?").Value(&confirm))).Run()
			if confirm {
				agent.DeleteAgentWorkspace(id)
				return
			}
		}
	}
}

func addNewAgentWizard() {
	var name string
	var model string
	var aType string

	huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Agent Name:").Value(&name),
			huh.NewInput().Title("Model (provider:model):").Value(&model),
			huh.NewSelect[string]().
				Title("Agent Type:").
				Options(huh.NewOption("Main Agent", "main"), huh.NewOption("Subagent", "sub")).
				Value(&aType),
		),
	).Run()

	if name != "" && model != "" {
		agent.AddAgent(name, model, agent.AgentType(aType))
		fmt.Println("🚀 Agent added successfully!")
	}
}

func manageChannels() {
	conf, _ := config.Load()

	options := make([]huh.Option[string], 0, len(conf.Channels)+1)
	for i, c := range conf.Channels {
		options = append(options, huh.NewOption(fmt.Sprintf("%s (%s)", c.Name, c.Type), fmt.Sprintf("%d", i)))
	}
	options = append(options, huh.NewOption("Add New Channel", "add"))
	options = append(options, huh.NewOption("<- Back", "back"))

	var selectIdx string
	huh.NewForm(huh.NewGroup(huh.NewSelect[string]().Title("Manage Channels").Options(options...).Value(&selectIdx))).Run()

	if selectIdx == "back" || selectIdx == "" {
		return
	}

	if selectIdx == "add" {
		// Just a placeholder for now to keep it concise
		fmt.Println("To add a new channel, please use the 'onboard' wizard for now.")
	} else {
		// Manage existing channel (delete)
		var confirm bool
		huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("Do you want to delete this channel?").Value(&confirm))).Run()
		if confirm {
			var idx int
			fmt.Sscanf(selectIdx, "%d", &idx)
			conf.Channels = append(conf.Channels[:idx], conf.Channels[idx+1:]...)
			config.Save(conf)
		}
	}
}
