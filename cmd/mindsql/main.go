// mindsql - CLI client for MindBalancer admin interface
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/peterh/liner"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	host := flag.String("h", "127.0.0.1", "MindBalancer admin host")
	port := flag.Int("P", 6032, "MindBalancer admin port")
	user := flag.String("u", "admin", "Admin username")
	password := flag.String("p", "", "Admin password")
	execute := flag.String("e", "", "Execute command and exit")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("mindsql %s (built %s)\n", Version, BuildTime)
		os.Exit(0)
	}

	// Connect to MindBalancer admin interface
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/", *user, *password, *host, *port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "Connection failed: %v\n", err)
		os.Exit(1)
	}

	// Single command mode
	if *execute != "" {
		executeCommand(db, *execute)
		return
	}

	// Interactive mode with readline support
	runInteractive(db)
}

func executeCommand(db *sql.DB, command string) {
	command = strings.TrimSpace(command)
	if command == "" {
		return
	}

	upper := strings.ToUpper(command)

	// Handle QUIT/EXIT
	if upper == "QUIT" || upper == "EXIT" || upper == "\\Q" {
		return
	}

	// Handle HELP
	if upper == "HELP" || upper == "\\H" {
		printHelp()
		return
	}

	rows, err := db.Query(command)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return
	}

	// Print results
	if len(columns) == 1 && columns[0] == "result" {
		// Raw result (our custom result format)
		for rows.Next() {
			var result string
			if err := rows.Scan(&result); err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
				return
			}
			fmt.Println(result)
		}
	} else {
		// Tabular result
		printTabularResults(rows, columns)
	}
}

func printTabularResults(rows *sql.Rows, columns []string) {
	// Prepare scan targets
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// Collect all rows first
	var allRows [][]string
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}
		row := make([]string, len(columns))
		for i, v := range values {
			switch val := v.(type) {
			case nil:
				row[i] = "NULL"
			case []byte:
				row[i] = string(val)
			default:
				row[i] = fmt.Sprintf("%v", val)
			}
		}
		allRows = append(allRows, row)
	}

	// Calculate column widths
	widths := make([]int, len(columns))
	for i, col := range columns {
		widths[i] = len(col)
	}
	for _, row := range allRows {
		for i, val := range row {
			if len(val) > widths[i] {
				widths[i] = len(val)
			}
		}
	}

	// Print header
	printSeparator(widths)
	printRow(columns, widths)
	printSeparator(widths)

	// Print rows
	for _, row := range allRows {
		printRow(row, widths)
	}
	printSeparator(widths)

	fmt.Printf("%d rows in set\n", len(allRows))
}

func printSeparator(widths []int) {
	fmt.Print("+")
	for _, w := range widths {
		fmt.Print(strings.Repeat("-", w+2))
		fmt.Print("+")
	}
	fmt.Println()
}

func printRow(values []string, widths []int) {
	fmt.Print("|")
	for i, v := range values {
		fmt.Printf(" %-*s |", widths[i], v)
	}
	fmt.Println()
}

func runInteractive(db *sql.DB) {
	line := liner.NewLiner()
	defer line.Close()

	// Configure liner
	line.SetCtrlCAborts(true)
	line.SetMultiLineMode(true)

	// History file
	historyFile := getHistoryFile()
	if f, err := os.Open(historyFile); err == nil {
		line.ReadHistory(f)
		f.Close()
	}

	// Save history on exit
	defer func() {
		if f, err := os.Create(historyFile); err == nil {
			line.WriteHistory(f)
			f.Close()
		}
	}()

	// Auto-completion
	line.SetCompleter(func(line string) []string {
		commands := []string{
			"SELECT * FROM ai_servers;",
			"SELECT * FROM ai_users;",
			"SELECT * FROM ai_routing_rules;",
			"SELECT * FROM global_variables;",
			"SELECT * FROM stats_ai_servers;",
			"SELECT * FROM stats_ai_requests;",
			"SHOW PROCESSLIST;",
			"SHOW STATS;",
			"SHOW HOSTGROUPS;",
			"SHOW API KEYS;",
			"SHOW HEALTH STATUS;",
			"LOAD AI SERVERS TO RUNTIME;",
			"LOAD AI ROUTING RULES TO RUNTIME;",
			"INSERT INTO ai_servers",
			"DELETE FROM ai_servers WHERE name = ",
			"quit",
			"help",
		}

		var completions []string
		lower := strings.ToLower(line)
		for _, cmd := range commands {
			if strings.HasPrefix(strings.ToLower(cmd), lower) {
				completions = append(completions, cmd)
			}
		}
		return completions
	})

	fmt.Println("Welcome to mindsql - MindBalancer Admin CLI")
	fmt.Println("Type 'help' for available commands, 'quit' to exit.")
	fmt.Println()

	var commandBuffer strings.Builder

	for {
		prompt := "mindsql> "
		if commandBuffer.Len() > 0 {
			prompt = "      -> "
		}

		input, err := line.Prompt(prompt)
		if err != nil {
			if err == liner.ErrPromptAborted {
				if commandBuffer.Len() > 0 {
					commandBuffer.Reset()
					fmt.Println()
					continue
				}
				fmt.Println("Bye")
				break
			}
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Handle special commands
		upper := strings.ToUpper(input)
		if upper == "QUIT" || upper == "EXIT" || upper == "\\Q" {
			fmt.Println("Bye")
			break
		}
		if upper == "HELP" || upper == "\\H" {
			printHelp()
			line.AppendHistory(input)
			continue
		}
		if upper == "CLEAR" || upper == "\\C" {
			commandBuffer.Reset()
			continue
		}
		if upper == "STATUS" || upper == "\\S" {
			printStatus(db)
			line.AppendHistory(input)
			continue
		}

		// Accumulate command
		if commandBuffer.Len() > 0 {
			commandBuffer.WriteString(" ")
		}
		commandBuffer.WriteString(input)

		// Check if command is complete (ends with ; or is a special command)
		cmd := commandBuffer.String()
		if strings.HasSuffix(cmd, ";") || isSpecialCommand(cmd) {
			cmd = strings.TrimSuffix(cmd, ";")
			line.AppendHistory(commandBuffer.String())
			executeCommand(db, cmd)
			commandBuffer.Reset()
		}
	}
}

func getHistoryFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".mindsql_history"
	}
	return filepath.Join(home, ".mindsql_history")
}

func isSpecialCommand(cmd string) bool {
	upper := strings.ToUpper(strings.TrimSpace(cmd))
	specialCmds := []string{
		"SHOW PROCESSLIST",
		"SHOW STATS",
		"SHOW HOSTGROUPS",
		"SHOW API KEYS",
		"SHOW HEALTH STATUS",
		"SHUTDOWN",
		"FLUSH LOGS",
	}
	for _, sc := range specialCmds {
		if strings.HasPrefix(upper, sc) {
			return true
		}
	}
	return false
}

func printHelp() {
	fmt.Print(`
MindBalancer Admin Commands
===========================

Server Management:
  SELECT * FROM ai_servers;           - List all servers
  INSERT INTO ai_servers (name, provider_type, endpoint, api_key_encrypted, hostgroup, weight)
    VALUES ('name', 'openai', 'https://...', 'sk-xxx', 0, 1);
  DELETE FROM ai_servers WHERE name = 'xxx';
  LOAD AI SERVERS TO RUNTIME;         - Apply server changes

User Management:
  SELECT * FROM ai_users;             - List all users
  LOAD AI USERS TO RUNTIME;           - Apply user changes

Routing Rules:
  SELECT * FROM ai_routing_rules;     - List routing rules
  LOAD AI ROUTING RULES TO RUNTIME;   - Apply rule changes

Configuration:
  SELECT * FROM global_variables;     - Show all variables
  SET variable-name = value;          - Set a variable

Statistics & Monitoring:
  SELECT * FROM stats_ai_servers;     - Server statistics
  SELECT * FROM stats_ai_requests;    - Recent requests
  SHOW PROCESSLIST;                   - Active connections
  SHOW STATS;                         - Summary statistics
  SHOW HOSTGROUPS;                    - Hostgroup overview
  SHOW API KEYS;                      - Show API keys (masked)
  SHOW HEALTH STATUS;                 - Server health status

Shortcuts:
  ↑/↓        - Navigate command history
  Tab        - Auto-complete commands
  Ctrl+C     - Cancel current input
  \h, help   - Show this help
  \q, quit   - Exit mindsql
  \c, clear  - Clear current command
  \s, status - Show connection status
`)
}

func printStatus(db *sql.DB) {
	fmt.Println("--------------")
	fmt.Println("mindsql - MindBalancer Admin CLI")
	fmt.Println("--------------")

	// Try to get some stats
	rows, err := db.Query("SHOW STATS")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var result string
			if err := rows.Scan(&result); err == nil {
				fmt.Println(result)
			}
		}
	}
	fmt.Println("--------------")
}
