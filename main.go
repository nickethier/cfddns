package main

import (
	"fmt"
	"os"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go"
	externalip "github.com/glendc/go-external-ip"
	hclog "github.com/hashicorp/go-hclog"
)

func main() {
	log := hclog.Default()
	consensus := buildConsensus(log)
	// Construct a new API object
	api, err := cloudflare.New(os.Getenv("CLOUDFLARE_TOKEN"), os.Getenv("CLOUDFLARE_EMAIL"))
	if err != nil {
		log.Error("failed to build cf api client", "error", err)
		os.Exit(1)
	}

	// Fetch the zone ID
	zoneName := os.Getenv("CLOUDFLARE_ZONE")
	if zoneName == "" {
		log.Error("CLOUDFLARE_ZONE environment var must be set")
		os.Exit(1)
	}
	id, err := api.ZoneIDByName("nick.sh") // Assuming example.com exists in your Cloudflare account already
	if err != nil {
		log.Error("failed to lookup zone ID", "zone", zoneName, "error", err)
	}

	// Find or create DNS record
	recordName := os.Getenv("RECORD") + "." + zoneName
	record, err := getRecord(api, id, recordName)
	if err != nil {
		log.Error("failed to get record", "record", recordName, "error", err)
		os.Exit(1)
	}
	if record == nil {
		record, err = createRecord(api, id, recordName, consensus)
		if err != nil {
			log.Error("failed to create record", "record", recordName, "error", err)
			os.Exit(1)
		}
	}

	// Parse interval
	interval, err := time.ParseDuration(os.Getenv("INTERVAL"))
	if err != nil && os.Getenv("INTERVAL") != "" {
		log.Warn("failed to parse INTERVAL, setting to 5m", "INTERVAL", os.Getenv("INTERVAL"), "error", err)
	}
	if interval < time.Second {
		interval = 5 * time.Minute
	}

	lastIP, err := consensus.ExternalIP()
	if err != nil {
		log.Warn("failed to derive external IP", "error", err)
	}

	log.Info("Starting ddns client", "record", recordName, "interval", interval.String())
	for {
		ip, err := consensus.ExternalIP()
		if err != nil {
			log.Warn("Failed to derive external IP", "error", err)
		} else if lastIP.String() != ip.String() {
			log.Info("IP change detected, updating record", "record", recordName, "ip", ip.String())
			if err := api.UpdateDNSRecord(id, record.ID, cloudflare.DNSRecord{Name: recordName, Content: ip.String()}); err != nil {
				log.Warn("failed to update DNS record", "record", recordName, "ip", ip.String(), "error", err)
			} else {
				lastIP = ip
				goto SLEEP
			}
		}

		record, err = getRecord(api, id, recordName)
		if err != nil {
			log.Warn("failed to lookup A record", "record", recordName, "error", err)
		} else if record == nil {
			record, err = createRecord(api, id, recordName, consensus)
			if err != nil {
				log.Warn("failed to create A record", "record", recordName, "error", err)
			}
		} else if record.Content != ip.String() {
			log.Info("IP change detected, updating record", "record", recordName, "ip", ip.String())
			if err := api.UpdateDNSRecord(id, record.ID, cloudflare.DNSRecord{Name: recordName, Content: ip.String()}); err != nil {
				log.Warn("failed to update DNS record", "record", recordName, "ip", ip.String(), "error", err)
			} else {
				lastIP = ip
			}
		}

	SLEEP:
		time.Sleep(interval)
	}
}

func getRecord(api *cloudflare.API, id, record string) (*cloudflare.DNSRecord, error) {
	filter := cloudflare.DNSRecord{Name: record}
	records, err := api.DNSRecords(id, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup DNS record: %v", err)
	}
	if len(records) > 1 {
		return nil, fmt.Errorf("too many records returned, expected 1")
	}
	if len(records) == 0 {
		return nil, nil
	}
	return &records[0], nil
}

func createRecord(api *cloudflare.API, id, record string, consensus *externalip.Consensus) (*cloudflare.DNSRecord, error) {
	ip, err := consensus.ExternalIP()
	if err != nil {
		return nil, err
	}
	resp, err := api.CreateDNSRecord(id, cloudflare.DNSRecord{
		Name:    record,
		Content: ip.String(),
		Type:    "A",
	})

	if err != nil {
		return nil, err
	}

	return &resp.Result, nil
}

func buildConsensus(logger hclog.Logger) *externalip.Consensus {
	consensus := externalip.NewConsensus(nil, logger.StandardLogger(&hclog.StandardLoggerOptions{InferLevels: true}))

	// TLS-protected providers
	consensus.AddVoter(externalip.NewHTTPSource("https://icanhazip.com/"), 3)
	consensus.AddVoter(externalip.NewHTTPSource("https://myexternalip.com/raw"), 3)

	// Plain-text providers
	consensus.AddVoter(externalip.NewHTTPSource("http://ifconfig.io/ip"), 1)
	consensus.AddVoter(externalip.NewHTTPSource("http://checkip.amazonaws.com/"), 1)
	consensus.AddVoter(externalip.NewHTTPSource("http://ident.me/"), 1)
	consensus.AddVoter(externalip.NewHTTPSource("http://whatismyip.akamai.com/"), 1)
	consensus.AddVoter(externalip.NewHTTPSource("http://tnx.nl/ip"), 1)
	consensus.AddVoter(externalip.NewHTTPSource("http://diagnostic.opendns.com/myip"), 1)

	return consensus
}
