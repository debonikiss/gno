package main

import (
	"fmt"
	"os"

	"github.com/gnolang/gno/pkgs/amino"
	"github.com/gnolang/gno/pkgs/bft/rpc/client"
	"github.com/gnolang/gno/pkgs/command"
	"github.com/gnolang/gno/pkgs/std"

	// XXX better way?
	_ "github.com/gnolang/gno/pkgs/sdk/auth"
	_ "github.com/gnolang/gno/pkgs/sdk/bank"
	_ "github.com/gnolang/gno/pkgs/sdk/vm"
)

type txExportOptions struct {
	Remote      string `flag:"remote" help:"Remote RPC addr:port"`
	StartHeight int64  `flag:"start" help:"Start height"`
	EndHeight   int64  `flag:"end" help:"End height (optional)"`
	OutFile     string `flag:"out" help:"Output file path"`
}

var defaultTxExportOptions = txExportOptions{
	Remote:      "gno.land:36657",
	StartHeight: 1,
	EndHeight:   0,
	OutFile:     "txexport.log",
}

func txExportApp(cmd *command.Command, args []string, iopts interface{}) error {
	opts := iopts.(txExportOptions)
	c := client.NewHTTP(opts.Remote, "/websocket")
	status, err := c.Status()
	if err != nil {
		panic(err)
	}
	last := int64(0)
	if opts.EndHeight == 0 {
		last = status.SyncInfo.LatestBlockHeight
	} else {
		last = opts.EndHeight
	}
	out, err := os.OpenFile(opts.OutFile, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}

	for height := int64(opts.StartHeight); height <= last; height++ {
		block, err := c.Block(&height)
		if err != nil {
			panic(err)
		}
		txs := block.Block.Data.Txs
		if len(txs) == 0 {
			continue
		}
		bres, err := c.BlockResults(&height)
		if err != nil {
			// TODO: consider retry for latest height.
			panic(err)
		}
		for i := 0; i < len(txs); i++ {
			if bres.Results.DeliverTxs[i].Error != nil {
				continue
			}
			tx := txs[i]
			stdtx := std.Tx{}
			amino.MustUnmarshal(tx, &stdtx)
			bz := amino.MustMarshalJSON(stdtx)
			fmt.Fprintln(out, string(bz))
		}
	}
	return nil
}
