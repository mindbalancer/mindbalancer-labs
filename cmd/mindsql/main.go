// mindsql - CLI client for MindBalancer admin interface
package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
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

	// Interactive mode
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
	fmt.Println("Welcome to mindsql - MindBalancer Admin CLI")
	fmt.Println("Type 'help' for available commands, 'quit' to exit.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	var commandBuffer strings.Builder

	for {
		if commandBuffer.Len() == 0 {
			fmt.Print("mindsql> ")
		} else {
			fmt.Print("      -> ")
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)

		// Handle special commands
		upper := strings.ToUpper(line)
		if upper == "QUIT" || upper == "EXIT" || upper == "\\Q" {
			fmt.Println("Bye")
			break
		}
		if upper == "HELP" || upper == "\\H" {
			printHelp()
			continue
		}
		if upper == "CLEAR" || upper == "\\C" {
			commandBuffer.Reset()
			continue
		}
		if upper == "STATUS" || upper == "\\S" {
			printStatus(db)
			continue
		}

		// Accumulate command
		if commandBuffer.Len() > 0 {
			commandBuffer.WriteString(" ")
		}
		commandBuffer.WriteString(line)

		// Check if command is complete (ends with ; or is a special command)
		cmd := commandBuffer.String()
		if strings.HasSuffix(cmd, ";") || isSpecialCommand(cmd) {
			cmd = strings.TrimSuffix(cmd, ";")
			executeCommand(db, cmd)
			commandBuffer.Reset()
		}
	}

}

func isSpecialCommand(cmd string) bool {
	upper := strings.ToUpper(strings.TrimSpace(cmd))
	specialCmds := []string{
		"SHOW PROCESSLIST",
		"SHOW STATS",
		"SHOW HOSTGROUPS",
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
	fmt.Println(`
MindBalancer Admin Commands
===========================

Server Management:
  SELECT * FROM ai_servers;           - List all servers
  INSERT INTO ai_servers ...          - Add a server
  UPDATE ai_servers SET ...           - Update a server
  DELETE FROM ai_servers WHERE ...    - Remove a server
  LOAD AI SERVERS TO RUNTIME;         - Apply server changes

User Management:
  SELECT * FROM ai_users;             - List all users
  INSERT INTO ai_users ...            - Add a user
  UPDATE ai_users SET ...             - Update a user
  DELETE FROM ai_users WHERE ...      - Remove a user
  LOAD AI USERS TO RUNTIME;           - Apply user changes

Routing Rules:
  SELECT * FROM ai_routing_rules;     - List routing rules
  INSERT INTO ai_routing_rules ...    - Add a rule
  DELETE FROM ai_routing_rules ...    - Remove a rule
  LOAD AI ROUTING RULES TO RUNTIME;   - Apply rule changes

Configuration:
  SELECT * FROM global_variables;     - Show all variables
  SET variable-name = value;          - Set a variable
  LOAD VARIABLES TO RUNTIME;          - Apply variable changes
  SAVE VARIABLES TO DISK;             - Persist variables

Statistics:
  SELECT * FROM stats_ai_servers;     - Server statistics
  SELECT * FROM stats_ai_requests;    - Recent requests
  SHOW PROCESSLIST;                   - Active connections
  SHOW STATS;                         - Summary statistics
  SHOW HOSTGROUPS;                    - Hostgroup overview

Admin:
  KILL CONNECTION <id>;               - Terminate a request
  FLUSH LOGS;                         - Rotate log files
  SHUTDOWN;                           - Graceful shutdown

Shortcuts:
  \h, help     - Show this help
  \q, quit     - Exit mindsql
  \c, clear    - Clear current command
  \s, status   - Show connection status
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
