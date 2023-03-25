package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/getlantern/systray"
)

type Stats struct {
	TotalUsageTime   int `json:"totalUsageTime"`
	SessionStartTime int `json:"sessionStartTime"`
}

var (
	quit  chan struct{}
	stats Stats
)

func newWatcher(path string) (*fsnotify.Watcher, error) {
	// Create a new watcher for the given path
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			err = watcher.Add(path)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return watcher, nil
}

func getHomeDir() string {
	if os.Getenv("HOME") != "" {
		return os.Getenv("HOME")
	}
	return os.Getenv("USERPROFILE")
}

func onReady() {
	// Set up the systray icon and menu
	systray.SetIcon(getIcon())
	systray.SetTitle("Device Tracker")
	systray.SetTooltip("Device Tracker")

	// Initialize the stats
	stats, _ = loadStats()

	// Start the timer
	timer := time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case <-timer.C:
				// Increment the usage times
				stats.TotalUsageTime++
				stats.SessionStartTime++
				err := saveStats(stats)
				if err != nil {
					fmt.Println("Error saving stats:", err)
				}

				// Update the tooltip with the usage time
				tooltip := fmt.Sprintf("Total Usage Time: %s\nSession Time: %s",
					formatTime(stats.TotalUsageTime),
					formatTime(stats.SessionStartTime))
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
	// Load the stats from the stats file
	statsFile := "stats.json"
	data, err := ioutil.ReadFile(statsFile)
	if err != nil {
		if os.IsNotExist(err) {
			// stats file doesn't exist yet, create a new one
			stats := Stats{}
			err = saveStats(stats)
			if err != nil {
				return Stats{}, err
			}

			// return the default stats (with zero usage times)
			return Stats{}, nil
		}
		return Stats{}, err
	}
	err = json.Unmarshal(data, &stats)
	if err != nil {
		return Stats{}, err
	}
	return stats, nil
}

func saveStats(stats Stats) error {
	// Save the stats to the stats file
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("stats.json", data, 0644)
	if err != nil {
		return err
	}
	return nil
}

func getTotalUsageTime() int {
	// Get the total usage time from the stats file
	stats, err := loadStats()
	if err != nil {
		fmt.Println("Error loading stats:", err)
		return 0
	}
	return stats.TotalUsageTime
}

func getCurrentSessionTime() int {
	// Get the time elapsed in the current session
	return stats.SessionStartTime
}

func getIcon() []byte {
	// Load the icon from the file system
	iconFile := "icon.ico"
	data, err := ioutil.ReadFile(iconFile)
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

	// Set up the watcher for the home directory
	homeDir := getHomeDir()
	watcher, err := newWatcher(homeDir)
	if err != nil {
		fmt.Println("Error setting up watcher:", err)
		return
	}
	defer watcher.Close()

	// Set up the systray
	go systray.Run(onReady, onExit)

	// Watch for filesystem events
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					// If a file is written to, recompile and run the program
					fmt.Println("File modified:", event.Name)
					cmd := exec.Command("go", "run", os.Args[0])
					err := cmd.Start()
					if err != nil {
						fmt.Println("Error recompiling program:", err)
					} else {
						fmt.Println("Recompiling program...")
						cmd.Wait()
					}
				}
			case err := <-watcher.Errors:
				fmt.Println("Error watching files:", err)
			case <-quit:
				return
			}
		}
	}()

	// Wait for the program to exit
	<-quit
}
