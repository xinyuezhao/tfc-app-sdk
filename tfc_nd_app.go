package main

import (
	"context"
	"fmt"
	"log"

	tfe "github.com/hashicorp/go-tfe"
)

func main() {
	config := &tfe.Config{
		Token: "ai1yMKOzv3Mptg.atlasv1.lOseEHJzlB49Vz0fXTlFUFRGGTuugiP3040sr1MGGOkHgRqzQ9FrpiUJzyTH1DzzFTM",
	}

	client, err := tfe.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	// Create a context
	ctx := context.Background()

	// Query all organizations
	orgs, err := client.Organizations.List(ctx, tfe.OrganizationListOptions{})
	if err != nil {
		log.Fatal(err)
	}

	// Query entitlement for each organization
	for _, element := range orgs.Items {
		fmt.Println(element.Name)
		entitlements, err := client.Organizations.Entitlements(ctx, element.Name)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(fmt.Sprintf("Entitlement of organization %v is %v", element.Name, entitlements.Agents))
	}
	// Query all agent pools for cisco-dcn-ecosystem
	ecosystemOrg := orgs.Items[0]
	agentPools, err := client.AgentPools.List(ctx, ecosystemOrg.Name, tfe.AgentPoolListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	for _, agentPool := range agentPools.Items {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(fmt.Sprintf("AgentPool name is: %v, ID is %v", agentPool.Name, agentPool.ID))
	}

	// Create a new agentpool in cisco-dcn-ecosystem
	agentName := "tfc_nd_test"
	createOptions := tfe.AgentPoolCreateOptions{Name: &agentName}
	agentPl, err := client.AgentPools.Create(ctx, ecosystemOrg.Name, createOptions)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(fmt.Sprintf("AgentPool %v created", agentPl.Name))

	// Query existing Agent Tokens for AgentPool 'ams-lab'
	agentPlAmsLabID := agentPools.Items[2].ID
	// fmt.Println(agentPools.Items[2].Name)
	agentTokens, err := client.AgentTokens.List(ctx, agentPlAmsLabID)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println(agentTokens)
	for _, agentToken := range agentTokens.Items {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(fmt.Sprintf("AgentToken ID is %v, token is %v, created at %v", agentToken.ID, agentToken.Token, agentToken.CreatedAt))
	}

	// Create new Agent Token in AgentPool 'tfc_nd_test'
	desc := "New AgentToken"
	agentTokenNew, err := client.AgentTokens.Generate(ctx, agentPl.ID, tfe.AgentTokenGenerateOptions{Description: &desc})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(fmt.Sprintf("AgentToken %v created at %v with description %v", agentTokenNew.ID, agentTokenNew.CreatedAt, agentTokenNew.Description))
}
