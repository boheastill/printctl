package status

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
)

type PrinterStatusReport struct {
	IP           string `json:"ip"`
	ScannerState string `json:"scanner_state"`
	Online       bool   `json:"online"`
	LastJobState string `json:"last_job_state,omitempty"`
	Message      string `json:"message"`
}

type eSCLStatusXML struct {
	XMLName xml.Name `xml:"ScannerStatus"`
	State   string   `xml:"State"`
	Jobs    struct {
		JobInfo struct {
			JobState string `xml:"JobState"`
		} `xml:"JobInfo"`
	} `xml:"Jobs"`
}

// GetStatus retrieves status report from printer
func GetStatus(printerIP string) (*PrinterStatusReport, error) {
	url := fmt.Sprintf("http://%s/eSCL/ScannerStatus", printerIP)
	client := &http.Client{Timeout: 3 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "printctl/0.0.1-alpha")

	resp, err := client.Do(req)
	if err != nil {
		return &PrinterStatusReport{
			IP:      printerIP,
			Online:  false,
			Message: fmt.Sprintf("Unreachable: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &PrinterStatusReport{
			IP:      printerIP,
			Online:  true,
			Message: fmt.Sprintf("HTTP %d", resp.StatusCode),
		}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var statusXml eSCLStatusXML
	err = xml.Unmarshal(body, &statusXml)
	if err != nil {
		return &PrinterStatusReport{
			IP:      printerIP,
			Online:  true,
			Message: "Online (XML parse error)",
		}, nil
	}

	return &PrinterStatusReport{
		IP:           printerIP,
		Online:       true,
		ScannerState: statusXml.State,
		LastJobState: statusXml.Jobs.JobInfo.JobState,
		Message:      "Ready",
	}, nil
}
