package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/boheastill/printctl/pkg/discovery"
	"github.com/boheastill/printctl/pkg/print"
	"github.com/boheastill/printctl/pkg/scan"
	"github.com/boheastill/printctl/pkg/status"
)

const Version = "0.0.1-alpha"

func printUsage() {
	fmt.Printf(`printctl - Universal Driverless Print & Scan Engine (v%s)

"We do the dirty hardware bridge, so your apps and AI agents get instant physical I/O."

USAGE:
  printctl <command> [arguments] [flags]

CORE SURFACES:
  1. CLI Mode           Direct terminal commands for scripts, cron, and sysadmins.
  2. REST API Mode      Run 'printctl serve' to turn any machine into a printer microservice.
  3. AI Agent / Skill   Native JSON/MCP-ready endpoints for DeepSeek, Claude, Gemini, etc.

COMMANDS:
  discover              Scan local network and list all available IPP/eSCL printers
  print <file>          Print an image (PNG, JPG, WebP) or PDF via IPP Everywhere
  scan [output_file]    Scan a physical document via eSCL REST directly to JPG or PDF
  status                Query online health, scanner state, and telemetry
  serve                 Launch a lightweight local REST API microservice (for SDK integrations)
  mcp                   Run as a Model Context Protocol (MCP) server for AI Agents
  version               Display version info

FLAGS:
  --ip <address>        Printer IP address (auto-discovered on subnet if omitted)
  --port <number>       Port for REST API server in 'serve' mode (default: 8080)
  --dpi <number>        Resolution for scan/print (default: 300)
  --color <mode>        Color mode: RGB24, Grayscale8 (default: RGB24)
  --json                Output results in JSON format (ideal for scripting/AI)

EXAMPLES:
  # CLI: Auto-discover printer and print an image:
  printctl print ./photo.png

  # CLI: Scan physical document directly to A4 PDF:
  printctl scan ./invoice.pdf --dpi 300

  # REST API: Start headless printer daemon:
  printctl serve --port 8080

  # AI MCP: Run as standard MCP stdio server:
  printctl mcp
`, Version)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	cmd := os.Args[1]

	// Common Flags
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	ipFlag := fs.String("ip", "", "Printer IP address")
	portFlag := fs.Int("port", 8080, "Port for HTTP REST server")
	dpiFlag := fs.Int("dpi", 300, "Resolution in DPI (75, 150, 300, 600)")
	colorFlag := fs.String("color", "RGB24", "Color mode (RGB24, Grayscale8)")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	switch cmd {
	case "version", "-v", "--version":
		fmt.Printf("printctl v%s (pure Go, zero-driver IPP/eSCL engine)\n", Version)

	case "discover":
		devices, err := discovery.Discover(2 * time.Second)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Discovery error: %v\n", err)
			os.Exit(1)
		}
		if *jsonFlag {
			enc, _ := json.MarshalIndent(devices, "", "  ")
			fmt.Println(string(enc))
			return
		}
		if len(devices) == 0 {
			fmt.Println("[-] No printers discovered. Ensure printer is connected to same Wi-Fi.")
			return
		}
		fmt.Printf("[+] Found %d printer(s):\n", len(devices))
		for i, d := range devices {
			fmt.Printf("  [%d] %s (%s) -> IP: %s\n", i+1, d.Name, d.Model, d.IP)
			fmt.Printf("      eSCL : %s\n", d.ESCLURL)
			fmt.Printf("      IPP  : %s\n", d.IPPURL)
		}

	case "status":
		targetIP := getTargetIP(*ipFlag)
		rep, err := status.GetStatus(targetIP)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Status error: %v\n", err)
			os.Exit(1)
		}
		if *jsonFlag {
			enc, _ := json.MarshalIndent(rep, "", "  ")
			fmt.Println(string(enc))
			return
		}
		fmt.Printf("[*] Printer at %s:\n", rep.IP)
		fmt.Printf("    Online : %v\n", rep.Online)
		fmt.Printf("    Scanner: %s\n", rep.ScannerState)
		fmt.Printf("    Message: %s\n", rep.Message)

	case "scan":
		targetIP := getTargetIP(*ipFlag)
		outputFile := "scanned_output.jpg"
		args := fs.Args()
		if len(args) > 0 && args[0] != "" {
			outputFile = args[0]
		}

		opts := scan.DefaultOptions(targetIP, outputFile)
		opts.DPI = *dpiFlag
		opts.ColorMode = *colorFlag

		if !*jsonFlag {
			fmt.Printf("[*] Initiating scan on %s (%d DPI, %s) -> %s\n", targetIP, opts.DPI, opts.ColorMode, outputFile)
		}
		err := scan.ExecuteScan(opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Scan failed: %v\n", err)
			os.Exit(1)
		}
		if *jsonFlag {
			res := map[string]interface{}{
				"status": "success",
				"file":   outputFile,
				"dpi":    opts.DPI,
				"ip":     targetIP,
			}
			enc, _ := json.MarshalIndent(res, "", "  ")
			fmt.Println(string(enc))
		} else {
			fmt.Printf("[+] Scan completed successfully! Saved to %s\n", outputFile)
		}

	case "print":
		args := fs.Args()
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Error: please specify a file to print (e.g., printctl print ./document.pdf)")
			os.Exit(1)
		}
		filePath := args[0]
		targetIP := getTargetIP(*ipFlag)

		opts := print.DefaultPrintOptions(targetIP)
		if !*jsonFlag {
			fmt.Printf("[*] Sending print job to %s via IPP Everywhere: %s\n", targetIP, filePath)
		}
		err := print.PrintFile(filePath, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Print failed: %v\n", err)
			os.Exit(1)
		}
		if *jsonFlag {
			res := map[string]interface{}{
				"status": "success",
				"file":   filePath,
				"ip":     targetIP,
			}
			enc, _ := json.MarshalIndent(res, "", "  ")
			fmt.Println(string(enc))
		} else {
			fmt.Println("[+] Print job spooled successfully! Physical page output in progress.")
		}

	case "serve":
		startRESTServer(*portFlag, *ipFlag)

	case "mcp":
		startMCPServer(*ipFlag)

	case "help", "--help", "-h":
		printUsage()

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func getTargetIP(specifiedIP string) string {
	if specifiedIP != "" {
		return specifiedIP
	}
	dev, err := discovery.QuickFirst(2 * time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v (use --ip <address> to specify manually)\n", err)
		os.Exit(1)
	}
	return dev.IP
}

func startRESTServer(port int, defaultIP string) {
	mux := http.NewServeMux()

	// GET /api/discover
	mux.HandleFunc("/api/discover", func(w http.ResponseWriter, r *http.Request) {
		devices, _ := discovery.Discover(2 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"devices": devices,
		})
	})

	// GET /api/status?ip=...
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		ip := r.URL.Query().Get("ip")
		if ip == "" {
			ip = defaultIP
		}
		if ip == "" {
			dev, _ := discovery.QuickFirst(1 * time.Second)
			if dev != nil {
				ip = dev.IP
			}
		}
		rep, err := status.GetStatus(ip)
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(rep)
	})

	// POST /api/scan?ip=...&dpi=300&format=jpeg
	mux.HandleFunc("/api/scan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ip := r.URL.Query().Get("ip")
		if ip == "" {
			ip = defaultIP
		}
		if ip == "" {
			dev, _ := discovery.QuickFirst(1 * time.Second)
			if dev != nil {
				ip = dev.IP
			}
		}
		dpiStr := r.URL.Query().Get("dpi")
		dpi := 300
		if val, err := strconv.Atoi(dpiStr); err == nil && val > 0 {
			dpi = val
		}
		format := r.URL.Query().Get("format")
		if format == "" {
			format = "jpeg"
		}

		tmpFile, err := os.CreateTemp("", "printctl_scan_*."+format)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpPath := tmpFile.Name()
		tmpFile.Close()
		defer os.Remove(tmpPath)

		opts := scan.DefaultOptions(ip, tmpPath)
		opts.DPI = dpi
		opts.Format = format

		err = scan.ExecuteScan(opts)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		data, err := os.ReadFile(tmpPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if format == "pdf" {
			w.Header().Set("Content-Type", "application/pdf")
		} else {
			w.Header().Set("Content-Type", "image/jpeg")
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.Write(data)
	})

	// POST /api/print?ip=...
	mux.HandleFunc("/api/print", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		ip := r.URL.Query().Get("ip")
		if ip == "" {
			ip = defaultIP
		}
		if ip == "" {
			dev, _ := discovery.QuickFirst(1 * time.Second)
			if dev != nil {
				ip = dev.IP
			}
		}

		fileData, err := io.ReadAll(r.Body)
		if err != nil || len(fileData) == 0 {
			http.Error(w, "Empty file body", http.StatusBadRequest)
			return
		}

		tmpFile, err := os.CreateTemp("", "printctl_upload_*")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpPath := tmpFile.Name()
		os.WriteFile(tmpPath, fileData, 0644)
		tmpFile.Close()
		defer os.Remove(tmpPath)

		opts := print.DefaultPrintOptions(ip)
		err = print.PrintFile(tmpPath, opts)
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"message": "Job spooled to printer successfully",
			"ip":      ip,
		})
	})

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("[*] printctl REST API daemon listening on http://127.0.0.1%s\n", addr)
	fmt.Println("    Endpoints:")
	fmt.Println("      GET  /api/discover       - List network printers")
	fmt.Println("      GET  /api/status         - Hardware & telemetry state")
	fmt.Println("      GET  /api/scan           - Optical scan and stream back JPG/PDF")
	fmt.Println("      POST /api/print          - Post binary image/PDF to physically print")
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "Server failed: %v\n", err)
	}
}

// startMCPServer provides stdio JSON-RPC MCP server for AI Agents (DeepSeek, Claude, Gemini)
func startMCPServer(defaultIP string) {
	fmt.Fprintf(os.Stderr, "[*] printctl MCP (Model Context Protocol) Server active on stdio\n")
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		var req map[string]interface{}
		if err := decoder.Decode(&req); err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		method, _ := req["method"].(string)
		id := req["id"]

		switch method {
		case "tools/list":
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]interface{}{
					"tools": []map[string]interface{}{
						{
							"name":        "printer_scan",
							"description": "Trigger the physical optical scanner to scan a paper document into a JPG or PDF file.",
							"inputSchema": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"output_path": map[string]interface{}{
										"type":        "string",
										"description": "Absolute file path to save the scanned image or PDF (e.g. /tmp/scanned.jpg)",
									},
									"dpi": map[string]interface{}{
										"type":        "integer",
										"description": "Scan resolution: 75, 150, 300, 600. Default: 300",
									},
								},
								"required": []string{"output_path"},
							},
						},
						{
							"name":        "printer_print",
							"description": "Physically print an image (PNG, JPG, WebP) or PDF file directly onto paper with automatic safe-margin layout.",
							"inputSchema": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"file_path": map[string]interface{}{
										"type":        "string",
										"description": "Absolute file path of the image or PDF to print",
									},
								},
								"required": []string{"file_path"},
							},
						},
						{
							"name":        "printer_status",
							"description": "Query printer online status, scanner idle state, and telemetry.",
							"inputSchema": map[string]interface{}{
								"type": "object",
							},
						},
					},
				},
			}
			encoder.Encode(resp)

		case "tools/call":
			params, _ := req["params"].(map[string]interface{})
			name, _ := params["name"].(string)
			args, _ := params["arguments"].(map[string]interface{})

			var resultText string
			var isErr bool

			targetIP := defaultIP
			if targetIP == "" {
				dev, _ := discovery.QuickFirst(1 * time.Second)
				if dev != nil {
					targetIP = dev.IP
				}
			}

			switch name {
			case "printer_scan":
				outPath, _ := args["output_path"].(string)
				if outPath == "" {
					outPath = "/tmp/printctl_scanned.jpg"
				}
				dpi := 300
				if d, ok := args["dpi"].(float64); ok && d > 0 {
					dpi = int(d)
				}
				opts := scan.DefaultOptions(targetIP, outPath)
				opts.DPI = dpi
				if err := scan.ExecuteScan(opts); err != nil {
					resultText = fmt.Sprintf("Scan failed: %v", err)
					isErr = true
				} else {
					resultText = fmt.Sprintf("Successfully scanned paper to: %s (%d DPI)", outPath, dpi)
				}

			case "printer_print":
				filePath, _ := args["file_path"].(string)
				opts := print.DefaultPrintOptions(targetIP)
				if err := print.PrintFile(filePath, opts); err != nil {
					resultText = fmt.Sprintf("Print failed: %v", err)
					isErr = true
				} else {
					resultText = fmt.Sprintf("Successfully spooled print job for file: %s", filePath)
				}

			case "printer_status":
				rep, err := status.GetStatus(targetIP)
				if err != nil {
					resultText = fmt.Sprintf("Status error: %v", err)
					isErr = true
				} else {
					b, _ := json.Marshal(rep)
					resultText = string(b)
				}

			default:
				resultText = "Unknown tool: " + name
				isErr = true
			}

			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]interface{}{
					"content": []map[string]interface{}{
						{
							"type": "text",
							"text": resultText,
						},
					},
					"isError": isErr,
				},
			}
			encoder.Encode(resp)

		default:
			// Ping / initialize / notifications
			encoder.Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      id,
				"result":  map[string]interface{}{"status": "ready"},
			})
		}
	}
}
