// https://developers.cloudflare.com/1.1.1.1/dns-over-https/json-format/
package redisHub

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type DNSResponse struct {
	Status   int  `json:"Status"`
	TC       bool `json:"TC"`
	RD       bool `json:"RD"`
	RA       bool `json:"RA"`
	AD       bool `json:"AD"`
	CD       bool `json:"CD"`
	Question []struct {
		Name string `json:"name"`
		Type int    `json:"type"`
	} `json:"Question"`
	Answer []struct {
		Name string `json:"name"`
		Type int    `json:"type"`
		TTL  int    `json:"TTL"`
		Data string `json:"data"`
	} `json:"Answer"`
}

var dnsClient = NewHTTPClient(10)

func ParseSchemeHostname(host string) (string, string, error) {
	if strings.Contains(host, "://") {
		u, err := url.Parse(host)
		if err != nil {
			log.Printf("parse url failed: %s\n", err)
			return "", host, err
		}
		return u.Scheme, u.Hostname(), nil
	}
	return "", host, nil
}

func checkDNS(host string) error {
	url := fmt.Sprintf("https://cloudflare-dns.com/dns-query?type=A&name=%s", host)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/dns-json")

	resp, err := dnsClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("status code of DNS query response is %d", resp.StatusCode)
	}

	d := &DNSResponse{}
	json.NewDecoder(resp.Body).Decode(&d)
	for _, x := range d.Answer {
		if validIP4(x.Data) {
			log.Printf("got DNS for %s host %s\n", x.Data, host)
			return nil
		}
	}
	return fmt.Errorf("DNS with type A for host %s has not been found through 1.1.1.1", host)
}

func validIP4(ipAddress string) bool {
	ipAddress = strings.Trim(ipAddress, " ")
	re, _ := regexp.Compile(`^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`)
	if re.MatchString(ipAddress) {
		return true
	}
	return false
}
