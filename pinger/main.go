package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-ping/ping"
)

type ContainerStatus struct {
	IP          string    `json:"ip"`
	PingTime    int       `json:"ping_time"`
	LastSuccess time.Time `json:"last_success"`
}

func pingHost(ip string) (int, error) {
	pinger, err := ping.NewPinger(ip)
	if err != nil {
		return 0, err
	}
	// Используем привилегированный режим для отправки ICMP-пакетов
	pinger.SetPrivileged(true)
	pinger.Count = 3
	pinger.Timeout = time.Second * 3
	if err = pinger.Run(); err != nil {
		return 0, err
	}
	stats := pinger.Statistics()
	// Возвращаем среднее время в миллисекундах
	return int(stats.AvgRtt.Milliseconds()), nil
}

func updateStatus(backendURL string, status ContainerStatus) error {
	jsonData, err := json.Marshal(status)
	if err != nil {
		return err
	}
	resp, err := http.Post(backendURL+"/status", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return err
	}
	return nil
}

func main() {
	targetsEnv := os.Getenv("TARGETS")
	if targetsEnv == "" {
		log.Fatal("No TARGETS provided")
	}
	targets := strings.Split(targetsEnv, ",")
	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		backendURL = "http://backend:8080" // имя сервиса backend в docker-compose
	}
	intervalEnv := os.Getenv("PING_INTERVAL")
	interval := time.Second * 30
	if intervalEnv != "" {
		if parsed, err := time.ParseDuration(intervalEnv); err == nil {
			interval = parsed
		}
	}
	log.Printf("Starting pinger. Targets: %v, Interval: %v, Backend: %s", targets, interval, backendURL)

	for {
		for _, ip := range targets {
			ip = strings.TrimSpace(ip)
			if ip == "" {
				continue
			}
			pingTime, err := pingHost(ip)
			if err != nil {
				log.Printf("Error pinging %s: %v", ip, err)
				continue
			}
			status := ContainerStatus{
				IP:          ip,
				PingTime:    pingTime,
				LastSuccess: time.Now().UTC(),
			}
			if err := updateStatus(backendURL, status); err != nil {
				log.Printf("Error updating status for %s: %v", ip, err)
			} else {
				log.Printf("Updated status for %s: %d ms", ip, pingTime)
			}
		}
		time.Sleep(interval)
	}
}
