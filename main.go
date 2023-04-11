package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/getlantern/systray"
)

type Session struct {
	StartTime int64 `json:"startTime"`
	Duration  int   `json:"duration"`
}

type Stats struct {
	TotalUsageTime int       `json:"totalUsageTime"`
	Sessions       []Session `json:"sessions"`
}

var (
	quit  chan struct{}
	stats Stats
)

func onReady() {
	// Set up the systray icon and menu
	systray.SetIcon(getIcon())
	systray.SetTitle("Device Tracker")
	systray.SetTooltip("Device Tracker")

	// Initialize the stats
	stats, _ = loadStats()

	// newSession := Session{StartTime: time.Now().Unix(), Duration: 0}

	// Used to check if new session was created in the timer loop or it will overwrite the last session
	created := 0

	// Start the timer
	timer := time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case <-timer.C:
				// Increment the usage times
				stats.TotalUsageTime++

				// Create a new session object if doesnt exist
				if len(stats.Sessions) == 0 && created == 0 {
					session := Session{StartTime: time.Now().Unix(), Duration: 0}
					stats.Sessions = append(stats.Sessions, session)
					created = 1
				}
				// Create a new session object if last session start time + duration is before now
				lastSession := stats.Sessions[len(stats.Sessions)-1]
				if lastSession.StartTime+int64(lastSession.Duration) <= time.Now().Unix() && created == 0 {
					session := Session{StartTime: time.Now().Unix(), Duration: 0}
					stats.Sessions = append(stats.Sessions, session)
					created = 1
				}

				// lastSession.duration = time since lastSession.startTime
				stats.Sessions[len(stats.Sessions)-1].Duration = int(time.Now().Unix() - lastSession.StartTime)

				// Append the new session to the last session in sessions array in stats
				// stats.Sessions[len(stats.Sessions)-1] = newSession

				err := saveStats(stats)
				if err != nil {
					fmt.Println("Error saving stats:", err)
				}

				// Update the tooltip with the usage time
				tooltip := fmt.Sprintf("Total Usage Time: %s\nSession Time: %s",
					formatTime(stats.TotalUsageTime),
					formatTime(getCurrentSessionTime()))
				systray.SetTooltip(tooltip)
			case <-quit:
				timer.Stop()
				return
			}
		}
	}()

	// Add a menu item to quit the program
	mQuit := systray.AddMenuItem("Quit", "Quit the program")
	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
		close(quit)
	}()
}

func onExit() {
	// Save the stats before exiting
	saveStats(stats)
}

func loadStats() (Stats, error) {
	statsFile := "stats.json"
	data, err := os.ReadFile(statsFile)
	if err != nil {
		return Stats{}, err
	}

	var stats Stats
	err = json.Unmarshal(data, &stats)
	if err != nil {
		return Stats{}, err
	}

	return stats, nil
}

func saveStats(stats Stats) error {
	statsFile := "stats.json"
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}

	err = os.WriteFile(statsFile, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func getCurrentSessionTime() int {
	// Get the time elapsed in the current session
	if len(stats.Sessions) > 0 {
		session := stats.Sessions[len(stats.Sessions)-1]
		sessionTime := time.Now().Unix() - session.StartTime
		return int(sessionTime)
	}
	return 0
}

func getIcon() []byte {
	// Load the icon from the file system
	iconFile := "icon.ico"
	data, err := os.ReadFile(iconFile)
	if err != nil {
		fmt.Println("Error loading icon:", err)
		return []byte{}
	}
	return data
}

func formatTime(totalSeconds int) string {
	// Format the total seconds into a human-readable format
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func main() {
	// Set up the quit channel
	quit = make(chan struct{})

	// Set up the systray
	go systray.Run(onReady, onExit)

	// Wait for the program to exit
	<-quit
}
