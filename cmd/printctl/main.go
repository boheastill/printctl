package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/boheastill/printctl/pkg/discovery"
	"github.com/boheastill/printctl/pkg/print"
	"github.com/boheastill/printctl/pkg/scan"
	"github.com/boheastill/printctl/pkg/status"
)

const Version = "0.0.1-alpha"

func printUsage() {
	fmt.Printf(`printctl - The Modern Driverless Print & Scan Engine in Pure Go (v%s)

USAGE:
  printctl <command> [arguments] [flags]

COMMANDS:
  discover              Scan local network and list all available IPP/eSCL printers
  print <file>          Print an image (PNG, JPG, WebP) or PDF to printer via IPP Everywhere
  scan [output_file]    Scan a document/photo via eSCL REST and save to JPG or PDF
  status                Query online health and scanner state from printer
  version               Display version info

FLAGS:
  --ip <address>        Printer IP address (auto-discovered on local subnet if omitted)
  --dpi <number>        Resolution for scan/print (default: 300)
  --color <mode>        Color mode: RGB24, Grayscale8 (default: RGB24)
  --json                Output results in JSON format (for scripting/AI agents)

EXAMPLES:
  # Auto-discover printer on Wi-Fi and print an image:
  printctl print ./photo.png

  # Scan document directly to A4 PDF with 300 DPI:
  printctl scan ./invoice.pdf

  # Scan high-res photo at 600 DPI:
  printctl scan ./photo_600dpi.jpg --dpi 600

  # Query status with JSON output:
  printctl status --json
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
		fmt.Println("[*] Scanning local network for IPP/eSCL printers...")
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
		targetIP := *ipFlag
		if targetIP == "" {
			dev, err := discovery.QuickFirst(2 * time.Second)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v (use --ip to specify)\n", err)
				os.Exit(1)
			}
			targetIP = dev.IP
		}
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
		targetIP := *ipFlag
		if targetIP == "" {
			dev, err := discovery.QuickFirst(2 * time.Second)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v (use --ip to specify)\n", err)
				os.Exit(1)
			}
			targetIP = dev.IP
		}

		outputFile := "scanned_output.jpg"
		args := fs.Args()
		if len(args) > 0 && args[0] != "" {
			outputFile = args[0]
		}

		opts := scan.DefaultOptions(targetIP, outputFile)
		opts.DPI = *dpiFlag
		opts.ColorMode = *colorFlag

		fmt.Printf("[*] Initiating scan on %s (%d DPI, %s) -> %s\n", targetIP, opts.DPI, opts.ColorMode, outputFile)
		err := scan.ExecuteScan(opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Scan failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[+] Scan completed successfully! Saved to %s\n", outputFile)

	case "print":
		args := fs.Args()
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Error: please specify a file to print (e.g., printctl print ./document.pdf)")
			os.Exit(1)
		}
		filePath := args[0]

		targetIP := *ipFlag
		if targetIP == "" {
			dev, err := discovery.QuickFirst(2 * time.Second)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v (use --ip to specify)\n", err)
				os.Exit(1)
			}
			targetIP = dev.IP
		}

		opts := print.DefaultPrintOptions(targetIP)
		fmt.Printf("[*] Sending print job to %s via IPP Everywhere: %s\n", targetIP, filePath)
		err := print.PrintFile(filePath, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Print failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("[+] Print job spooled successfully! Physical page output in progress.")

	case "help", "--help", "-h":
		printUsage()

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}
