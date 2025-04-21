package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type ProjectItem struct {
	ID           string
	Title        string
	URL          string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DueDate      string
	AssignedTo   []string
	Labels       []string
	Description  string
	Recipient    string
	BountyAmount string
	BountySymbol string
}

func main() {
	// Get GitHub token from environment variable
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("GitHub token not found. Set the GITHUB_TOKEN environment variable.")
	}

	// Create GitHub client
	ctx := context.Background()
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(ctx, src)
	client := githubv4.NewClient(httpClient)

	// Project details
	org := "NautilusOSS"
	projectNumber := 2

	// Get project ID
	projectID, err := getProjectID(ctx, client, org, projectNumber)
	if err != nil {
		log.Fatalf("Error getting project ID: %v", err)
	}
	fmt.Printf("Project ID: %s\n", projectID)

	// Get project items
	items, err := getProjectItems(ctx, client, projectID)
	if err != nil {
		log.Fatalf("Error getting project items: %v", err)
	}
	fmt.Printf("Found %d 'Pending Payment' items in the project\n", len(items))

	// Generate CSV file
	if err := generateCSV(items, "pending_payment_tasks.csv"); err != nil {
		log.Fatalf("Error generating CSV: %v", err)
	}
	fmt.Println("CSV file generated: pending_payment_tasks.csv")

	// Generate summary report
	if err := generateSummaryReport(items, "pending_payment_summary.txt"); err != nil {
		log.Fatalf("Error generating summary report: %v", err)
	}
	fmt.Println("Summary report generated: pending_payment_summary.txt")
}

func getProjectID(ctx context.Context, client *githubv4.Client, org string, projectNumber int) (string, error) {
	var query struct {
		Organization struct {
			ProjectV2 struct {
				ID string
			} `graphql:"projectV2(number: $number)"`
		} `graphql:"organization(login: $login)"`
	}

	variables := map[string]interface{}{
		"login":  githubv4.String(org),
		"number": githubv4.Int(projectNumber),
	}

	err := client.Query(ctx, &query, variables)
	if err != nil {
		return "", err
	}

	return query.Organization.ProjectV2.ID, nil
}

func getProjectItems(ctx context.Context, client *githubv4.Client, projectID string) ([]ProjectItem, error) {
	var query struct {
		Node struct {
			ProjectV2 struct {
				Items struct {
					Nodes []struct {
						ID          string
						FieldValues struct {
							Nodes []struct {
								// We need to use fragments for union types
								Status struct {
									Name string
								} `graphql:"... on ProjectV2ItemFieldSingleSelectValue"`
								Text struct {
									Text string
								} `graphql:"... on ProjectV2ItemFieldTextValue"`
								Number struct {
									Number float64
								} `graphql:"... on ProjectV2ItemFieldNumberValue"`
							}
						} `graphql:"fieldValues(first: 100)"`
						Content struct {
							Issue struct {
								Title     string
								URL       string
								CreatedAt time.Time
								UpdatedAt time.Time
								Body      string
								Assignees struct {
									Nodes []struct {
										Login string
									}
								} `graphql:"assignees(first: 100)"`
								Labels struct {
									Nodes []struct {
										Name string
									}
								} `graphql:"labels(first: 100)"`
							} `graphql:"... on Issue"`
						}
					}
				} `graphql:"items(first: 100)"`
			} `graphql:"... on ProjectV2"`
		} `graphql:"node(id: $id)"`
	}

	variables := map[string]interface{}{
		"id": githubv4.ID(projectID),
	}

	err := client.Query(ctx, &query, variables)
	if err != nil {
		return nil, err
	}

	var items []ProjectItem
	for _, node := range query.Node.ProjectV2.Items.Nodes {
		issue := node.Content.Issue
		// Check if the item is in "Pending Payment" status
		isPendingPayment := false
		var recipient string
		var bountyAmount string
		var bountySymbol string

		for _, fieldValue := range node.FieldValues.Nodes {
			if fieldValue.Status.Name == "Pending Payment" {
				isPendingPayment = true
			}
			// Check for recipient field (text field)
			if fieldValue.Text.Text != "" {
				// Check if this text field contains a bounty value
				if strings.HasSuffix(strings.TrimSpace(fieldValue.Text.Text), "BUIDL") {
					parts := strings.Fields(fieldValue.Text.Text)
					if len(parts) == 2 {
						bountyAmount = parts[0]
						bountySymbol = parts[1]
					}
				} else if !strings.Contains(fieldValue.Text.Text, "BUIDL") {
					// Only set as recipient if it's not a bounty value
					recipient = fieldValue.Text.Text
				}
			}
			// Keep the number field check as a fallback
			if fieldValue.Number.Number > 0 {
				bountyAmount = fmt.Sprintf("%.0f", fieldValue.Number.Number)
				bountySymbol = "BUIDL"
			}
		}

		if isPendingPayment {
			assignees := make([]string, len(issue.Assignees.Nodes))
			for i, a := range issue.Assignees.Nodes {
				assignees[i] = a.Login
			}
			labels := make([]string, len(issue.Labels.Nodes))
			for i, l := range issue.Labels.Nodes {
				labels[i] = l.Name
			}

			items = append(items, ProjectItem{
				ID:           node.ID,
				Title:        issue.Title,
				URL:          issue.URL,
				CreatedAt:    issue.CreatedAt,
				UpdatedAt:    issue.UpdatedAt,
				AssignedTo:   assignees,
				Labels:       labels,
				Description:  issue.Body,
				Recipient:    recipient,
				BountyAmount: bountyAmount,
				BountySymbol: bountySymbol,
			})
		}
	}

	return items, nil
}

func generateCSV(items []ProjectItem, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"ID", "Title", "URL", "Created At", "Updated At", "Due Date", "Description", "Recipient", "Bounty Amount", "Bounty Symbol"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write data
	for _, item := range items {
		row := []string{
			item.ID,
			item.Title,
			item.URL,
			item.CreatedAt.Format(time.RFC3339),
			item.UpdatedAt.Format(time.RFC3339),
			item.DueDate,
			item.Description,
			item.Recipient,
			item.BountyAmount,
			item.BountySymbol,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func generateSummaryReport(items []ProjectItem, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	totalBounty := 0.0
	for _, item := range items {
		if item.BountyAmount != "" {
			bountyValue := 0.0
			fmt.Sscanf(item.BountyAmount, "%f", &bountyValue)
			totalBounty += bountyValue
		}
	}

	// Write summary
	fmt.Fprintf(file, "# Project Summary Report\n")
	fmt.Fprintf(file, "Generated on: %s\n\n", time.Now().Format(time.RFC1123))

	fmt.Fprintf(file, "## Overview\n")
	fmt.Fprintf(file, "Total Items: %d\n", len(items))
	fmt.Fprintf(file, "Total Bounty Value: %.0f BUIDL\n\n", totalBounty)

	fmt.Fprintf(file, "## Items by Recipient\n")
	recipientMap := make(map[string]float64)
	for _, item := range items {
		if item.Recipient != "" {
			bountyValue := 0.0
			fmt.Sscanf(item.BountyAmount, "%f", &bountyValue)
			recipientMap[item.Recipient] += bountyValue
		}
	}
	for recipient, amount := range recipientMap {
		fmt.Fprintf(file, "- %s: %.0f BUIDL\n", recipient, amount)
	}
	fmt.Fprintf(file, "\n")

	fmt.Fprintf(file, "## Recent Activity\n")
	count := 0
	for _, item := range items {
		if count >= 5 {
			break
		}
		fmt.Fprintf(file, "- %s (Updated: %s) - Recipient: %s, Bounty: %s %s\n",
			item.Title,
			item.UpdatedAt.Format("2006-01-02"),
			item.Recipient,
			item.BountyAmount,
			item.BountySymbol,
		)
		count++
	}

	return nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
