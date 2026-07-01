package cmd

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newWhiteboardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "whiteboard",
		Short: "Manage whiteboards",
	}
	cmd.AddCommand(newWhiteboardListCmd())
	cmd.AddCommand(newWhiteboardAddCmd())
	cmd.AddCommand(newWhiteboardGetCmd())
	cmd.AddCommand(newWhiteboardDeleteCmd())
	cmd.AddCommand(newWhiteboardReloadCmd())
	cmd.AddCommand(newWhiteboardNodeCmd())
	cmd.AddCommand(newWhiteboardLinksCmd())
	cmd.AddCommand(newWhiteboardLinkCmd())
	return cmd
}

func newWhiteboardListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all whiteboards",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			canvases, err := client.ListCanvases()
			if err != nil {
				return err
			}
			return output.Print(cmd, canvases)
		},
	}
}

func newWhiteboardAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a new whiteboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			ownerLogin, _ := cmd.Flags().GetString("owner")

			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}

			req := api.AddCanvasRequest{Name: name}
			ownerTarget := ownerLogin
			if ownerTarget == "" {
				me, err := client.GetCurrentUser()
				if err != nil {
					return err
				}
				ownerTarget = me.Login
			}
			u, err := client.GetUser(ownerTarget)
			if err != nil {
				return err
			}
			req.Owner = &api.User{ID: u.ID}

			canvas, err := client.AddCanvas(req)
			if err != nil {
				return err
			}
			return output.Print(cmd, canvas)
		},
	}
	cmd.Flags().String("name", "", "Whiteboard name (required)")
	cmd.Flags().String("owner", "", "Owner login")
	internal.MustMarkRequired(cmd, "name")
	return cmd
}

func newWhiteboardGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a whiteboard by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			canvas, err := client.GetCanvas(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, canvas)
		},
	}
}

func newWhiteboardDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a whiteboard by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteCanvas(args[0]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "deleted whiteboard "+args[0])
			return nil
		},
	}
}

func newWhiteboardReloadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reload <id>",
		Short: "Reload a whiteboard by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.ReloadCanvas(args[0])
		},
	}
}

func newWhiteboardNodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Manage whiteboard nodes",
	}
	cmd.AddCommand(newWhiteboardNodeAddCmd())
	cmd.AddCommand(newWhiteboardNodeListCmd())
	cmd.AddCommand(newWhiteboardNodeGetCmd())
	return cmd
}

func newWhiteboardNodeListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <whiteboardId>",
		Short: "List nodes on a whiteboard",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			nodes, err := client.ListCanvasCards(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, nodes)
		},
	}
}

func newWhiteboardNodeGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <whiteboardId> <cardId>",
		Short: "Get a node from a whiteboard",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			node, err := client.GetCanvasCard(args[0], args[1])
			if err != nil {
				return err
			}
			return output.Print(cmd, node)
		},
	}
}

func newWhiteboardNodeAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <whiteboardId>",
		Short: "Add a node to a whiteboard",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			x, _ := cmd.Flags().GetInt("x")
			y, _ := cmd.Flags().GetInt("y")
			label, _ := cmd.Flags().GetString("label")
			description, _ := cmd.Flags().GetString("description")

			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			node, err := client.AddCanvasNode(args[0], x, y, label, description)
			if err != nil {
				return err
			}
			return output.Print(cmd, node)
		},
	}
	cmd.Flags().Int("x", 0, "X coordinate (required)")
	cmd.Flags().Int("y", 0, "Y coordinate (required)")
	cmd.Flags().String("label", "", "Node label")
	cmd.Flags().String("description", "", "Node description")
	internal.MustMarkRequired(cmd, "x", "y")
	return cmd
}

func newWhiteboardLinksCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "links <whiteboardId>",
		Short: "List links in a whiteboard",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			links, err := client.GetCanvasLinks(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, links)
		},
	}
}

func newWhiteboardLinkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "link <whiteboardId>",
		Short: "Link two nodes in a whiteboard",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			source, _ := cmd.Flags().GetString("source")
			target, _ := cmd.Flags().GetString("target")
			linkType, _ := cmd.Flags().GetString("link-type")
			proto, _ := cmd.Flags().GetString("link-prototype")

			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.LinkCanvasNodes(args[0], source, target, api.CanvasLinkKind(linkType), proto)
		},
	}
	cmd.Flags().String("source", "", "Source card ID (required)")
	cmd.Flags().String("target", "", "Target card ID (required)")
	cmd.Flags().String("link-type", "ONE_WAY", "Link type: ONE_WAY, TWO_WAY, PROTOTYPED")
	cmd.Flags().String("link-prototype", "", "Issue link type ID (required for PROTOTYPED links)")
	internal.MustMarkRequired(cmd, "source", "target")
	return cmd
}
