// Command config-hot-reload 跑配置热更样例。
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"example.com/hotreload"
)

// Run 执行一次命令行调用，返回退出码。
func Run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("config-hot-reload", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sample := flags.String("sample", "stale", "stale / atomic / rollback / scan")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	values := map[string]int{"tick": 30}
	switch {
	case *sample == "stale":
		store := hotreload.NewStore()
		store.Publish(2, values)
		ok, reason := store.Publish(1, map[string]int{"tick": 10})
		fmt.Fprintf(stdout, "ok=%v reason=%s version=%d\n", ok, reason, store.Current().Version)
	case *sample == "atomic":
		store := hotreload.NewStore()
		store.Publish(1, values)
		values["tick"] = 99
		fmt.Fprintf(stdout, "tick=%d\n", store.Current().Values["tick"])
	case *sample == "rollback":
		store := hotreload.NewStore()
		store.Publish(1, map[string]int{"tick": 20})
		store.Publish(2, map[string]int{"tick": 30})
		ok := store.Rollback()
		fmt.Fprintf(stdout, "ok=%v version=%d\n", ok, store.Current().Version)
	case *sample == "scan":
		store := hotreload.NewStore()
		for index := 1; index <= 3000; index++ {
			store.Publish(index, map[string]int{"tick": index})
		}
		for query := 0; query < 3000; query++ {
			store.Published()
		}
		fmt.Fprintf(stdout, "scanned=%d\n", store.Scanned())
	default:
		fmt.Fprintln(stderr, "需要 --sample stale|atomic|rollback|scan")
		return 2
	}
	return 0
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
