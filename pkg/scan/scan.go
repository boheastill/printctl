package scan

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type ScanOptions struct {
	IP        string
	DPI       int
	ColorMode string // "RGB24" or "Grayscale8"
	Format    string // "jpeg" or "pdf"
	Output    string
}

// DefaultOptions returns standard A4 300 DPI color scan options
func DefaultOptions(ip string, outputPath string) ScanOptions {
	format := "jpeg"
	if strings.HasSuffix(strings.ToLower(outputPath), ".pdf") {
		format = "pdf"
	}
	return ScanOptions{
		IP:        ip,
		DPI:       300,
		ColorMode: "RGB24",
		Format:    format,
		Output:    outputPath,
	}
}

// ExecuteScan triggers the optical scan via eSCL REST API and saves the file
func ExecuteScan(opts ScanOptions) error {
	if opts.DPI <= 0 {
		opts.DPI = 300
	}
	if opts.ColorMode == "" {
		opts.ColorMode = "RGB24"
	}
	docFormat := "image/jpeg"
	if opts.Format == "pdf" || strings.HasSuffix(strings.ToLower(opts.Output), ".pdf") {
		docFormat = "application/pdf"
	}

	// Calculate A4 pixel bounds based on DPI
	height := int(3508 * (float64(opts.DPI) / 300.0))
	width := int(2550 * (float64(opts.DPI) / 300.0))

	xmlPayload := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<scan:ScanSettings xmlns:scan="http://schemas.hp.com/imaging/escl/2011/05/03" xmlns:pwg="http://www.pwg.org/schemas/2010/12/sm">
   <pwg:Version>2.0</pwg:Version>
   <scan:Intent>Document</scan:Intent>
   <pwg:ScanRegions>
      <pwg:ScanRegion>
         <pwg:Height>%d</pwg:Height>
         <pwg:Width>%d</pwg:Width>
         <pwg:XOffset>0</pwg:XOffset>
         <pwg:YOffset>0</pwg:YOffset>
      </pwg:ScanRegion>
   </pwg:ScanRegions>
   <pwg:DocumentFormat>%s</pwg:DocumentFormat>
   <scan:ColorMode>%s</scan:ColorMode>
   <scan:XResolution>%d</scan:XResolution>
   <scan:YResolution>%d</scan:YResolution>
   <pwg:InputSource>Platen</pwg:InputSource>
</scan:ScanSettings>`, height, width, docFormat, opts.ColorMode, opts.DPI, opts.DPI)

	jobURL := fmt.Sprintf("http://%s/eSCL/ScanJobs", opts.IP)
	req, err := http.NewRequest("POST", jobURL, bytes.NewBufferString(xmlPayload))
	if err != nil {
		return fmt.Errorf("failed to create scan request: %w", err)
	}
	req.Header.Set("Content-Type", "text/xml")
	req.Header.Set("User-Agent", "printctl/0.0.1-alpha")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to submit scan job to %s: %w", opts.IP, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("scanner returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	location := resp.Header.Get("Location")
	if location == "" {
		return fmt.Errorf("no Location header in scanner response")
	}

	nextDocURL := location + "/NextDocument"
	if !strings.HasPrefix(location, "http") {
		nextDocURL = fmt.Sprintf("http://%s%s/NextDocument", opts.IP, location)
	}

	// Poll until hardware optical scan completes
	pullClient := &http.Client{Timeout: 30 * time.Second}
	var data []byte
	var lastErr error

	for i := 0; i < 25; i++ {
		time.Sleep(1500 * time.Millisecond)
		docResp, err := pullClient.Get(nextDocURL)
		if err != nil {
			lastErr = err
			continue
		}
		if docResp.StatusCode == http.StatusOK {
			data, err = io.ReadAll(docResp.Body)
			docResp.Body.Close()
			if err == nil && len(data) > 0 {
				lastErr = nil
				break
			}
		} else {
			docResp.Body.Close()
		}
	}

	if len(data) == 0 {
		if lastErr != nil {
			return fmt.Errorf("timed out downloading scanned document: %w", lastErr)
		}
		return fmt.Errorf("timed out waiting for scanner optical head")
	}

	// Output to file or stdout
	if opts.Output == "-" || opts.Output == "" {
		_, err = os.Stdout.Write(data)
		return err
	}

	return os.WriteFile(opts.Output, data, 0644)
}
