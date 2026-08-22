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
	fmt.Println("║               📄 【防呆双面二次放纸指引 (3秒自检放纸)】              ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════════════╣")
	fmt.Println("║  ✅ 第一批（正面）已全部打印完毕！                                   ║")
	fmt.Println("║                                                                      ║")
	fmt.Println("║  【只需 1 个动作放回进纸槽】：                                       ║")
	fmt.Println("║  1. 从出纸托盘中整叠拿起刚才的纸张，翻到【背面空白朝上】；           ║")
	fmt.Println("║  2. 找到纸张边缘印有【 ⬇ 向下箭头标记 】的那一端；                   ║")
	fmt.Println("║  3. 将【 ⬇ 箭头端朝下】直接插入后部进纸槽最深处；                    ║")
	fmt.Println("║                                                                      ║")
	fmt.Println("║  🔍【物理自检口诀】：纸插好后，如果【 ⬇ 箭头】被插槽吞没看不见，就是对的！║")
	fmt.Println("║                                                                      ║")
	fmt.Println("║  放好后，请在下方按【回车键 (Enter)】开始打印背面（偶数页）...        ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════╝")
	fmt.Print("👉 箭头已沉入插槽看不见，按 [Enter] 继续: ")

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
