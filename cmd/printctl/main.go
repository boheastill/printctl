package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/boheastill/printctl/pkg/discovery"
	"github.com/boheastill/printctl/pkg/print"
	"github.com/boheastill/printctl/pkg/scan"
	"github.com/boheastill/printctl/pkg/status"
)

const Version = "0.0.2-patch"

func printUsage() {
	fmt.Printf(`printctl - Universal Driverless Print & Scan Engine (v%s)

"We do the dirty hardware bridge, so your apps and AI agents get instant physical I/O."

USAGE:
  printctl <command> [arguments] [flags]

COMMANDS:
  discover              Scan local network and list all available IPP/eSCL printers
  print <file...>       Print an image (PNG, JPG, WebP) or PDF via IPP Everywhere
  scan [output_file]    Scan a physical document via eSCL REST directly to JPG or PDF
  status                Query online health, scanner state, and telemetry
  version               Display version info

FLAGS:
  --ip <address>        Printer IP address (auto-discovered on subnet if omitted)
  --dpi <number>        Resolution for scan/print (default: 300)
  --color <mode>        Color mode: RGB24, Grayscale8 (default: RGB24)
  --sides <mode>        Sides mode: one-sided, two-sided-long-edge, two-sided-short-edge
  --duplex              Enable manual foolproof two-sided printing (long-edge)
  --json                Output results in JSON format (ideal for scripting/AI)
`, Version)
}

func getTargetIP(flagIP string) string {
	if flagIP != "" {
		return flagIP
	}
	devices, err := discovery.Discover(2 * time.Second)
	if err != nil || len(devices) == 0 {
		return "127.0.0.1"
	}
	return devices[0].IP
}

// normalizeArgs reorders arguments so flags (-flag, --flag) precede positional arguments,
// preventing Go's standard flag package from prematurely stopping argument parsing.
func normalizeArgs(args []string) []string {
	var flags []string
	var posArgs []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			posArgs = append(posArgs, args[i+1:]...)
			break
		}
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			if !strings.Contains(arg, "=") {
				cleanName := strings.TrimLeft(arg, "-")
				if cleanName != "json" && cleanName != "duplex" && cleanName != "v" && cleanName != "version" && cleanName != "help" && cleanName != "h" {
					if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
						flags = append(flags, args[i+1])
						i++
					}
				}
			}
		} else {
			posArgs = append(posArgs, arg)
		}
	}
	return append(flags, posArgs...)
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
	duplexFlag := fs.Bool("duplex", false, "Enable manual foolproof two-sided printing (long-edge)")
	sidesFlag := fs.String("sides", "one-sided", "Sides mode: one-sided, two-sided-long-edge, two-sided-short-edge")
	paperFlag := fs.String("paper", "A4", "Paper size: A4, 4x6 (photo)")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	// Parse with normalized argument order
	normalized := normalizeArgs(os.Args[2:])
	if err := fs.Parse(normalized); err != nil {
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
		targetIP := getTargetIP(*ipFlag)
		opts := print.DefaultPrintOptions(targetIP)
		opts.PaperSize = *paperFlag

		sidesMode := *sidesFlag
		if *duplexFlag {
			sidesMode = "two-sided-long-edge"
		}

		if len(args) > 1 || sidesMode != "one-sided" {
			// Multi-page or duplex workflow
			duplexOpts := print.DuplexOptions{
				PrintOpts: opts,
				Manual:    true,
				Sides:     sidesMode,
			}
			err := print.PrintDuplexImages(args, duplexOpts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Duplex print failed: %v\n", err)
				os.Exit(1)
			}
		} else {
			filePath := args[0]
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
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}
