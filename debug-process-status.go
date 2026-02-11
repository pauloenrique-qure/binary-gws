package main

import (
	"fmt"
	"github.com/shirou/gopsutil/v3/process"
)

func main() {
	procs, err := process.Processes()
	if err != nil {
		fmt.Printf("Error getting processes: %v\n", err)
		return
	}

	fmt.Printf("Total processes: %d\n\n", len(procs))

	// Mapa para contar estados
	stateCount := make(map[string]int)
	errorCount := 0
	emptyCount := 0

	// Muestra primeros 20 procesos para debug
	fmt.Println("=== First 20 processes ===")
	for i, p := range procs {
		if i >= 20 {
			break
		}

		name, _ := p.Name()
		status, err := p.Status()

		if err != nil {
			fmt.Printf("%3d. PID=%-6d Name=%-20s Status=ERROR: %v\n", i+1, p.Pid, name, err)
			continue
		}

		fmt.Printf("%3d. PID=%-6d Name=%-20s Status=%v (len=%d)\n", i+1, p.Pid, name, status, len(status))
	}

	// Cuenta todos los estados
	fmt.Println("\n=== Counting all process states ===")
	for _, p := range procs {
		status, err := p.Status()

		if err != nil {
			errorCount++
			continue
		}

		if len(status) == 0 {
			emptyCount++
			continue
		}

		// Cuenta el primer carácter del estado
		state := status[0]
		stateCount[state]++
	}

	// Muestra resultados
	fmt.Println("\n=== Results ===")
	for state, count := range stateCount {
		var desc string
		switch state {
		case "R":
			desc = "Running"
		case "S":
			desc = "Sleeping (interruptible)"
		case "D":
			desc = "Disk sleep (uninterruptible)"
		case "Z":
			desc = "Zombie"
		case "T":
			desc = "Stopped"
		case "I":
			desc = "Idle"
		default:
			desc = "Unknown"
		}
		fmt.Printf("  %s (%s): %d\n", state, desc, count)
	}

	fmt.Printf("\n  Errors: %d\n", errorCount)
	fmt.Printf("  Empty: %d\n", emptyCount)
	fmt.Printf("  Total counted: %d\n", len(procs))

	// Comparación con el código actual
	currentRunning := stateCount["R"]
	currentSleeping := stateCount["S"]

	fmt.Println("\n=== Current Code Behavior ===")
	fmt.Printf("  Running (R): %d\n", currentRunning)
	fmt.Printf("  Sleeping (S): %d\n", currentSleeping)
	fmt.Printf("  Other states ignored: %d\n", len(procs)-currentRunning-currentSleeping-errorCount-emptyCount)
}
