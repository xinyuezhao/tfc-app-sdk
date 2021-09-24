package main

import (
	"context"
	"fmt"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/manifoldco/promptui"
)

func main() {
	prompt := promptui.Prompt{
		Label: "Please enter your user token",
	}
	result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	config := &tfe.Config{
		Token: result,
	}

	client, err := tfe.NewClient(config)
	if err != nil {
		fmt.Printf("Authentication failed %v\n", err)
	}

	fmt.Printf("Query all organizations\n")

	// Create a context
	ctx := context.Background()

	// Query all organizations under current user and filter by entitlements
	orgs, err := queryAllOrgs(client, ctx)

	// Select an organization
	selector := promptui.Select{
		Label: "Select an organization",
		Items: orgs,
	}
	_, orgName, err := selector.Run()

	if err != nil {
		fmt.Printf(fmt.Sprintf("Prompt failed %v\n", err))
		return
	}

	fmt.Printf(fmt.Sprintf("Choose organization %v\n", orgName))

	fmt.Printf(fmt.Sprintf("Add a new agent for organization %v", orgName))

	var choosenAgentPl *tfe.AgentPool
	// Choose to use existing agentPools or creating a new agentPools
	selectAgentPool := promptui.Select{
		Label: "Add agent to an angentPool",
		Items: []string{"Choose from existing agentPools", "Create a new agentPool"},
	}
	_, agentPool, err := selectAgentPool.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	if agentPool == "Choose from existing agentPools" {
		agentPools, _ := queryAgentPools(client, ctx, orgName)
		var agentPoolsName []string
		for _, agentPool := range agentPools {
			agentPoolsName = append(agentPoolsName, agentPool.Name)
		}
		selectAgentName := promptui.Select{
			Label: "Choose from below agentPools",
			Items: agentPoolsName,
		}
		_, agentName, err := selectAgentName.Run()
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}
		// query agentPool
		agentPl, err := queryAgentPool(agentPools, agentName)
		if err != nil {
			fmt.Printf(fmt.Sprintf("Query agentPool %v failed\n", agentName))
		}
		fmt.Println(fmt.Sprintf("Choose agentPool %v", agentName))
		choosenAgentPl = agentPl
	} else {
		// enter name for new agentPool
		prompt := promptui.Prompt{
			Label: "Enter name for new agentPool",
		}
		agentName, err := prompt.Run()

		if err != nil {
			fmt.Printf(fmt.Sprintf("Prompt failed %v\n", err))
		}
		agentPl, err := createAgentPool(client, ctx, orgName, agentName)
		if err != nil {
			fmt.Printf(fmt.Sprintf("Creating agentPool failed %v\n", err))
		}
		fmt.Println(fmt.Sprintf("New agentPool %v created", agentPl.Name))
		choosenAgentPl = agentPl
	}

	// Choose to create a new agentToken or deleting an existing agentToken or query all existing agentTokens
	selectOption := promptui.Select{
		Label: fmt.Sprintf("Choose option for agentPool %v", choosenAgentPl.Name),
		Items: []string{"Create a new agentToken", "Delete an existing agentToken", "Query all existing agentTokens"},
	}
	_, option, err := selectOption.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	if option == "Create a new agentToken" {
		fmt.Println(fmt.Sprintf("Adding a new agent token in agentpool %v", choosenAgentPl.Name))

		// Enter description for created agentToken
		descPrompt := promptui.Prompt{
			Label: "Enter description for new agent token",
		}
		desc, err := descPrompt.Run()
		if err != nil {
			fmt.Printf(fmt.Sprintf("Prompt failed %v\n", err))
		}

		// Create a new AgentToken in choosed AgentPool
		agentToken, err := createAgentToken(client, ctx, choosenAgentPl, desc)
		fmt.Println(fmt.Sprintf("New agent token %v created", agentToken.Description))
	} else {
		// query all agentTokens in choosed AgentPool
		agentTokens, err := queryAgentTokens(client, ctx, choosenAgentPl)
		if err != nil {
			fmt.Printf(fmt.Sprintf("Query agentTokens failed %v\n", err))
		}
		// var agentTokensDesc []string
		// for _, agentToken := range agentTokens {
		// 	agentTokensDesc = append(agentTokensDesc, agentToken.Description)
		// }

		if option == "Delete an existing agentToken" {
			fmt.Println(fmt.Sprintf("Deleting an existing agent token in agentpool %v", choosenAgentPl.Name))
			templates := &promptui.SelectTemplates{
				Label:    "{{ . }}?",
				Active:   "> AgentToken description: {{ .Description | cyan }}, ID: {{ .ID | red }}",
				Inactive: "AgentToken description: {{ .Description }}, ID: {{ .ID }}",
			}
			selectToken := promptui.Select{
				Label:     fmt.Sprintf("Choose option for agentPool %v", choosenAgentPl.Name),
				Templates: templates,
				Items:     agentTokens,
			}
			i, _, err := selectToken.Run()
			if err != nil {
				fmt.Printf("Prompt failed %v\n", err)
				return
			}
			removeAgentToken(client, ctx, agentTokens[i])
			// check whether the AgentPool is empty, delete the AgentPool if yes
			tokensAfterDelete, err := queryAgentTokens(client, ctx, choosenAgentPl)
			if len(tokensAfterDelete) == 0 {
				fmt.Println(fmt.Sprintf("Deleting AgentPool %v as it is empty", choosenAgentPl.Name))
				removeAgentPool(client, ctx, choosenAgentPl)
			}
		}
	}

	// Query existing Agent Tokens for choosed AgentPool
	agentTokens, err := queryAgentTokens(client, ctx, choosenAgentPl)
	fmt.Println(fmt.Sprintf("List all agent tokens in agent pool %v", choosenAgentPl.Name))
	for _, agentToken := range agentTokens {
		fmt.Println(fmt.Sprintf("AgentToken description %v", agentToken.Description))
	}
}

// Query all orgs' name under current user
func queryAllOrgs(client *tfe.Client, ctx context.Context) ([]string, error) {
	var res []string
	orgs, err := client.Organizations.List(ctx, tfe.OrganizationListOptions{})
	if err != nil {
		return nil, err
	}
	// filter orgs by entitlement
	for _, element := range orgs.Items {
		// fmt.Println(element.Name)
		entitlements, err := client.Organizations.Entitlements(ctx, element.Name)
		if err != nil {
			return nil, err
		}
		if entitlements.Agents {
			res = append(res, element.Name)
		}
		// fmt.Println(fmt.Sprintf("Entitlement of organization %v is %v", element.Name, entitlements.Agents))
	}
	return res, err
}

// Query all agentPools for an organization
func queryAgentPools(client *tfe.Client, ctx context.Context, name string) ([]*tfe.AgentPool, error) {
	agentPools, err := client.AgentPools.List(ctx, name, tfe.AgentPoolListOptions{})
	if err != nil {
		return nil, err
	}
	res := agentPools.Items
	return res, nil
}

// Create a new agentPool for an organization
func createAgentPool(client *tfe.Client, ctx context.Context, orgName string, agentPlName string) (*tfe.AgentPool, error) {
	createOptions := tfe.AgentPoolCreateOptions{Name: &agentPlName}
	agentPl, err := client.AgentPools.Create(ctx, orgName, createOptions)
	if err != nil {
		return nil, err
	}
	return agentPl, nil
}

// Query agentPool by the name
func queryAgentPool(agentPools []*tfe.AgentPool, name string) (*tfe.AgentPool, error) {
	for _, agentPl := range agentPools {
		if agentPl.Name == name {
			return agentPl, nil
		}
	}
	return nil, fmt.Errorf(fmt.Sprintf("There is no agentPool named %v", name))
}

// Query AgentTokens in an agentPool
func queryAgentTokens(client *tfe.Client, ctx context.Context, agentPl *tfe.AgentPool) ([]*tfe.AgentToken, error) {
	agentTokens, err := client.AgentTokens.List(ctx, agentPl.ID)
	if err != nil {
		fmt.Printf("Query agentTokens failed\n")
		return nil, err
	}
	res := agentTokens.Items
	return res, nil
}

// Create a new agentToken in an agentPool
func createAgentToken(client *tfe.Client, ctx context.Context, agentPl *tfe.AgentPool, desc string) (*tfe.AgentToken, error) {
	agentToken, err := client.AgentTokens.Generate(ctx, agentPl.ID, tfe.AgentTokenGenerateOptions{Description: &desc})
	if err != nil {
		return nil, err
	}
	fmt.Println(fmt.Sprintf("agent token created: %v", agentToken.Token))
	return agentToken, nil
}

// Delete an existing agentToken
func removeAgentToken(client *tfe.Client, ctx context.Context, agentToken *tfe.AgentToken) error {
	err := client.AgentTokens.Delete(ctx, agentToken.ID)
	if err != nil {
		return err
	}
	fmt.Println(fmt.Sprintf("AgentToken %v was deleted", agentToken.Description))
	return nil
}

// Delete an existing agentPool
func removeAgentPool(client *tfe.Client, ctx context.Context, agentPl *tfe.AgentPool) error {
	err := client.AgentPools.Delete(ctx, agentPl.ID)
	if err != nil {
		return err
	}
	fmt.Println(fmt.Sprintf("AgentPool %v was deleted", agentPl.Name))
	return nil
}
