package cmd

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal/output"
	"github.com/JetBrains/youtrack-cli/internal"
)

func newCanvasCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "canvas",
		Short: "Manage planning canvases",
	}
	cmd.AddCommand(newCanvasAddCmd())
	cmd.AddCommand(newCanvasGetCmd())
	cmd.AddCommand(newCanvasDeleteCmd())
	cmd.AddCommand(newCanvasReloadCmd())
	cmd.AddCommand(newCanvasNodeCmd())
	cmd.AddCommand(newCanvasLinksCmd())
	cmd.AddCommand(newCanvasLinkCmd())
	return cmd
}

func newCanvasAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a new canvas",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			ownerLogin, _ := cmd.Flags().GetString("owner")

			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}

			req := api.AddCanvasRequest{Name: name}
			if ownerLogin != "" {
				u, err := client.GetUser(ownerLogin)
				if err != nil {
					return err
				}
				req.Owner = &api.User{ID: u.ID}
			}

			canvas, err := client.AddCanvas(req)
			if err != nil {
				return err
			}
			return output.Print(cmd, canvas)
		},
	}
	cmd.Flags().String("name", "", "Canvas name (required)")
	cmd.Flags().String("owner", "", "Owner login")
	cmd.MarkFlagRequired("name")
	return cmd
}

func newCanvasGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a canvas by ID",
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

func newCanvasDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a canvas by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.DeleteCanvas(args[0])
		},
	}
}

func newCanvasReloadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reload <id>",
		Short: "Reload a canvas by ID",
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

func newCanvasNodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Manage canvas nodes",
	}
	cmd.AddCommand(newCanvasNodeAddCmd())
	return cmd
}

func newCanvasNodeAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <canvasId>",
		Short: "Add a node to a canvas",
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
			return client.AddCanvasNode(args[0], x, y, label, description)
		},
	}
	cmd.Flags().Int("x", 0, "X coordinate (required)")
	cmd.Flags().Int("y", 0, "Y coordinate (required)")
	cmd.Flags().String("label", "", "Node label")
	cmd.Flags().String("description", "", "Node description")
	cmd.MarkFlagRequired("x")
	cmd.MarkFlagRequired("y")
	return cmd
}

func newCanvasLinksCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "links <canvasId>",
		Short: "List links in a canvas",
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

func newCanvasLinkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "link <canvasId>",
		Short: "Link two nodes in a canvas",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			source, _ := cmd.Flags().GetString("source")
			target, _ := cmd.Flags().GetString("target")

			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.LinkCanvasNodes(args[0], source, target)
		},
	}
	cmd.Flags().String("source", "", "Source card ID (required)")
	cmd.Flags().String("target", "", "Target card ID (required)")
	cmd.MarkFlagRequired("source")
	cmd.MarkFlagRequired("target")
	return cmd
}
