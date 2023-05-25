package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	cfddns "github.com/tiewei/ddns/cf"
)

const (
	DDNS_API_TOKEN_KEY = "DDNS_API_TOKEN"
	DDNS_ZONE_KEY      = "DDNS_ZONE"
	DDNS_SUBDOMAIN_KEY = "DDNS_SUBDOMAIN"
	DDNS_PROXIED_KEY   = "DDNS_PROXIED"
	DDNS_INTERVAL_KEY  = "DDNS_INTERVAL"
)

func run(d *cfddns.DDNS, domain string, zone string, proxied bool, timeout time.Duration) {
	ctx, cancelFn := context.WithTimeout(context.Background(), timeout-1*time.Second)
	defer cancelFn()
	log.Println("Start reconciling DNS")
	if err := d.Reconcile(ctx, domain, zone, proxied); err != nil {
		log.Printf("Failed to reconcile DNS: %s\n", err.Error())
		return
	}
	log.Println("Successfully reconciled DNS")

}

func main() {
	apiToken := os.Getenv(DDNS_API_TOKEN_KEY)
	zone := strings.TrimLeft(os.Getenv(DDNS_ZONE_KEY), ".")
	subdomain := strings.TrimRight(os.Getenv(DDNS_SUBDOMAIN_KEY), ".")
	proxied := os.Getenv(DDNS_PROXIED_KEY)
	interval := os.Getenv(DDNS_INTERVAL_KEY)

	if apiToken == "" {
		log.Fatalf("%s must be set", DDNS_API_TOKEN_KEY)
	}
	domain := ""
	if subdomain != "" {
		subdomain = subdomain + "."
	}

	if zone == "" {
		log.Fatalf("%s must be set", DDNS_ZONE_KEY)
	}
	domain = subdomain + zone

	var cfProxied bool
	if strings.ToLower(proxied) == "y" || strings.ToLower(proxied) == "yes" {
		cfProxied = true
	}

	intervalTime := 5 * time.Minute
	if interval != "" {
		if duration, err := time.ParseDuration(interval); err != nil {
			log.Printf("Failed parse %s %s, use 5min\n", DDNS_INTERVAL_KEY, interval)
		} else if duration < intervalTime {
			log.Printf("Interval %s is too short, use 5min\n", interval)
		} else {
			intervalTime = duration
		}
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(intervalTime)
	d, err := cfddns.New(apiToken)
	if err != nil {
		log.Fatalln(err.Error())
	}
	log.Printf("Starting DDNS service for %q [proxied: %t, interval: %s]", domain, cfProxied, intervalTime)
	run(d, domain, zone, cfProxied, intervalTime)
	for {
		select {
		case sig := <-sigs:
			log.Printf("Stop DDNS service after received SIG: %s\n", sig)
			return
		case <-ticker.C:
			run(d, domain, zone, cfProxied, intervalTime)
		}
	}
}
