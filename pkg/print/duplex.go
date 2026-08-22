package print

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

// DuplexOptions holds configuration for manual two-sided printing
type DuplexOptions struct {
	PrintOpts PrintOptions
	Manual    bool
	Sides     string // "one-sided", "two-sided-long-edge", "two-sided-short-edge"
}

// PrintDuplexImages orchestrates foolproof two-sided printing for a list of image/page files
func PrintDuplexImages(pageFiles []string, opts DuplexOptions) error {
	total := len(pageFiles)
	if total == 0 {
		return fmt.Errorf("no pages to print")
	}

	if total == 1 || opts.Sides == "one-sided" {
		// Single page or one-sided: print normally
		for _, f := range pageFiles {
			if err := PrintFile(f, opts.PrintOpts); err != nil {
				return err
			}
		}
		return nil
	}

	// 1. Separate into Odd (Front) and Even (Back) batches
	var oddFiles []string
	var evenFiles []string

	for i, f := range pageFiles {
		pageNum := i + 1
		if pageNum%2 != 0 {
			oddFiles = append(oddFiles, f)
		} else {
			evenFiles = append(evenFiles, f)
		}
	}

	// 2. Print Odd Batch (Front sides)
	fmt.Printf("[*] 正在打印【正面（奇数页）批次】共 %d 页...\n", len(oddFiles))
	for idx, f := range oddFiles {
		pageNum := (idx * 2) + 1
		fmt.Printf("    -> 正在打印第 %d 页: %s\n", pageNum, f)
		if err := PrintFile(f, opts.PrintOpts); err != nil {
			return fmt.Errorf("failed printing odd page %d: %w", pageNum, err)
		}
		time.Sleep(3 * time.Second) // wait between spooling
	}

	// 3. Display Foolproof Manual Flip Guide in Terminal
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║               📄 【防呆双面翻面指引 (Manual Duplex Guide)】            ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════════════╣")
	fmt.Println("║  ✅ 第一批（正面）已全部打印完毕！                                   ║")
	fmt.Println("║                                                                      ║")
	fmt.Println("║  【长边翻转 (Long-Edge) 3 步翻面操作】：                             ║")
	fmt.Println("║  1. 从出纸托盘中【整叠拿起】刚才打印出来的所有纸张（不要打乱顺序）； ║")
	fmt.Println("║  2. 观察纸张右上角防呆标记【★ / ↖ 印记】；                          ║")
	fmt.Println("║  3. 将整叠纸【翻转 180°（正面朝下）】，保持【印记角朝上】放入后部进纸槽；║")
	fmt.Println("║                                                                      ║")
	fmt.Println("║  放好纸张后，请在下方按【回车键 (Enter)】开始打印背面（偶数页）...    ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════╝")
	fmt.Print("👉 纸张已放好，按 [Enter] 继续: ")

	// Wait for user confirmation in interactive terminal
	bufio.NewReader(os.Stdin).ReadBytes('\n')

	// 4. Print Even Batch (Back sides) in reverse order for rear tray
	fmt.Printf("\n[*] 正在打印【背面（偶数页）批次】共 %d 页...\n", len(evenFiles))
	for idx, f := range evenFiles {
		pageNum := (idx + 1) * 2
		fmt.Printf("    -> 正在打印第 %d 页 (背面): %s\n", pageNum, f)
		if err := PrintFile(f, opts.PrintOpts); err != nil {
			return fmt.Errorf("failed printing even page %d: %w", pageNum, err)
		}
		time.Sleep(3 * time.Second)
	}

	fmt.Println("\n🎉 双面打印全部大功告成！请检查正反面装订效果。")
	return nil
}
