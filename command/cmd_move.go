package command

import (
	"fmt"

	"github.com/peak/s5cmd/storage"
	"github.com/urfave/cli/v2"
)

var MoveCommand = &cli.Command{
	Name:     "mv",
	HelpName: "move",
	Usage:    "TODO",
	Flags:    append(copyCommandFlags, globalFlags...), // move and copy commands share the same flags
	Before: func(c *cli.Context) error {
		validate := func() error {
			if err := validateGlobalFlags(c); err != nil {
				return err
			}

			if c.Args().Len() != 2 {
				return fmt.Errorf("expected source and destination arguments")
			}
			return nil
		}
		if err := validate(); err != nil {
			printError(givenCommand(c), c.Command.Name, err)
			return err
		}

		setGlobalFlags(c)
		return nil
	},
	Action: func(c *cli.Context) error {
		noClobber := c.Bool("no-clobber")
		ifSizeDiffer := c.Bool("if-size-differ")
		ifSourceNewer := c.Bool("if-source-newer")
		recursive := c.Bool("recursive")
		parents := c.Bool("parents")
		storageClass := storage.LookupClass(c.String("storage-class"))

		err := Copy(
			c.Context,
			c.Args().Get(0),
			c.Args().Get(1),
			c.Command.Name,
			true, // delete source
			// flags
			noClobber,
			ifSizeDiffer,
			ifSourceNewer,
			recursive,
			parents,
			storageClass,
		)
		if err != nil {
			printError(givenCommand(c), c.Command.Name, err)
			return err
		}

		return nil
	},
}
