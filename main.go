package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/shravanasati/rekt/internal"
	"github.com/urfave/cli/v3"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var errPortRequired = errors.New("port argument is required")
var errInvalidPort = errors.New("port must lie between 1 and 65535")

func exitError(e error) error {
	return cli.Exit(e, 1)
}

func main() {
	cmd := &cli.Command{
		Name:                   "rekt",
		Usage:                  "slay the evil process holding your port hostage",
		Description:            "Find the process occupying a port\n$ rekt 8000 (-v for verbose output)\n\nKill or terminate the process\n$ rekt 8000 -k (or -t)",
		UseShortOptionHandling: true,
		Arguments: []cli.Argument{
			&cli.IntArg{
				Name:      "port",
				UsageText: "The port to free.",
			},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "kill", Aliases: []string{"k"}, Usage: "Force kill the process."},
			&cli.BoolFlag{Name: "terminate", Aliases: []string{"t"}, Usage: "Terminate the process (has same behavior as kill on Windows)."},
			&cli.BoolFlag{Name: "verbose", Aliases: []string{"v"}, Usage: "Show verbose output."},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			port := c.IntArg("port")
			if port == 0 {
				return exitError(errPortRequired)
			}

			if port < 1 || port > 65535 {
				return exitError(errInvalidPort)
			}

			verbose := c.Bool("verbose")

			pf := internal.NewProcessFinder()
			processes, err := (pf.FindPIDByPort(port, verbose))
			if err != nil {
				return exitError(err)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			if verbose {
				fmt.Fprintln(w, "PORT\tPID\tNAME\tTYPE\tUSER")
				for _, process := range processes {
					fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\n", port, process.PID, process.Name, process.Type, process.User)
				}
			} else {
				fmt.Fprintln(w, "PORT\tPID")
				for _, process := range processes {
					fmt.Fprintf(w, "%v\t%v\n", port, process.PID)
				}
			}
			w.Flush()

			killFlag := c.Bool("kill")
			terminateFlag := c.Bool("terminate")
			if !killFlag && !terminateFlag {
				return nil
			}

			killmode := internal.ModeTerm
			if killFlag {
				killmode = internal.ModeKill
			}

			ps := internal.NewProcessSlayer()
			slayErrors := []error{}
			var slayMethod func(int) error
			switch killmode {
			case internal.ModeTerm:
				slayMethod = ps.TermProcess
			case internal.ModeKill:
				slayMethod = ps.KillProcess
			}

			for _, pid := range processes {
				err := slayMethod(pid.PID)
				if err != nil {
					slayErrors = append(slayErrors, err)
				}
			}

			err = errors.Join(slayErrors...)
			if err != nil {
				return exitError(err)
			}
			return nil
		},

		Commands: []*cli.Command{{
			Name:  "version",
			Usage: "Print version information.",
			Action: func(ctx context.Context, c *cli.Command) error {
				fmt.Printf("rotom %s (commit: %s, built at: %s)\n", version, commit, date)
				return nil
			},
		}},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

}
