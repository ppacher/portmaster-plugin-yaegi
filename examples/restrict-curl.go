package main

import (
	"context"
	"log"

	"github.com/safing/portmaster/plugin/shared/proto"
)

func DecideOnConnection(ctx context.Context, conn *proto.Connection) (proto.Verdict, string, error) {
	log.Printf("evaluating %s against curl rule", conn.GetId())

	if conn.GetProcess().GetBinaryPath() == "/usr/bin/curl" {
		switch conn.GetEntity().GetDomain() {
		case "safing.io.":
		case "example.com.":
			return proto.Verdict_VERDICT_ACCEPT, "safing and example are fine", nil
		}

		return proto.Verdict_VERDICT_BLOCK, "curl is restricted to two domains", nil
	}

	return proto.Verdict_VERDICT_UNDECIDED, "", nil
}
