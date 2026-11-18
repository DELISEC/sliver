package cli

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	clientmcp "github.com/bishopfox/sliver/client/mcp"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/keepalive"
)

func mcpCmd(con *console.SliverClient) *cobra.Command {
	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start the MCP server (stdio or HTTP)",
		Long: `Start a Model Context Protocol server that exposes Sliver tools to AI agents.

By default starts in stdio mode (for Claude Code stdio transport).
Use --transport http to start an HTTP server (for multiplayer / remote agents).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCP(cmd, con)
		},
	}
	mcpCmd.Flags().String("transport", "stdio", "Transport mode: stdio, http, or sse")
	mcpCmd.Flags().String("listen", "127.0.0.1:8080", "Listen address for HTTP/SSE transport")
	mcpCmd.Flags().String("zergling-path", "", "Path to zergling packer binary (enables generate_packed_implant)")
	mcpCmd.Flags().String("bof-modules-dir", "", "Directory containing BOF modules")
	return mcpCmd
}

func runMCP(cmd *cobra.Command, con *console.SliverClient) error {
	if err := applyMultiplayerConnectMode(cmd); err != nil {
		return err
	}

	configPath, _ := cmd.Flags().GetString("config")
	config, err := loadMCPClientConfig(configPath)
	if err != nil {
		fmt.Printf("%s\n", err)
		return nil
	}

	// Keep the gRPC link alive while the MCP bridge sits idle between tool calls.
	// The daemon permits pings >=30s without streams (server.yaml: grpc.keepalive).
	transport.SetKeepaliveParams(keepalive.ClientParameters{
		Time:                60 * time.Second,
		Timeout:             20 * time.Second,
		PermitWithoutStream: true,
	})

	rpc, ln, err := transport.MTLSConnect(config)
	if err != nil {
		fmt.Printf("Connection to server failed %s\n", err)
		return nil
	}
	defer transport.CloseGRPCConnection(ln)

	con.Rpc = rpc

	zerglingPath, _ := cmd.Flags().GetString("zergling-path")
	bofModulesDir, _ := cmd.Flags().GetString("bof-modules-dir")

	rawTransport, _ := cmd.Flags().GetString("transport")
	if rawTransport == "stdio" || rawTransport == "" {
		cfg := clientmcp.DefaultConfig()
		cfg.ZerglingPath = zerglingPath
		cfg.BOFModulesDir = bofModulesDir
		if err := clientmcp.ServeStdio(cfg, con.Rpc); err != nil {
			fmt.Printf("MCP server error: %s\n", err)
			return nil
		}
		return nil
	}

	mcpTransport, err := clientmcp.ParseTransport(rawTransport)
	if err != nil {
		fmt.Printf("Invalid transport: %s\n", err)
		return nil
	}

	listen, _ := cmd.Flags().GetString("listen")
	cfg := clientmcp.Config{
		Transport:     mcpTransport,
		ListenAddress: listen,
		ZerglingPath:  zerglingPath,
		BOFModulesDir: bofModulesDir,
	}.WithDefaults()

	if err := clientmcp.Start(cfg, con.Rpc); err != nil {
		fmt.Printf("MCP server start failed: %s\n", err)
		return nil
	}

	status := clientmcp.GetStatus()
	endpoint, _ := cfg.EndpointURL()
	fmt.Printf("MCP server running (%s) on %s\n", cfg.Transport, cfg.ListenAddress)
	fmt.Printf("Endpoint: %s\n", endpoint)
	if status.AuthToken != "" {
		fmt.Printf("Auth header: %s\n", status.AuthHeader)
		fmt.Printf("Auth token: %s\n", status.AuthToken)
	}
	fmt.Printf("Press Ctrl+C to stop\n")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Printf("\nShutting down...\n")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := clientmcp.Stop(ctx); err != nil {
		fmt.Printf("Shutdown error: %s\n", err)
	}
	return nil
}

func loadMCPClientConfig(path string) (*assets.ClientConfig, error) {
	if path != "" {
		configPath := path
		if !filepath.IsAbs(configPath) {
			if _, err := os.Stat(configPath); err != nil {
				configPath = filepath.Join(assets.GetConfigDir(), configPath)
			}
		}
		return assets.ReadConfig(configPath)
	}

	configs := assets.GetConfigs()
	switch len(configs) {
	case 0:
		return nil, fmt.Errorf("no config files found at %s (use --config)", assets.GetConfigDir())
	case 1:
		for _, config := range configs {
			return config, nil
		}
	default:
		return nil, fmt.Errorf("multiple configs found; use --config to select one")
	}
	return nil, fmt.Errorf("no config files found")
}
