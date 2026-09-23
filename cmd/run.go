// Package cmd wires flags to the read-only model and, later, the TUI.
package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/signalandform/constelate/internal/app"
	"github.com/signalandform/constelate/internal/ui"
)

// Options are the parsed command-line flags.
func Run(args []string) error {
	fs := flag.NewFlagSet("constelate", flag.ContinueOnError)
	var o app.Options
	fs.StringVar(&o.Project, "project", "", "project directory (default: current directory)")
	fs.BoolVar(&o.DryRun, "dry-run", false, "show every write without making it")
	fs.BoolVar(&o.Summary, "summary", false, "print a read-only summary and exit")
	fs.BoolVar(&o.JSON, "json", false, "with --summary, print JSON")
	var render string
	fs.StringVar(&render, "render", "", "print one frame and exit: WxH followed by keys, e.g. 80x24+tab+enter")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if o.Project == "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		o.Project = wd
	}
	o.Project, _ = filepath.Abs(o.Project)

	st, err := app.Load(o)
	if err != nil {
		return err
	}
	if o.Summary || o.JSON {
		return PrintSummary(os.Stdout, st)
	}
	if render != "" {
		var w, h int
		parts := strings.Split(render, "+")
		if _, err := fmt.Sscanf(parts[0], "%dx%d", &w, &h); err != nil {
			return fmt.Errorf("--render wants WxH[+key...], got %q", render)
		}
		fmt.Println(ui.RenderOnce(st, w, h, parts[1:]))
		return nil
	}
	return ui.Run(st)
}
