package print

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/gif"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type PrintOptions struct {
	IP        string
	PaperSize string // "A4", "4x6"
	Copies    int
	Quality   string // "normal", "high"
}

// DefaultPrintOptions returns standard A4 print options
func DefaultPrintOptions(ip string) PrintOptions {
	return PrintOptions{
		IP:        ip,
		PaperSize: "A4",
		Copies:    1,
		Quality:   "normal",
	}
}

// PrintFile prints a file (PDF or Image) to the network printer via IPP
func PrintFile(filePath string, opts PrintOptions) error {
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	isPDF := strings.HasSuffix(strings.ToLower(filePath), ".pdf") || bytes.HasPrefix(fileData, []byte("%PDF-"))

	var payload []byte
	var mimeType string

	if isPDF {
		payload = fileData
		mimeType = "application/pdf"
	} else {
		// Image input: process through normalization & PDF canvas layout engine
		pdfData, err := ImageToA4PDF(fileData)
		if err != nil {
			return fmt.Errorf("failed to layout image onto A4 PDF: %w", err)
		}
		payload = pdfData
		mimeType = "application/pdf"
	}

	return SendIPPPrintJob(opts.IP, payload, mimeType, filePath)
}

// ImageToA4PDF takes any raw image, flattens alpha onto white, fits into A4 with safe margins, and outputs a valid PDF
func ImageToA4PDF(imgBytes []byte) ([]byte, error) {
	srcImg, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// 1. Alpha Flattening: Composite onto pure white canvas
	bounds := srcImg.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	whiteCanvas := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(whiteCanvas, whiteCanvas.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	draw.Draw(whiteCanvas, whiteCanvas.Bounds(), srcImg, bounds.Min, draw.Over)

	// 2. Encode to high-quality JPEG stream for PDF embedding
	var jpegBuf bytes.Buffer
	err = jpeg.Encode(&jpegBuf, whiteCanvas, &jpeg.Options{Quality: 95})
	if err != nil {
		return nil, fmt.Errorf("failed to encode normalized JPEG: %w", err)
	}
	jpegData := jpegBuf.Bytes()

	// 3. A4 Dimensions in Points (72 points per inch: 595.28 x 841.89)
	pageWidth := 595.28
	pageHeight := 841.89
	margin := 20.0 // ~7mm safe margin

	printableWidth := pageWidth - (2 * margin)
	printableHeight := pageHeight - (2 * margin)

	// Calculate aspect ratio scaling
	imgAspect := float64(w) / float64(h)
	pageAspect := printableWidth / printableHeight

	var drawW, drawH, drawX, drawY float64

	if imgAspect > pageAspect {
		// Image is wider than printable area: fit to width
		drawW = printableWidth
		drawH = printableWidth / imgAspect
		drawX = margin
		drawY = margin + ((printableHeight - drawH) / 2.0)
	} else {
		// Image is taller than printable area: fit to height
		drawH = printableHeight
		drawW = printableHeight * imgAspect
		drawX = margin + ((printableWidth - drawW) / 2.0)
		drawY = margin
	}

	// 4. Construct Minimal Single-Page PDF with embedded JPEG
	return buildSimplePDF(jpegData, w, h, pageWidth, pageHeight, drawX, drawY, drawW, drawH)
}

func buildSimplePDF(jpegData []byte, imgW, imgH int, pageW, pageH, x, y, w, h float64) ([]byte, error) {
	var pdf bytes.Buffer
	offsets := make([]int, 6)

	pdf.WriteString("%PDF-1.4\n")

	// Object 1: Catalog
	offsets[1] = pdf.Len()
	pdf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	// Object 2: Pages
	offsets[2] = pdf.Len()
	pdf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	// Object 3: Page
	offsets[3] = pdf.Len()
	fmt.Fprintf(&pdf, "3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] /Contents 4 0 R /Resources << /XObject << /Im1 5 0 R >> >> >>\nendobj\n", pageW, pageH)

	// Object 4: Contents (Position and scale image)
	contentStream := fmt.Sprintf("q\n%.2f 0 0 %.2f %.2f %.2f cm\n/Im1 Do\nQ\n", w, h, x, y)
	offsets[4] = pdf.Len()
	fmt.Fprintf(&pdf, "4 0 obj\n<< /Length %d >>\nstream\n%sendstream\nendobj\n", len(contentStream), contentStream)

	// Object 5: Image XObject (JPEG DCTDecode)
	offsets[5] = pdf.Len()
	fmt.Fprintf(&pdf, "5 0 obj\n<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length %d >>\nstream\n", imgW, imgH, len(jpegData))
	pdf.Write(jpegData)
	pdf.WriteString("\nendstream\nendobj\n")

	// Cross-Reference Table
	xrefOffset := pdf.Len()
	pdf.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for i := 1; i <= 5; i++ {
		fmt.Fprintf(&pdf, "%010d 00000 n \n", offsets[i])
	}

	// Trailer
	fmt.Fprintf(&pdf, "trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xrefOffset)

	return pdf.Bytes(), nil
}

// SendIPPPrintJob constructs a standard RFC 8011 IPP Print-Job request and sends to port 631
func SendIPPPrintJob(printerIP string, documentData []byte, mimeType string, jobName string) error {
	var ippReq bytes.Buffer

	// IPP Header: Version 2.0 (0x0200), Operation Print-Job (0x0002), Request-ID 1
	ippReq.Write([]byte{0x02, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x01})

	// Operation Attributes Tag (0x01)
	ippReq.WriteByte(0x01)

	// attributes-charset = utf-8
	writeStringAttr(&ippReq, 0x47, "attributes-charset", "utf-8")
	// attributes-natural-language = en-us
	writeStringAttr(&ippReq, 0x48, "attributes-natural-language", "en-us")
	// printer-uri
	printerURI := fmt.Sprintf("ipp://%s/ipp/print", printerIP)
	writeStringAttr(&ippReq, 0x45, "printer-uri", printerURI)
	// requesting-user-name
	writeStringAttr(&ippReq, 0x42, "requesting-user-name", "printctl")
	// job-name
	cleanJobName := "printctl_job"
	if jobName != "" {
		parts := strings.Split(jobName, "/")
		cleanJobName = parts[len(parts)-1]
	}
	writeStringAttr(&ippReq, 0x42, "job-name", cleanJobName)
	// document-format
	writeStringAttr(&ippReq, 0x49, "document-format", mimeType)

	// End of Attributes Tag (0x03)
	ippReq.WriteByte(0x03)

	// Append Document Binary Data
	ippReq.Write(documentData)

	// Send HTTP POST to IPP endpoint
	url := fmt.Sprintf("http://%s:631/ipp/print", printerIP)
	httpReq, err := http.NewRequest("POST", url, &ippReq)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/ipp")
	httpReq.Header.Set("User-Agent", "printctl/0.0.1-alpha")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to connect to printer at %s:631: %w", printerIP, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("printer rejected IPP job with HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// Read IPP response header (status code is in bytes 2-3)
	respData, err := io.ReadAll(resp.Body)
	if err == nil && len(respData) >= 4 {
		statusCode := int(respData[2])<<8 | int(respData[3])
		if statusCode > 0x00FF { // 0x0000 - 0x00FF are successful status codes in IPP
			return fmt.Errorf("IPP server returned error code 0x%04X", statusCode)
		}
	}

	return nil
}

func writeStringAttr(buf *bytes.Buffer, tag byte, name, value string) {
	buf.WriteByte(tag)
	// Name length and Name
	buf.WriteByte(byte(len(name) >> 8))
	buf.WriteByte(byte(len(name) & 0xFF))
	buf.WriteString(name)
	// Value length and Value
	buf.WriteByte(byte(len(value) >> 8))
	buf.WriteByte(byte(len(value) & 0xFF))
	buf.WriteString(value)
}
