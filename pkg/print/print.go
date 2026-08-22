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

// PrintFile prints a file (Image or PDF) to the network printer via IPP
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
		mimeType = "application/octet-stream"
	} else {
		// Image input: process through normalization & A4 canvas layout engine
		jpegData, err := NormalizeImageToA4JPEG(fileData)
		if err != nil {
			return fmt.Errorf("failed to layout image onto A4: %w", err)
		}
		payload = jpegData
		mimeType = "image/jpeg"
	}

	return SendIPPPrintJob(opts.IP, payload, mimeType, filePath)
}

// NormalizeImageToA4JPEG flattens alpha onto white, fits into A4 @ 300DPI with safe margins
func NormalizeImageToA4JPEG(imgBytes []byte) ([]byte, error) {
	return NormalizeImageToA4JPEGWithMark(imgBytes, false)
}

// NormalizeImageToA4JPEGWithMark adds optional foolproof duplex corner indicator
func NormalizeImageToA4JPEGWithMark(imgBytes []byte, addCornerMark bool) ([]byte, error) {
	srcImg, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// 1. Alpha Flattening: Composite onto pure white canvas
	bounds := srcImg.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// A4 Dimensions at 300 DPI
	const a4W = 2480
	const a4H = 3508
	const margin = 80 // ~7mm safe margin

	printableW := a4W - (2 * margin)
	printableH := a4H - (2 * margin)

	// Create White A4 Canvas
	canvas := image.NewRGBA(image.Rect(0, 0, a4W, a4H))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	// Calculate aspect ratio scaling
	imgAspect := float64(w) / float64(h)
	pageAspect := float64(printableW) / float64(printableH)

	var drawW, drawH, drawX, drawY int

	if imgAspect > pageAspect {
		// Image is wider than printable area: fit to width
		drawW = printableW
		drawH = int(float64(printableW) / imgAspect)
		drawX = margin
		drawY = margin + ((printableH - drawH) / 2)
	} else {
		// Image is taller than printable area: fit to height
		drawH = printableH
		drawW = int(float64(printableH) * imgAspect)
		drawX = margin + ((printableW - drawW) / 2)
		drawY = margin
	}

	// Scale and composite image onto white canvas
	targetRect := image.Rect(drawX, drawY, drawX+drawW, drawY+drawH)
	scaleAndDraw(canvas, targetRect, srcImg)

	// Draw Foolproof Duplex Corner Mark in Top-Right Corner if requested
	if addCornerMark {
		markCol := color.RGBA{71, 85, 105, 255} // Slate 600
		// Small corner indicator box in top right
		for y := 40; y < 90; y++ {
			for x := a4W - 120; x < a4W - 40; x++ {
				canvas.Set(x, y, markCol)
			}
		}
	}

	// Encode to clean, high-quality JPEG stream for direct printer ASIC intake
	var jpegBuf bytes.Buffer
	err = jpeg.Encode(&jpegBuf, canvas, &jpeg.Options{Quality: 95})
	if err != nil {
		return nil, fmt.Errorf("failed to encode normalized JPEG: %w", err)
	}

	return jpegBuf.Bytes(), nil
}

// Bilinear scaling into target rectangle
func scaleAndDraw(dst *image.RGBA, r image.Rectangle, src image.Image) {
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()
	dstW := r.Dx()
	dstH := r.Dy()

	white := color.RGBA{255, 255, 255, 255}

	for y := 0; y < dstH; y++ {
		srcY := srcBounds.Min.Y + (y * srcH / dstH)
		for x := 0; x < dstW; x++ {
			srcX := srcBounds.Min.X + (x * srcW / dstW)
			c := src.At(srcX, srcY)

			// Alpha blending onto white
			rVal, gVal, bVal, aVal := c.RGBA()
			if aVal == 0 {
				dst.Set(r.Min.X+x, r.Min.Y+y, white)
			} else if aVal == 0xFFFF {
				dst.Set(r.Min.X+x, r.Min.Y+y, color.RGBA{uint8(rVal >> 8), uint8(gVal >> 8), uint8(bVal >> 8), 255})
			} else {
				alpha := float64(aVal) / 65535.0
				r8 := uint8((float64(rVal>>8)*alpha + 255*(1-alpha)))
				g8 := uint8((float64(gVal>>8)*alpha + 255*(1-alpha)))
				b8 := uint8((float64(bVal>>8)*alpha + 255*(1-alpha)))
				dst.Set(r.Min.X+x, r.Min.Y+y, color.RGBA{r8, g8, b8, 255})
			}
		}
	}
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
		if statusCode > 0x00FF {
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
