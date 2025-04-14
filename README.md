# BUIDL Tools

A Go application for managing and tracking pending payments in GitHub Projects.

## Features

- Fetches pending payment items from GitHub Projects
- Generates a CSV report of pending payments
- Creates a summary report of pending payments
- Tracks bounty amounts and recipients

## Prerequisites

- Go 1.16 or higher
- GitHub account with access to the target organization
- GitHub Personal Access Token with appropriate permissions

## Setup

1. Clone the repository:
```bash
git clone https://github.com/NautilusOSS/buidl-tools.git
cd buidl-tools
```

2. Install dependencies:
```bash
go mod download
```

3. Set up your GitHub Personal Access Token:
   - Go to GitHub Settings > Developer Settings > Personal Access Tokens > Tokens (classic)
   - Click "Generate new token (classic)"
   - Give your token a descriptive name
   - Select the following scopes:
     - `repo` (Full control of private repositories)
     - `read:org` (Read organization and team membership)
     - `read:project` (Read project boards)
   - Click "Generate token"
   - Copy the generated token immediately (you won't be able to see it again)

4. Set the GitHub token as an environment variable:
```bash
export GITHUB_TOKEN=your_token_here
```

## Usage

Run the application:
```bash
go run main.go
```

The application will:
1. Connect to the specified GitHub project
2. Fetch all items with "Pending Payment" status
3. Generate two files:
   - `pending_payment_tasks.csv`: Detailed CSV report of all pending payments
   - `pending_payment_summary.txt`: Summary report of pending payments

## Output Files

### pending_payment_tasks.csv
Contains detailed information about each pending payment item, including:
- ID
- Title
- URL
- Creation and update dates
- Description
- Recipient
- Bounty amount and symbol

### pending_payment_summary.txt
Provides a summary of all pending payments, including:
- Total number of pending payments
- Total bounty amount
- List of recipients and their respective amounts

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.