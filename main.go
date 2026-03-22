package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/shravanasati/rekt/internal"
	"github.com/urfave/cli/v3"
)

var errPortRequired = errors.New("port argument is required")
var errInvalidPort = errors.New("port must lie between 1 and 65535")

func exitError(e error) error {
	return cli.Exit(e, 1)
}

// todo rekt list -> list all processes occupying a port

func main() {
	cmd := &cli.Command{
		Name:  "rekt",
		Usage: "slay the evil pid holding your port hostage",
		Arguments: []cli.Argument{
			&cli.IntArg{
				Name: "port",
			},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "kill", Aliases: []string{"k"}},
			&cli.BoolFlag{Name: "terminate", Aliases: []string{"t"}},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			port := c.IntArg("port")
			if port == 0 {
				return exitError(errPortRequired)
			}

			if port < 1 || port > 65535 {
				return exitError(errInvalidPort)
			}

			pf := internal.NewProcessFinder()
			pids, err := (pf.FindPIDByPort(port))
			if err != nil {
				return exitError(err)
			}

			// if len(pids) > 1 {
			// 	fmt.Printf("detected %d processes occupying %d, likely SO_REUSEPORT\n", len(pids), port)
			// }
			for _, pid := range pids {
				fmt.Printf("port %v -> pid %v\n", port, pid)
			}

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
			switch killmode {
			case internal.ModeTerm:
				for _, pid := range pids {
					err := ps.TermProcess(pid)
					if err != nil {
						slayErrors = append(slayErrors, err)
					}
				}
			case internal.ModeKill:
				for _, pid := range pids {
					err := ps.KillProcess(pid)
					if err != nil {
						slayErrors = append(slayErrors, err)
					}
				}
			}

			err = errors.Join(slayErrors...)
			if err != nil {
				return exitError(err)
			}
			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

}
