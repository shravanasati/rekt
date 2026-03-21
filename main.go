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

// todo verify udp and tcp6
// todo rekt 3000 -k -> kill process
// todo rekt 3000 -t -> terminate process
// todo rekt list -> list all processes occupying a port
// todo handle edge cases like SO_REUSEPORT

func main() {
	cmd := &cli.Command{
		Name:  "rekt",
		Usage: "slay the evil pid holding your port hostage",
		Arguments: []cli.Argument{
			&cli.IntArg{
				Name: "port",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			port := c.IntArg("port")
			if port == 0 {
				return errPortRequired
			}

			if port < 1 || port > 65535 {
				return errInvalidPort
			}
			pf := internal.NewProcessFinder()
			pid, err := (pf.FindPIDByPort(port))
			if err != nil {
				return err
			}
			fmt.Printf("port %v -> pid %v\n", port, pid)
			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

}
