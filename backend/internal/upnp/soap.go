package upnp

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SOAPClient performs UPnP SOAP requests against a discovered control URL.
type SOAPClient struct {
	Endpoint    string
	ServiceType string
	HTTPClient  *http.Client
}

func (c *SOAPClient) client() *http.Client {
	if c != nil && c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 5 * time.Second}
}

// GetExternalIPAddress fetches the public IP from the gateway.
func (c *SOAPClient) GetExternalIPAddress() (string, error) {
	body, err := c.call("GetExternalIPAddress", nil)
	if err != nil {
		return "", err
	}
	var resp struct {
		Body struct {
			Response struct {
				ExternalIP string `xml:"NewExternalIPAddress"`
			} `xml:"GetExternalIPAddressResponse"`
			Fault *soapFault `xml:"Fault"`
		} `xml:"Body"`
	}
	if err := xml.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parse soap response: %w", err)
	}
	if resp.Body.Fault != nil {
		return "", resp.Body.Fault
	}
	if resp.Body.Response.ExternalIP == "" {
		return "", fmt.Errorf("soap response missing external ip")
	}
	return resp.Body.Response.ExternalIP, nil
}

// AddPortMapping sends a SOAP AddPortMapping request.
func (c *SOAPClient) AddPortMapping(m PortMapping) error {
	if err := ValidatePortMapping(m); err != nil {
		return err
	}
	_, err := c.call("AddPortMapping", map[string]string{
		"NewRemoteHost":            "",
		"NewExternalPort":          fmt.Sprintf("%d", m.ExternalPort),
		"NewProtocol":              strings.ToUpper(strings.TrimSpace(m.Protocol)),
		"NewInternalPort":          fmt.Sprintf("%d", m.InternalPort),
		"NewInternalClient":        m.InternalIP,
		"NewEnabled":               "1",
		"NewPortMappingDescription": m.Description,
		"NewLeaseDuration":         fmt.Sprintf("%d", m.LeaseDurationSeconds),
	})
	return err
}

// DeletePortMapping sends a SOAP DeletePortMapping request.
func (c *SOAPClient) DeletePortMapping(protocol string, externalPort int) error {
	if err := validateProtocol(protocol); err != nil {
		return err
	}
	if err := validatePort("external port", externalPort); err != nil {
		return err
	}
	_, err := c.call("DeletePortMapping", map[string]string{
		"NewRemoteHost":   "",
		"NewExternalPort": fmt.Sprintf("%d", externalPort),
		"NewProtocol":     strings.ToUpper(strings.TrimSpace(protocol)),
	})
	return err
}

// GetGenericPortMappingEntry fetches a port mapping entry by index from the gateway.
func (c *SOAPClient) GetGenericPortMappingEntry(index int) (PortMapping, error) {
	body, err := c.call("GetGenericPortMappingEntry", map[string]string{
		"NewPortMappingIndex": fmt.Sprintf("%d", index),
	})
	if err != nil {
		return PortMapping{}, err
	}

	var resp struct {
		Body struct {
			Response struct {
				RemoteHost     string `xml:"NewRemoteHost"`
				ExternalPort   int    `xml:"NewExternalPort"`
				Protocol       string `xml:"NewProtocol"`
				InternalPort   int    `xml:"NewInternalPort"`
				InternalClient string `xml:"NewInternalClient"`
				Enabled        string `xml:"NewEnabled"` // "1" or "0"
				Description    string `xml:"NewPortMappingDescription"`
				LeaseDuration  int    `xml:"NewLeaseDuration"`
			} `xml:"GetGenericPortMappingEntryResponse"`
			Fault *soapFault `xml:"Fault"`
		} `xml:"Body"`
	}

	if err := xml.Unmarshal(body, &resp); err != nil {
		return PortMapping{}, fmt.Errorf("parse soap response: %w", err)
	}
	if resp.Body.Fault != nil {
		return PortMapping{}, resp.Body.Fault
	}

	m := PortMapping{
		Protocol:             resp.Body.Response.Protocol,
		ExternalPort:         resp.Body.Response.ExternalPort,
		InternalIP:           resp.Body.Response.InternalClient,
		InternalPort:         resp.Body.Response.InternalPort,
		Description:          resp.Body.Response.Description,
		LeaseDurationSeconds: resp.Body.Response.LeaseDuration,
	}
	return m, nil
}

func (c *SOAPClient) call(action string, body map[string]string) ([]byte, error) {
	envelope, err := buildSOAPEnvelope(action, c.ServiceType, body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.Endpoint, bytes.NewReader(envelope))
	if err != nil {
		return nil, fmt.Errorf("create soap request: %w", err)
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", fmt.Sprintf("%q", c.ServiceType+"#"+action))

	resp, err := c.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("post soap request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read soap response: %w", err)
	}
	if resp.StatusCode >= 400 {
		if faultErr := parseSOAPFault(data); faultErr != nil {
			return nil, faultErr
		}
		return nil, fmt.Errorf("soap request failed: %s", resp.Status)
	}
	if faultErr := parseSOAPFault(data); faultErr != nil {
		return nil, faultErr
	}
	return data, nil
}

func buildSOAPEnvelope(action, serviceType string, body map[string]string) ([]byte, error) {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="utf-8"?>`)
	b.WriteString(`<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">`)
	b.WriteString(`<s:Body>`)
	b.WriteString(`<u:`)
	b.WriteString(xmlEscape(action))
	b.WriteString(` xmlns:u="`)
	b.WriteString(xmlEscape(serviceType))
	b.WriteString(`">`)
	for k, v := range body {
		b.WriteString(`<`)
		b.WriteString(xmlEscape(k))
		b.WriteString(`>`)
		b.WriteString(xmlEscape(v))
		b.WriteString(`</`)
		b.WriteString(xmlEscape(k))
		b.WriteString(`>`)
	}
	b.WriteString(`</u:`)
	b.WriteString(xmlEscape(action))
	b.WriteString(`>`)
	b.WriteString(`</s:Body></s:Envelope>`)
	return []byte(b.String()), nil
}

func xmlEscape(s string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

type soapFault struct {
	XMLName xml.Name `xml:"Fault"`
	String  string   `xml:"faultstring"`
	Detail  struct {
		UPnPError struct {
			Code int    `xml:"errorCode"`
			Desc string `xml:"errorDescription"`
		} `xml:"UPnPError"`
	} `xml:"detail"`
}

func (f *soapFault) Error() string {
	if f == nil {
		return "soap fault"
	}
	if f.Detail.UPnPError.Desc != "" {
		return fmt.Sprintf("soap fault: %s (%d): %s", f.String, f.Detail.UPnPError.Code, f.Detail.UPnPError.Desc)
	}
	if f.String != "" {
		return "soap fault: " + f.String
	}
	return "soap fault"
}

func parseSOAPFault(data []byte) error {
	var env struct {
		Body struct {
			Fault *soapFault `xml:"Fault"`
		} `xml:"Body"`
	}
	if err := xml.Unmarshal(data, &env); err != nil {
		return nil
	}
	if env.Body.Fault != nil {
		return env.Body.Fault
	}
	return nil
}
