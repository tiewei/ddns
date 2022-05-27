package cfddns

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

const (
	DEFAULT_IP_SOURCE    = "https://api.ipify.org"
	DEFAULT_HTTP_TIMEOUT = 3 * time.Second
)

type RecordType string

const Record_A RecordType = "A"
const Record_AAAA RecordType = "AAAA"
const Record_CNAME RecordType = "CNAME"

type DDNS struct {
	http     *http.Client
	cf       *cloudflare.API
	zoneID   string
	recordID string
}

func New(apiToken string) (*DDNS, error) {
	httpClient := &http.Client{
		Timeout: DEFAULT_HTTP_TIMEOUT,
	}
	cf, err := cloudflare.NewWithAPIToken(apiToken, cloudflare.HTTPClient(httpClient))
	if err != nil {
		return nil, err
	}
	d := &DDNS{
		cf:   cf,
		http: httpClient,
	}
	return d, d.verifyToken()
}

func (d *DDNS) verifyToken() error {
	_, err := d.cf.VerifyAPIToken(context.Background())
	return err
}

func (d *DDNS) findZoneID(ctx context.Context, zone string) (string, error) {
	if len(d.zoneID) > 0 {
		return d.zoneID, nil
	}
	zones, err := d.cf.ListZones(ctx, zone)
	if err != nil {
		return "", err
	}
	if len(zones) != 1 {
		return "", fmt.Errorf("can't find exact zone with name %s, found %d", zone, len(zone))
	}
	d.zoneID = zones[0].ID
	return d.zoneID, nil
}

func (d *DDNS) httpDo(req *http.Request) ([]byte, error) {
	resp, err := d.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request %s %s: %w", req.URL, req.RequestURI, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("unexpected return code from %s: %d", req.URL, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("faild to read body from %s", req.RequestURI)
	}
	return data, nil
}

func (d *DDNS) myIP(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, DEFAULT_IP_SOURCE, nil)
	if err != nil {
		return "", fmt.Errorf("failed to build request to get IP: %w", err)
	}
	ip, err := d.httpDo(req)
	if err != nil {
		return "", fmt.Errorf("failed to get IP: %w", err)
	}
	return string(ip), nil
}

func (d *DDNS) findRecord(ctx context.Context, name string, recordType RecordType) (cloudflare.DNSRecord, error) {
	if len(d.recordID) > 0 {
		return d.cf.DNSRecord(ctx, d.zoneID, d.recordID)
	}

	records, err := d.cf.DNSRecords(ctx, d.zoneID, cloudflare.DNSRecord{
		Name: name,
		Type: string(recordType),
	})
	if err != nil {
		return cloudflare.DNSRecord{}, err
	}
	if len(records) > 1 {
		return cloudflare.DNSRecord{}, fmt.Errorf("find more than one record with name %s, found %d", name, len(records))
	}
	if len(records) == 0 {
		return cloudflare.DNSRecord{}, nil
	}
	d.recordID = records[0].ID
	return records[0], nil
}

func (d *DDNS) createDNSRecord(ctx context.Context, rr cloudflare.DNSRecord) error {
	rcd, err := d.cf.CreateDNSRecord(ctx, d.zoneID, rr)
	if err != nil {
		return err
	}
	d.recordID = rcd.Result.ID
	return nil
}

func (d *DDNS) Reconcile(ctx context.Context, domain string, zone string, proxied bool) error {
	ip, err := d.myIP(ctx)
	if err != nil {
		return err
	}
	zoneID, err := d.findZoneID(ctx, zone)
	if err != nil {
		return err
	}

	rrd, err := d.findRecord(ctx, domain, Record_A)
	if err != nil {
		return err
	}

	rr := cloudflare.DNSRecord{
		Name:    domain,
		ZoneID:  zoneID,
		Type:    string(Record_A),
		Content: ip,
		Proxied: &proxied,
		TTL:     1,
	}
	if rrd.ID == "" {
		log.Printf("Creating new dns record: %s=%s\n", rr.Name, rr.Content)
		return d.createDNSRecord(ctx, rr)
	}
	if rrd.Content != ip {
		log.Printf("Updating new dns record: %s=%s\n", rr.Name, rr.Content)
		return d.cf.UpdateDNSRecord(ctx, d.zoneID, rrd.ID, rr)
	}
	return nil
}
